package subscription

import (
	"encoding/json"
	"testing"

	"github.com/riweston/aztx/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func lookupTestManager(t *testing.T) *Manager {
	t.Helper()
	var cfg types.Configuration
	require.NoError(t, json.Unmarshal([]byte(`{
		"installationId": "11111111-1111-1111-1111-111111111111",
		"subscriptions": [
			{"id": "22222222-2222-2222-2222-222222222222", "name": "Azure subscription Sandbox", "state": "Enabled",
			 "user": {"name": "u", "type": "user"}, "tenantId": "33333333-3333-3333-3333-333333333333", "isDefault": true},
			{"id": "44444444-4444-4444-4444-444444444444", "name": "Azure Subscription PRD", "state": "Enabled",
			 "user": {"name": "u", "type": "user"}, "tenantId": "33333333-3333-3333-3333-333333333333", "isDefault": false}
		]
	}`), &cfg))
	return &Manager{BaseManager: types.BaseManager{Configuration: &cfg}}
}

func TestFindSubscriptionByNameOrID(t *testing.T) {
	sm := lookupTestManager(t)

	t.Run("finds by ID", func(t *testing.T) {
		sub, err := sm.FindSubscriptionByNameOrID("44444444-4444-4444-4444-444444444444")
		require.NoError(t, err)
		assert.Equal(t, "Azure Subscription PRD", sub.Name)
	})

	t.Run("finds by exact name", func(t *testing.T) {
		sub, err := sm.FindSubscriptionByNameOrID("Azure Subscription PRD")
		require.NoError(t, err)
		assert.Equal(t, "Azure Subscription PRD", sub.Name)
	})

	t.Run("finds by name case-insensitively", func(t *testing.T) {
		sub, err := sm.FindSubscriptionByNameOrID("azure subscription prd")
		require.NoError(t, err)
		assert.Equal(t, "Azure Subscription PRD", sub.Name)
	})

	t.Run("errors when not found", func(t *testing.T) {
		_, err := sm.FindSubscriptionByNameOrID("nope")
		assert.Error(t, err)
	})
}
