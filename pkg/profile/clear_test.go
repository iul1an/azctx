package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/riweston/aztx/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClearContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "azureProfile.json")
	require.NoError(t, os.WriteFile(path, []byte(`{
		"installationId": "11111111-1111-1111-1111-111111111111",
		"subscriptions": [
			{"id": "22222222-2222-2222-2222-222222222222", "name": "sub-a", "state": "Enabled",
			 "user": {"name": "u", "type": "user"}, "tenantId": "33333333-3333-3333-3333-333333333333",
			 "isDefault": true, "environmentName": "AzureCloud", "tenantDefaultDomain": "example.onmicrosoft.com"},
			{"id": "44444444-4444-4444-4444-444444444444", "name": "sub-b", "state": "Enabled",
			 "user": {"name": "u", "type": "user"}, "tenantId": "33333333-3333-3333-3333-333333333333",
			 "isDefault": false}
		]
	}`), 0o600))

	fa := &storage.FileAdapter{Path: path}
	adapter := NewConfigurationAdapter(fa, NewLogger("error"))
	require.NoError(t, adapter.ClearContext())

	var m map[string]any
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &m))

	subs := m["subscriptions"].([]any)
	require.Len(t, subs, 2)
	for _, s := range subs {
		assert.Equal(t, false, s.(map[string]any)["isDefault"])
	}
	// Unknown fields survive the rewrite.
	assert.Equal(t, "example.onmicrosoft.com", subs[0].(map[string]any)["tenantDefaultDomain"])
}
