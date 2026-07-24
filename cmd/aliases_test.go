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

func check(body string) error { return validateAliasKeys([]byte(body), "config.yml") }

func TestValidateAliasKeys(t *testing.T) {
	t.Run("accepts distinct aliases", func(t *testing.T) {
		assert.NoError(t, check("aliases:\n  prod: a\n  dev: b\n"))
	})

	t.Run("accepts a config without aliases", func(t *testing.T) {
		assert.NoError(t, check("by-tenant: true\n"))
	})

	t.Run("accepts an empty config", func(t *testing.T) {
		assert.NoError(t, check(""))
	})

	t.Run("rejects keys differing only in case", func(t *testing.T) {
		err := check("aliases:\n  PROD: a\n  Prod: b\n")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `"PROD"`)
		assert.Contains(t, err.Error(), `"Prod"`)
		assert.Contains(t, err.Error(), `"prod"`)
	})

	t.Run("reports every collision, sorted", func(t *testing.T) {
		err := check("aliases:\n  DEV: a\n  dev: b\n  PROD: c\n  prod: d\n")
		require.Error(t, err)
		assert.Regexp(t, `"dev".*"prod"`, err.Error())
	})

	t.Run("reports a malformed file instead of deferring to viper", func(t *testing.T) {
		err := check("aliases: [not, a, map]\n")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "config.yml")
	})
}
