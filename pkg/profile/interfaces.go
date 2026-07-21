// Package profile provides interfaces and implementations for managing Azure profiles,
// including tenant and subscription management, configuration storage, and logging.
package profile

import (
	"github.com/iul1an/azctx/pkg/types"
)

// StorageAdapter defines the interface for configuration storage operations.
// Implementations of this interface handle reading and writing of Azure configuration data.
type StorageAdapter interface {
	// ReadConfig retrieves the current Azure configuration.
	// Returns a Configuration object and any error encountered during the read operation.
	ReadConfig() (*types.Configuration, error)

	// WriteConfig persists the provided Azure configuration.
	// Returns an error if the write operation fails.
	WriteConfig(*types.Configuration) error
}

// Logger defines the interface for logging operations.
// It provides standard logging levels and formatting capabilities.
type Logger interface {
	// Info logs informational messages with optional formatting arguments.
	Info(msg string, args ...interface{})

	// Error logs error messages with optional formatting arguments.
	Error(msg string, args ...interface{})

	// Debug logs debug messages with optional formatting arguments.
	Debug(msg string, args ...interface{})

	// Warn logs warning messages with optional formatting arguments.
	Warn(msg string, args ...interface{})

	// Success logs success messages with optional formatting arguments.
	Success(msg string, args ...interface{})
}
