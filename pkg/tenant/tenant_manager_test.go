package tenant

import (
	"encoding/json"
	"testing"

	"github.com/iul1an/azctx/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testManager(t *testing.T) *Manager {
	t.Helper()
	var cfg types.Configuration
	require.NoError(t, json.Unmarshal([]byte(`{
		"installationId": "11111111-1111-1111-1111-111111111111",
		"subscriptions": [
			{"id": "22222222-2222-2222-2222-222222222222", "name": "sub-beta", "state": "Enabled",
			 "user": {"name": "u", "type": "user"}, "tenantId": "33333333-3333-3333-3333-333333333333", "isDefault": true},
			{"id": "44444444-4444-4444-4444-444444444444", "name": "sub-alpha", "state": "Enabled",
			 "user": {"name": "u", "type": "user"}, "tenantId": "33333333-3333-3333-3333-333333333333", "isDefault": false},
			{"id": "66666666-6666-6666-6666-666666666666", "name": "sub-gamma", "state": "Enabled",
			 "user": {"name": "u", "type": "user"}, "tenantId": "55555555-5555-5555-5555-555555555555", "isDefault": false}
		]
	}`), &cfg))
	return &Manager{BaseManager: types.BaseManager{Configuration: &cfg}}
}

func TestTenantPreview(t *testing.T) {
	tm := testManager(t)
	tenants, err := tm.GetTenants()
	require.NoError(t, err)
	preview := tm.previewFunc()

	byID := map[string]types.Tenant{}
	for _, tn := range tenants {
		byID[tn.ID.String()] = tn
	}

	t.Run("lists the tenant's subscriptions, default marked, sorted", func(t *testing.T) {
		got := preview(byID["33333333-3333-3333-3333-333333333333"])
		assert.Contains(t, got, "Tenant:        33333333-3333-3333-3333-333333333333")
		assert.Contains(t, got, "Subscriptions: 2")
		assert.Contains(t, got, "* sub-beta")
		assert.Contains(t, got, "  sub-alpha")
		assert.Less(t, indexOf(got, "  sub-alpha"), indexOf(got, "* sub-beta"),
			"sorted by name, not by the default marker")
	})

	t.Run("does not leak other tenants' subscriptions", func(t *testing.T) {
		got := preview(byID["55555555-5555-5555-5555-555555555555"])
		assert.Contains(t, got, "Subscriptions: 1")
		assert.Contains(t, got, "sub-gamma")
		assert.NotContains(t, got, "sub-alpha")
	})
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
