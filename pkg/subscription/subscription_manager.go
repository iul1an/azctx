package subscription

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	pkgerrors "github.com/iul1an/azctx/pkg/errors"
	"github.com/iul1an/azctx/pkg/finder"
	"github.com/iul1an/azctx/pkg/types"
)

type Manager struct {
	types.BaseManager
	Aliases Aliases // optional, from the user's config
}

// displayFunc renders a picker entry. Aliases are part of the line, so
// they are fuzzy-matchable.
func (sm *Manager) displayFunc() func(types.Subscription) string {
	idx := sm.AliasIndex()
	return func(s types.Subscription) string {
		if aliases := idx[s.ID]; len(aliases) > 0 {
			return fmt.Sprintf("%s [%s] (%s)", s.Name, strings.Join(aliases, ","), s.ID)
		}
		return fmt.Sprintf("%s (%s)", s.Name, s.ID)
	}
}

// previewFunc renders the preview pane, with an Aliases row when set.
func (sm *Manager) previewFunc() func(types.Subscription) string {
	idx := sm.AliasIndex()
	return func(s types.Subscription) string {
		def := "no"
		if s.IsDefault {
			def = "yes"
		}
		preview := fmt.Sprintf(
			"Name:        %s\nID:          %s\nTenant:      %s\nEnvironment: %s\nState:       %s\nDefault:     %s",
			s.Name, s.ID, s.TenantID, s.EnvironmentName, s.State, def)
		if aliases := idx[s.ID]; len(aliases) > 0 {
			preview += fmt.Sprintf("\nAliases:     %s", strings.Join(aliases, ", "))
		}
		return preview
	}
}

// FindSubscriptionIndex uses fuzzy finding to let user select a subscription
func (sm *Manager) FindSubscriptionIndex() (int, error) {
	if len(sm.Configuration.Subscriptions) == 0 {
		return -1, pkgerrors.ErrSubscriptionNotFound
	}

	sub, err := finder.FuzzyPreview(sm.Configuration.Subscriptions, sm.displayFunc(), sm.previewFunc())
	if err != nil {
		return -1, err
	}

	// Find the index of the selected subscription
	for i, s := range sm.Configuration.Subscriptions {
		if s.ID == sub.ID {
			return i, nil
		}
	}

	return -1, pkgerrors.ErrSubscriptionNotFound
}

// FindSubscriptionByNameOrID finds a subscription by configured alias, by
// UUID, or by case-insensitive exact name, for non-interactive selection.
// An alias wins over a subscription of the same name: it was configured
// deliberately, so an Azure-side rename should not shadow it.
func (sm *Manager) FindSubscriptionByNameOrID(query string) (*types.Subscription, error) {
	if target, ok := sm.Aliases.lookup(query); ok {
		sub, err := sm.lookup(target)
		if err != nil {
			return nil, fmt.Errorf("alias %q points to %q: %w", query, target, err)
		}
		return sub, nil
	}
	return sm.lookup(query)
}

// lookup resolves a UUID or exact name, ignoring the alias map, so alias
// targets never resolve to another alias.
func (sm *Manager) lookup(query string) (*types.Subscription, error) {
	if id, err := uuid.Parse(query); err == nil {
		return sm.FindSubscriptionByID(id)
	}
	for _, sub := range sm.Configuration.Subscriptions {
		if strings.EqualFold(sub.Name, query) {
			return &sub, nil
		}
	}
	return nil, pkgerrors.ErrSubscriptionNotFound
}

// FindSubscriptionByID finds a subscription by its ID
func (sm *Manager) FindSubscriptionByID(id uuid.UUID) (*types.Subscription, error) {
	return finder.ByID(sm.Configuration.Subscriptions, id)
}

// FindSubscriptionsByTenant returns subscriptions filtered by tenant ID
func (sm *Manager) FindSubscriptionsByTenant(tenantID uuid.UUID) ([]types.Subscription, error) {
	var tenantSubs []types.Subscription
	for _, sub := range sm.Configuration.Subscriptions {
		if sub.TenantID == tenantID {
			tenantSubs = append(tenantSubs, sub)
		}
	}
	if len(tenantSubs) == 0 {
		return nil, pkgerrors.ErrSubscriptionNotFound
	}
	return tenantSubs, nil
}

// FindSubscriptionIndexByTenant uses fuzzy finding to select a subscription from a specific tenant
func (sm *Manager) FindSubscriptionIndexByTenant(tenantID uuid.UUID) (*types.Subscription, error) {
	subs, err := sm.FindSubscriptionsByTenant(tenantID)
	if err != nil {
		return nil, err
	}

	return finder.FuzzyPreview(subs, sm.displayFunc(), sm.previewFunc())
}
