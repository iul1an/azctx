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

	t.Run("false for a non-aztx config dir", func(t *testing.T) {
		t.Setenv("AZURE_CONFIG_DIR", "/some/other/dir")
		assert.False(t, IsActive())
	})

	t.Run("true for an aztx tempdir", func(t *testing.T) {
		t.Setenv("AZURE_CONFIG_DIR", filepath.Join(os.TempDir(), "aztx.123456"))
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
	dir, err := os.MkdirTemp("", "aztx.*")
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

func TestSetupMissingAzureDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AZURE_CONFIG_DIR", "")
	_ = os.Unsetenv("AZURE_CONFIG_DIR")

	_, err := Setup()
	assert.Error(t, err)
}
