package profile

import (
	"errors"

	"github.com/google/uuid"
	pkgerrors "github.com/iul1an/azctx/pkg/errors"
	"github.com/iul1an/azctx/pkg/finder"
	"github.com/iul1an/azctx/pkg/subscription"
	"github.com/iul1an/azctx/pkg/types"
)

type ConfigurationAdapter struct {
	storage StorageAdapter
	logger  Logger
}

func NewConfigurationAdapter(storage StorageAdapter, logger Logger) *ConfigurationAdapter {
	return &ConfigurationAdapter{
		storage: storage,
		logger:  logger,
	}
}

func (c *ConfigurationAdapter) SelectWithFinder() (*types.Subscription, error) {
	if c.storage == nil {
		c.logger.Error("storage adapter is nil")
		return nil, pkgerrors.ErrEmptyConfiguration
	}

	c.logger.Debug("reading azure profile configuration")
	config, err := c.storage.ReadConfig()
	if err != nil {
		c.logger.Error("failed to read configuration: %v", err)
		return nil, pkgerrors.WrapError("reading configuration", err)
	}

	if len(config.Subscriptions) == 0 {
		c.logger.Warn("no subscriptions found in configuration")
		return nil, pkgerrors.ErrEmptyConfiguration
	}

	c.logger.Debug("initiating subscription selection with fuzzy finder")
	subManager := subscription.Manager{BaseManager: types.BaseManager{Configuration: config}}
	idx, err := subManager.FindSubscriptionIndex()
	if err != nil {
		if errors.Is(err, finder.ErrAbort) {
			return nil, err
		}
		c.logger.Error("failed to get subscription selection: %v", err)
		return nil, pkgerrors.WrapError("finding subscription", err)
	}

	if idx < 0 || idx >= len(config.Subscriptions) {
		c.logger.Error("selected subscription index %d is out of bounds", idx)
		return nil, pkgerrors.ErrSubscriptionNotFound
	}

	selected := &config.Subscriptions[idx]
	return selected, nil
}

func (c *ConfigurationAdapter) SetContext(subscriptionID uuid.UUID) error {
	if subscriptionID == uuid.Nil {
		c.logger.Error("invalid subscription ID provided")
		return pkgerrors.ErrInvalidSubscriptionID
	}

	c.logger.Debug("reading configuration to update context")
	config, err := c.storage.ReadConfig()
	if err != nil {
		c.logger.Error("failed to read configuration: %v", err)
		return pkgerrors.WrapError("reading configuration", err)
	}

	// First verify the target subscription exists
	var targetIndex = -1
	for i, sub := range config.Subscriptions {
		if sub.ID == subscriptionID {
			targetIndex = i
			break
		}
	}

	if targetIndex == -1 {
		c.logger.Error("subscription %s not found in configuration", subscriptionID)
		return pkgerrors.ErrSubscriptionNotFound
	}

	// Now that we know the target exists, safely update the default flags
	for i := range config.Subscriptions {
		if config.Subscriptions[i].IsDefault {
			c.logger.Debug("clearing default from subscription: %s", config.Subscriptions[i].Name)
			config.Subscriptions[i].IsDefault = false
		}
	}

	c.logger.Debug("setting new default subscription: %s", config.Subscriptions[targetIndex].Name)
	config.Subscriptions[targetIndex].IsDefault = true

	c.logger.Debug("writing updated configuration")
	if err := c.storage.WriteConfig(config); err != nil {
		c.logger.Error("failed to write configuration: %v", err)
		return pkgerrors.WrapError("writing configuration", err)
	}

	c.logger.Success("switched context to: %s (%s)", config.Subscriptions[targetIndex].Name, subscriptionID)
	return nil
}

// ClearContext clears the default flag on every subscription, leaving the
// active config dir with no default subscription selected.
func (c *ConfigurationAdapter) ClearContext() error {
	config, err := c.storage.ReadConfig()
	if err != nil {
		c.logger.Error("failed to read configuration: %v", err)
		return pkgerrors.WrapError("reading configuration", err)
	}

	for i := range config.Subscriptions {
		config.Subscriptions[i].IsDefault = false
	}

	if err := c.storage.WriteConfig(config); err != nil {
		c.logger.Error("failed to write configuration: %v", err)
		return pkgerrors.WrapError("writing configuration", err)
	}

	c.logger.Success("cleared default subscription")
	return nil
}
