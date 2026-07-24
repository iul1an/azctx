package isolation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsActive(t *testing.T) {
	t.Run("false when AZURE_CONFIG_DIR is unset", func(t *testing.T) {
		t.Setenv("AZURE_CONFIG_DIR", "")
		_ = os.Unsetenv("AZURE_CONFIG_DIR")
		assert.False(t, IsActive())
	})

	t.Run("false for a non-azctx config dir", func(t *testing.T) {
		t.Setenv("AZURE_CONFIG_DIR", "/some/other/dir")
		assert.False(t, IsActive())
	})

	t.Run("true for an azctx tempdir", func(t *testing.T) {
		t.Setenv("AZURE_CONFIG_DIR", filepath.Join(os.TempDir(), "azctx.123456"))
		assert.True(t, IsActive())
	})
}

func TestSetup(t *testing.T) {
	// Fake home with a .azure directory.
	home := t.TempDir()
	t.Setenv("HOME", home)
	azureDir := filepath.Join(home, ".azure")
	require.NoError(t, os.MkdirAll(filepath.Join(azureDir, "msal_token_cache"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(azureDir, "azureProfile.json"), []byte(`{"subscriptions":[]}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(azureDir, "msal_token_cache", "cache.json"), []byte(`{}`), 0o600))

	t.Setenv("AZURE_CONFIG_DIR", "")
	_ = os.Unsetenv("AZURE_CONFIG_DIR")

	tmpDir, err := Setup()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	// Tempdir matches the detection pattern and is now the active config dir.
	assert.Equal(t, tmpDir, os.Getenv("AZURE_CONFIG_DIR"))
	assert.True(t, IsActive())

	// Contents were copied recursively.
	data, err := os.ReadFile(filepath.Join(tmpDir, "azureProfile.json"))
	require.NoError(t, err)
	assert.Equal(t, `{"subscriptions":[]}`, string(data))
	_, err = os.Stat(filepath.Join(tmpDir, "msal_token_cache", "cache.json"))
	assert.NoError(t, err)

	// Tempdir is private to the user.
	info, err := os.Stat(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
}

func TestSetupPreservesFileModes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	azureDir := filepath.Join(home, ".azure")
	sub := filepath.Join(azureDir, "msal_token_cache")
	require.NoError(t, os.MkdirAll(sub, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(azureDir, "service_principal_entries.json"), []byte(`[]`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(azureDir, "config"), []byte("[core]\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "cache.json"), []byte(`{}`), 0o600))
	// WriteFile is subject to umask; force the modes the test asserts on.
	require.NoError(t, os.Chmod(filepath.Join(azureDir, "config"), 0o644))
	require.NoError(t, os.Chmod(sub, 0o700))

	t.Setenv("AZURE_CONFIG_DIR", "")
	_ = os.Unsetenv("AZURE_CONFIG_DIR")

	tmpDir, err := Setup()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	for path, want := range map[string]os.FileMode{
		"service_principal_entries.json": 0o600,
		"config":                         0o644,
		"msal_token_cache":               0o700,
		"msal_token_cache/cache.json":    0o600,
	} {
		info, err := os.Stat(filepath.Join(tmpDir, path))
		require.NoError(t, err, path)
		assert.Equal(t, want, info.Mode().Perm(), "mode of %s", path)
	}
}

func TestSetupSkipsDiagnosticDirs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	azureDir := filepath.Join(home, ".azure")
	require.NoError(t, os.MkdirAll(filepath.Join(azureDir, "logs", "2026"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(azureDir, "commands"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(azureDir, "logs", "2026", "az.log"), []byte("noise"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(azureDir, "commands", "cmd.json"), []byte("{}"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(azureDir, "azureProfile.json"), []byte(`{}`), 0o600))

	t.Setenv("AZURE_CONFIG_DIR", "")
	_ = os.Unsetenv("AZURE_CONFIG_DIR")

	tmpDir, err := Setup()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	assert.NoFileExists(t, filepath.Join(tmpDir, "logs", "2026", "az.log"))
	assert.NoDirExists(t, filepath.Join(tmpDir, "logs"))
	assert.NoDirExists(t, filepath.Join(tmpDir, "commands"))
	assert.FileExists(t, filepath.Join(tmpDir, "azureProfile.json"))
}

func TestSetupFollowsSymlinkedFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	azureDir := filepath.Join(home, ".azure")
	require.NoError(t, os.MkdirAll(azureDir, 0o700))
	// A profile symlinked into a dotfiles repo is a plausible setup.
	target := filepath.Join(home, "dotfiles-azureProfile.json")
	require.NoError(t, os.WriteFile(target, []byte(`{"subscriptions":[]}`), 0o600))
	require.NoError(t, os.Symlink(target, filepath.Join(azureDir, "azureProfile.json")))

	t.Setenv("AZURE_CONFIG_DIR", "")
	_ = os.Unsetenv("AZURE_CONFIG_DIR")

	tmpDir, err := Setup()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	copied := filepath.Join(tmpDir, "azureProfile.json")
	data, err := os.ReadFile(copied)
	require.NoError(t, err)
	assert.Equal(t, `{"subscriptions":[]}`, string(data))

	// The copy is a regular file, so writing to it cannot reach the original.
	info, err := os.Lstat(copied)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0), info.Mode()&os.ModeSymlink)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestSetupCopiesSymlinkedDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	azureDir := filepath.Join(home, ".azure")
	require.NoError(t, os.MkdirAll(azureDir, 0o700))
	shared := filepath.Join(home, "shared-extensions")
	require.NoError(t, os.MkdirAll(shared, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(shared, "ext.json"), []byte(`{}`), 0o600))
	require.NoError(t, os.Symlink(shared, filepath.Join(azureDir, "cliextensions")))
	// A loop must not hang the copy.
	require.NoError(t, os.Symlink(azureDir, filepath.Join(shared, "loop")))

	t.Setenv("AZURE_CONFIG_DIR", "")
	_ = os.Unsetenv("AZURE_CONFIG_DIR")

	tmpDir, err := Setup()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	copied := filepath.Join(tmpDir, "cliextensions", "ext.json")
	assert.FileExists(t, copied)
	info, err := os.Lstat(filepath.Join(tmpDir, "cliextensions"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0), info.Mode()&os.ModeSymlink, "copied as a real dir")
}

func TestSpawnShellInheritsConfigDir(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "out")
	shellScript := filepath.Join(dir, "fakeshell")
	require.NoError(t, os.WriteFile(shellScript, []byte("#!/bin/sh\nprintf '%s' \"$AZURE_CONFIG_DIR\" > "+outFile+"\n"), 0o755))

	t.Setenv("SHELL", shellScript)
	t.Setenv("AZURE_CONFIG_DIR", "/isolated/config/dir")

	require.NoError(t, SpawnShell())

	data, err := os.ReadFile(outFile)
	require.NoError(t, err)
	assert.Equal(t, "/isolated/config/dir", string(data))
}

func TestRunCommand(t *testing.T) {
	t.Run("inherits AZURE_CONFIG_DIR and returns exit code 0", func(t *testing.T) {
		dir := t.TempDir()
		outFile := filepath.Join(dir, "out")
		t.Setenv("AZURE_CONFIG_DIR", "/isolated/config/dir")

		code, err := RunCommand([]string{"/bin/sh", "-c", "printf '%s' \"$AZURE_CONFIG_DIR\" > " + outFile})
		require.NoError(t, err)
		assert.Equal(t, 0, code)

		data, err := os.ReadFile(outFile)
		require.NoError(t, err)
		assert.Equal(t, "/isolated/config/dir", string(data))
	})

	t.Run("propagates non-zero exit code without error", func(t *testing.T) {
		code, err := RunCommand([]string{"/bin/sh", "-c", "exit 3"})
		require.NoError(t, err)
		assert.Equal(t, 3, code)
	})

	t.Run("errors on unrunnable command", func(t *testing.T) {
		_, err := RunCommand([]string{"/nonexistent/binary"})
		assert.Error(t, err)
	})
}

// deadPID is above linux pid_max (4194304), so it can never be alive.
const deadPID = 99999999

func makeFakeContext(t *testing.T, pid int, withMeta bool, sub string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "azctx.*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	if withMeta {
		require.NoError(t, os.WriteFile(filepath.Join(dir, metaFileName),
			[]byte(fmt.Sprintf(`{"pid": %d, "started": "2026-07-21T10:00:00Z"}`, pid)), 0o600))
	}
	profile := `{"subscriptions": [{"name": "` + sub + `", "isDefault": true}, {"name": "other", "isDefault": false}]}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "azureProfile.json"), []byte(profile), 0o600))
	return dir
}

func TestListContexts(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	t.Setenv("AZURE_CONFIG_DIR", "")
	_ = os.Unsetenv("AZURE_CONFIG_DIR")

	live := makeFakeContext(t, os.Getpid(), true, "sub-live")
	dead := makeFakeContext(t, deadPID, true, "sub-dead")
	noMeta := makeFakeContext(t, 0, false, "sub-alien")
	t.Setenv("AZURE_CONFIG_DIR", live)

	ctxs, err := ListContexts()
	require.NoError(t, err)
	require.Len(t, ctxs, 2, "context without meta must be ignored")

	byDir := map[string]Context{}
	for _, c := range ctxs {
		byDir[c.Dir] = c
	}
	require.NotContains(t, byDir, noMeta)
	assert.True(t, byDir[live].Alive)
	assert.True(t, byDir[live].Active)
	assert.Equal(t, "sub-live", byDir[live].Subscription)
	assert.False(t, byDir[dead].Alive)
	assert.False(t, byDir[dead].Active)
	assert.Equal(t, "sub-dead", byDir[dead].Subscription)
}

func TestSweep(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	t.Setenv("AZURE_CONFIG_DIR", "")
	_ = os.Unsetenv("AZURE_CONFIG_DIR")

	live := makeFakeContext(t, os.Getpid(), true, "sub-live")
	dead := makeFakeContext(t, deadPID, true, "sub-dead")
	noMeta := makeFakeContext(t, 0, false, "sub-alien")

	n := Sweep()
	assert.Equal(t, 1, n)

	_, err := os.Stat(dead)
	assert.True(t, os.IsNotExist(err), "dead context must be removed")
	_, err = os.Stat(live)
	assert.NoError(t, err, "live context must be kept")
	_, err = os.Stat(noMeta)
	assert.NoError(t, err, "meta-less dir must never be touched")
}

func TestSweepKeepsActiveContext(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	dead := makeFakeContext(t, deadPID, true, "sub-dead")
	t.Setenv("AZURE_CONFIG_DIR", dead)

	assert.Equal(t, 0, Sweep())
	_, err := os.Stat(dead)
	assert.NoError(t, err, "the active context must never be swept, even with a dead pid")
}

func TestSetupWritesMeta(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".azure"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(home, ".azure", "azureProfile.json"), []byte(`{"subscriptions":[]}`), 0o600))
	t.Setenv("AZURE_CONFIG_DIR", "")
	_ = os.Unsetenv("AZURE_CONFIG_DIR")

	tmpDir, err := Setup()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	data, err := os.ReadFile(filepath.Join(tmpDir, metaFileName))
	require.NoError(t, err)
	var m meta
	require.NoError(t, json.Unmarshal(data, &m))
	assert.Equal(t, os.Getpid(), m.PID)
	assert.False(t, m.Started.IsZero())
}

func TestSetupEmpty(t *testing.T) {
	// No ~/.azure needed at all.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AZURE_CONFIG_DIR", "")
	_ = os.Unsetenv("AZURE_CONFIG_DIR")

	tmpDir, err := SetupEmpty()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	assert.Equal(t, tmpDir, os.Getenv("AZURE_CONFIG_DIR"))
	assert.True(t, IsActive())

	// Only the meta marker inside; nothing was copied.
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, metaFileName, entries[0].Name())
}

func TestSetupMissingAzureDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AZURE_CONFIG_DIR", "")
	_ = os.Unsetenv("AZURE_CONFIG_DIR")

	_, err := Setup()
	assert.Error(t, err)
}
