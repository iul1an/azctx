package subscription

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAliases(t *testing.T) {
	assert.Nil(t, NewAliases(nil))
	assert.Nil(t, NewAliases(map[string]string{"": "sub", "drop": ""}))

	a := NewAliases(map[string]string{
		"  PRD  ":   "  sub-a  ",
		"a\tb":      "sub-b",
		"c\nd":      "sub-c",
		"two words": "sub-d",
	})
	assert.Equal(t, Aliases{"prd": "sub-a", "two words": "sub-d"}, a,
		"keys are lowercased and trimmed; tabs and newlines are dropped, spaces kept")
}

func TestFindSubscriptionByAlias(t *testing.T) {
	sm := lookupTestManager(t)
	sm.Aliases = Aliases{
		"prd":      "44444444-4444-4444-4444-444444444444",
		"sbx":      "Azure subscription Sandbox",
		"dangling": "no such subscription",
		// An alias that shadows a real subscription name; the alias wins.
		"azure subscription prd": "22222222-2222-2222-2222-222222222222",
	}

	t.Run("resolves an alias pointing at an ID", func(t *testing.T) {
		sub, err := sm.FindSubscriptionByNameOrID("prd")
		require.NoError(t, err)
		assert.Equal(t, "Azure Subscription PRD", sub.Name)
	})

	t.Run("resolves an alias pointing at a name", func(t *testing.T) {
		sub, err := sm.FindSubscriptionByNameOrID("sbx")
		require.NoError(t, err)
		assert.Equal(t, "Azure subscription Sandbox", sub.Name)
	})

	t.Run("matches the alias case-insensitively", func(t *testing.T) {
		sub, err := sm.FindSubscriptionByNameOrID("PRD")
		require.NoError(t, err)
		assert.Equal(t, "Azure Subscription PRD", sub.Name)
	})

	t.Run("alias wins over a subscription of the same name", func(t *testing.T) {
		sub, err := sm.FindSubscriptionByNameOrID("Azure Subscription PRD")
		require.NoError(t, err)
		assert.Equal(t, "Azure subscription Sandbox", sub.Name)
	})

	t.Run("a dangling alias errors instead of falling through", func(t *testing.T) {
		_, err := sm.FindSubscriptionByNameOrID("dangling")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `alias "dangling"`)
		assert.Contains(t, err.Error(), "no such subscription")
	})

	t.Run("unaliased queries still resolve normally", func(t *testing.T) {
		sub, err := sm.FindSubscriptionByNameOrID("22222222-2222-2222-2222-222222222222")
		require.NoError(t, err)
		assert.Equal(t, "Azure subscription Sandbox", sub.Name)
	})

	t.Run("aliases are not resolved recursively", func(t *testing.T) {
		sm := lookupTestManager(t)
		sm.Aliases = Aliases{"a": "b", "b": "Azure Subscription PRD"}
		_, err := sm.FindSubscriptionByNameOrID("a")
		assert.Error(t, err)
	})
}

func TestAliasIndex(t *testing.T) {
	sm := lookupTestManager(t)
	sm.Aliases = Aliases{
		"prd":     "44444444-4444-4444-4444-444444444444",
		"p":       "Azure Subscription PRD",
		"sbx":     "Azure subscription Sandbox",
		"missing": "gone",
	}

	idx := sm.AliasIndex()
	assert.Equal(t, []string{"p", "prd"}, idx[sm.Configuration.Subscriptions[1].ID])
	assert.Equal(t, []string{"sbx"}, idx[sm.Configuration.Subscriptions[0].ID])
	assert.Len(t, idx, 2, "dangling aliases are not indexed")
}

func TestDanglingAliases(t *testing.T) {
	sm := lookupTestManager(t)
	sm.Aliases = Aliases{
		"prd":  "44444444-4444-4444-4444-444444444444",
		"gone": "no such subscription",
		"also": "55555555-5555-5555-5555-555555555555",
	}
	assert.Equal(t, []string{"also", "gone"}, sm.DanglingAliases())
}

func TestAliasesFor(t *testing.T) {
	sm := lookupTestManager(t)
	sm.Aliases = Aliases{"prd": "44444444-4444-4444-4444-444444444444"}

	assert.Equal(t, []string{"prd"}, sm.AliasesFor(sm.Configuration.Subscriptions[1].ID))
	assert.Empty(t, sm.AliasesFor(sm.Configuration.Subscriptions[0].ID))
}

func TestSubscriptionDisplayWithAliases(t *testing.T) {
	sm := lookupTestManager(t)
	sm.Aliases = Aliases{"prd": "44444444-4444-4444-4444-444444444444", "p": "Azure Subscription PRD"}
	display := sm.displayFunc()

	assert.Equal(t,
		"Azure Subscription PRD [p,prd] (44444444-4444-4444-4444-444444444444)",
		display(sm.Configuration.Subscriptions[1]))
	assert.Equal(t,
		"Azure subscription Sandbox (22222222-2222-2222-2222-222222222222)",
		display(sm.Configuration.Subscriptions[0]))
}

func TestSubscriptionPreviewWithAliases(t *testing.T) {
	sm := lookupTestManager(t)
	sm.Aliases = Aliases{"prd": "44444444-4444-4444-4444-444444444444"}
	preview := sm.previewFunc()

	assert.Contains(t, preview(sm.Configuration.Subscriptions[1]), "Aliases:     prd")
	assert.NotContains(t, preview(sm.Configuration.Subscriptions[0]), "Aliases:")
}
