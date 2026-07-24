package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".azctx.yml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	return path
}

func TestValidateAliasKeys(t *testing.T) {
	t.Run("accepts distinct aliases", func(t *testing.T) {
		assert.NoError(t, validateAliasKeys(writeConfig(t, "aliases:\n  prod: a\n  dev: b\n")))
	})

	t.Run("accepts a config without aliases", func(t *testing.T) {
		assert.NoError(t, validateAliasKeys(writeConfig(t, "by-tenant: true\n")))
	})

	t.Run("accepts no config file", func(t *testing.T) {
		assert.NoError(t, validateAliasKeys(""))
	})

	t.Run("rejects keys differing only in case", func(t *testing.T) {
		err := validateAliasKeys(writeConfig(t, "aliases:\n  PROD: a\n  Prod: b\n"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), `"PROD"`)
		assert.Contains(t, err.Error(), `"Prod"`)
		assert.Contains(t, err.Error(), `"prod"`)
	})

	t.Run("reports every collision, sorted", func(t *testing.T) {
		err := validateAliasKeys(writeConfig(t,
			"aliases:\n  DEV: a\n  dev: b\n  PROD: c\n  prod: d\n"))
		require.Error(t, err)
		assert.Regexp(t, `"dev".*"prod"`, err.Error())
	})

	t.Run("ignores a file it cannot parse", func(t *testing.T) {
		// viper reports malformed config first; do not double-report here.
		assert.NoError(t, validateAliasKeys(writeConfig(t, "aliases: [not, a, map]\n")))
	})
}
