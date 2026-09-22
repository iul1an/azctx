package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadConfig drives package-level viper state, so these cannot run in
// parallel with anything that reads it.
func withHome(t *testing.T, configBody string) string {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(configFileEnv, "")
	_ = os.Unsetenv(configFileEnv)
	path := filepath.Join(home, ".azctx.yml")
	if configBody != "" {
		require.NoError(t, os.WriteFile(path, []byte(configBody), 0o600))
	}
	return path
}

func TestLoadConfig(t *testing.T) {
	t.Run("reads the default path", func(t *testing.T) {
		withHome(t, "by-tenant: true\nlog-level: debug\n")
		require.NoError(t, loadConfig())
		assert.True(t, viper.GetBool("by-tenant"))
		assert.Equal(t, "debug", viper.GetString("log-level"))
	})

	t.Run("a missing default config is not an error", func(t *testing.T) {
		withHome(t, "")
		require.NoError(t, loadConfig())
		assert.False(t, viper.GetBool("by-tenant"))
	})

	t.Run("AZCTX_CONFIG_FILE replaces the default", func(t *testing.T) {
		withHome(t, "subscription: from-home\n")
		other := filepath.Join(t.TempDir(), "other.conf")
		require.NoError(t, os.WriteFile(other, []byte("subscription: from-env\n"), 0o600))
		t.Setenv(configFileEnv, other)

		require.NoError(t, loadConfig())
		assert.Equal(t, "from-env", viper.GetString("subscription"))
	})

	t.Run("a missing explicit config is an error", func(t *testing.T) {
		withHome(t, "")
		t.Setenv(configFileEnv, filepath.Join(t.TempDir(), "nope.yml"))
		assert.Error(t, loadConfig())
	})

	t.Run("an unreadable config is an error", func(t *testing.T) {
		path := withHome(t, "by-tenant: true\n")
		require.NoError(t, os.Chmod(path, 0o000))
		t.Cleanup(func() { _ = os.Chmod(path, 0o600) })
		if os.Geteuid() == 0 {
			t.Skip("root reads anything")
		}
		assert.ErrorContains(t, loadConfig(), "reading config")
	})

	t.Run("a malformed config is an error", func(t *testing.T) {
		withHome(t, "aliases: [not, a, map]\n")
		assert.Error(t, loadConfig())
	})

	t.Run("case-colliding aliases are an error", func(t *testing.T) {
		withHome(t, "aliases:\n  PROD: a\n  Prod: b\n")
		assert.ErrorContains(t, loadConfig(), "invalid config")
	})

	t.Run("the environment overrides the config file", func(t *testing.T) {
		withHome(t, "subscription: from-file\n")
		t.Setenv("AZCTX_SUBSCRIPTION", "from-env")
		require.NoError(t, loadConfig())
		assert.Equal(t, "from-env", viper.GetString("subscription"))
	})
}
