package tenant

import (
	"fmt"

	"github.com/google/uuid"
	pkgerrors "github.com/iul1an/azctx/pkg/errors"
	"github.com/iul1an/azctx/pkg/finder"
	"github.com/iul1an/azctx/pkg/types"
)

type Manager struct {
	types.BaseManager
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
			// Check if we have a custom name for this tenant
			for _, t := range tm.Configuration.Tenants {
				if t.ID == sub.TenantID && t.CustomName != "" {
					tenant.CustomName = t.CustomName
					break
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

// FindTenantIndex uses fuzzy finding to let user select a tenant
func (tm *Manager) FindTenantIndex() (*types.Tenant, error) {
	tenants, err := tm.GetTenants()
	if err != nil {
		return nil, fmt.Errorf("failed to get tenants: %w", err)
	}

	return finder.Fuzzy(tenants, func(t types.Tenant) string {
		if t.CustomName != "" {
			return fmt.Sprintf("%s (%s)", t.CustomName, t.ID)
		}
		return fmt.Sprintf("%s (%s)", t.Name, t.ID)
	})
}
