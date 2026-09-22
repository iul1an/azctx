package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveConfigFile(t *testing.T) {
	t.Run("unset means the default search path", func(t *testing.T) {
		t.Setenv(configFileEnv, "")
		path, err := resolveConfigFile()
		require.NoError(t, err)
		assert.Empty(t, path)
	})

	t.Run("returns an existing file", func(t *testing.T) {
		cfg := writeConfig(t, "by-tenant: true\n")
		t.Setenv(configFileEnv, cfg)
		path, err := resolveConfigFile()
		require.NoError(t, err)
		assert.Equal(t, cfg, path)
	})

	t.Run("errors on a missing file", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "nope.yml")
		t.Setenv(configFileEnv, missing)
		_, err := resolveConfigFile()
		require.Error(t, err)
		assert.Contains(t, err.Error(), missing)
	})

	t.Run("errors on a directory", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv(configFileEnv, dir)
		_, err := resolveConfigFile()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is a directory")
	})

	t.Run("any extension parses as yaml", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "azctx.conf")
		require.NoError(t, os.WriteFile(path, []byte("by-tenant: true\n"), 0o600))
		t.Setenv(configFileEnv, path)
		got, err := resolveConfigFile()
		require.NoError(t, err)
		assert.Equal(t, path, got)
	})
}
