package tenant

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	pkgerrors "github.com/iul1an/azctx/pkg/errors"
	"github.com/iul1an/azctx/pkg/finder"
	"github.com/iul1an/azctx/pkg/types"
)

type Manager struct {
	types.BaseManager
	// Labels names tenants for the picker, by tenant ID; optional. The
	// profile carries no tenant names, so this is the only source of one.
	Labels map[uuid.UUID]string
}

// GetTenants retrieves a list of unique tenants from subscriptions.
func (tm *Manager) GetTenants() ([]types.Tenant, error) {
	uniqueTenants := make(map[string]types.Tenant)

	for _, sub := range tm.Configuration.Subscriptions {
		if sub.TenantID != uuid.Nil {
			tenant := types.Tenant{
				ID:   sub.TenantID,
				Name: sub.User.Name,
			}
			// A configured label wins; the profile's customName is a
			// fallback that az itself never writes.
			if label := tm.Labels[sub.TenantID]; label != "" {
				tenant.CustomName = label
			} else {
				for _, t := range tm.Configuration.Tenants {
					if t.ID == sub.TenantID && t.CustomName != "" {
						tenant.CustomName = t.CustomName
						break
					}
				}
			}
			uniqueTenants[sub.TenantID.String()] = tenant
		}
	}

	if len(uniqueTenants) == 0 {
		return nil, pkgerrors.ErrTenantNotFound
	}

	tenants := make([]types.Tenant, 0, len(uniqueTenants))
	for _, tenant := range uniqueTenants {
		tenants = append(tenants, tenant)
	}
	return tenants, nil
}

func tenantDisplay(t types.Tenant) string {
	if t.CustomName != "" {
		return fmt.Sprintf("%s (%s)", t.CustomName, t.ID)
	}
	return fmt.Sprintf("%s (%s)", t.Name, t.ID)
}

// previewFunc renders a tenant's subscriptions, '*' marking the default.
// The profile carries no tenant names, so what a tenant contains is the only
// way to recognize it.
func (tm *Manager) previewFunc() func(types.Tenant) string {
	return func(t types.Tenant) string {
		var subs []types.Subscription
		for _, s := range tm.Configuration.Subscriptions {
			if s.TenantID == t.ID {
				subs = append(subs, s)
			}
		}
		sort.Slice(subs, func(i, j int) bool { return subs[i].Name < subs[j].Name })
		lines := make([]string, 0, len(subs))
		for _, s := range subs {
			marker := " "
			if s.IsDefault {
				marker = "*"
			}
			lines = append(lines, fmt.Sprintf("%s %s", marker, s.Name))
		}
		preview := fmt.Sprintf("Tenant:        %s\nSubscriptions: %d", t.ID, len(subs))
		if len(lines) > 0 {
			preview += "\n\n" + strings.Join(lines, "\n")
		}
		return preview
	}
}

// FindTenantIndex uses fuzzy finding to let user select a tenant
func (tm *Manager) FindTenantIndex() (*types.Tenant, error) {
	tenants, err := tm.GetTenants()
	if err != nil {
		return nil, fmt.Errorf("failed to get tenants: %w", err)
	}

	return finder.FuzzyPreview(tenants, tenantDisplay, tm.previewFunc())
}
