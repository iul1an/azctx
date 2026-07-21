package types

import (
	"fmt"

	"github.com/google/uuid"
)

// BaseManager provides common functionality for tenant and subscription managers
type BaseManager struct {
	Configuration *Configuration
}

// IDGetter is an interface that both Tenant and Subscription implement
type IDGetter interface {
	GetID() uuid.UUID
}

// FindByIDHelper is a utility function to find an item by UUID
func FindByIDHelper[T IDGetter](items []T, id uuid.UUID) (*T, error) {
	for _, item := range items {
		if item.GetID() == id {
			return &item, nil
		}
	}
	return nil, fmt.Errorf("item not found")
}
