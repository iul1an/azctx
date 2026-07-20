package isolation

import (
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

func TestSetupMissingAzureDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AZURE_CONFIG_DIR", "")
	_ = os.Unsetenv("AZURE_CONFIG_DIR")

	_, err := Setup()
	assert.Error(t, err)
}
