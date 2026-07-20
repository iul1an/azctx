package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Regression test for upstream issue #36: aztx must not drop fields it
// doesn't know about (environmentName, tenantDefaultDomain, ...) when it
// rewrites azureProfile.json, or the Azure CLI breaks.
func TestConfigurationRoundTripPreservesUnknownFields(t *testing.T) {
	src := []byte(`{
		"installationId": "11111111-1111-1111-1111-111111111111",
		"someFutureTopLevelField": {"nested": true},
		"subscriptions": [
			{
				"id": "22222222-2222-2222-2222-222222222222",
				"name": "sub-a",
				"state": "Enabled",
				"user": {"name": "user@example.com", "type": "user"},
				"isDefault": true,
				"tenantId": "33333333-3333-3333-3333-333333333333",
				"environmentName": "AzureCloud",
				"homeTenantId": "33333333-3333-3333-3333-333333333333",
				"tenantDefaultDomain": "example.onmicrosoft.com",
				"tenantDisplayName": "Example Tenant",
				"managedByTenants": [],
				"someFutureSubField": "keep-me"
			},
			{
				"id": "44444444-4444-4444-4444-444444444444",
				"name": "sub-b",
				"state": "Enabled",
				"user": {"name": "user@example.com", "type": "user"},
				"isDefault": false,
				"tenantId": "33333333-3333-3333-3333-333333333333",
				"environmentName": "AzureCloud",
				"tenantDefaultDomain": "example.onmicrosoft.com"
			}
		]
	}`)

	var cfg Configuration
	require.NoError(t, json.Unmarshal(src, &cfg))
	require.Len(t, cfg.Subscriptions, 2)

	// Mutate like SetContext does: flip the default subscription.
	cfg.Subscriptions[0].IsDefault = false
	cfg.Subscriptions[1].IsDefault = true

	out, err := json.Marshal(&cfg)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))

	// Unknown top-level fields survive.
	assert.Equal(t, map[string]any{"nested": true}, m["someFutureTopLevelField"])
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", m["installationId"])

	subs := m["subscriptions"].([]any)
	require.Len(t, subs, 2)
	subA := subs[0].(map[string]any)
	subB := subs[1].(map[string]any)

	// Unknown subscription fields survive.
	assert.Equal(t, "keep-me", subA["someFutureSubField"])
	assert.Equal(t, "example.onmicrosoft.com", subA["tenantDefaultDomain"])
	assert.Equal(t, "Example Tenant", subA["tenantDisplayName"])
	assert.Equal(t, "AzureCloud", subA["environmentName"])
	assert.Equal(t, "example.onmicrosoft.com", subB["tenantDefaultDomain"])

	// The mutation is reflected.
	assert.Equal(t, false, subA["isDefault"])
	assert.Equal(t, true, subB["isDefault"])
}

func TestConfigurationMarshalWithoutPriorUnmarshal(t *testing.T) {
	// Structs built directly (no raw JSON captured) must still marshal.
	var cfg Configuration
	require.NoError(t, json.Unmarshal([]byte(`{"installationId":"11111111-1111-1111-1111-111111111111","subscriptions":[]}`), &cfg))
	out, err := json.Marshal(&cfg)
	require.NoError(t, err)
	assert.Contains(t, string(out), "11111111-1111-1111-1111-111111111111")
}
