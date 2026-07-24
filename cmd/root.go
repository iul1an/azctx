/*
Copyright © 2024 Richard Weston

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
// Package cmd provides the command-line interface for the azctx application.
// It implements the core functionality for switching between Azure tenants and subscriptions
// using a fuzzy finder interface.
package cmd

import (
	"bytes"
	"errors"

	"fmt"
	"github.com/google/uuid"
	"os"
	"path/filepath"
	"strings"

	pkgerrors "github.com/iul1an/azctx/pkg/errors"
	"github.com/iul1an/azctx/pkg/finder"
	"github.com/iul1an/azctx/pkg/isolation"
	"github.com/iul1an/azctx/pkg/profile"
	"github.com/iul1an/azctx/pkg/state"
	"github.com/iul1an/azctx/pkg/storage"
	"github.com/iul1an/azctx/pkg/subscription"
	"github.com/iul1an/azctx/pkg/tenant"
	"github.com/iul1an/azctx/pkg/types"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "azctx",
	Short: "Azure Tenant Context Switcher",
	Long: `azctx is a command line tool that helps you switch between Azure tenants and subscriptions.
It provides a fuzzy finder interface to select subscriptions and remembers your last context.`,
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	// Cross-flag validation, shared by root and every subcommand.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		isolation.SetLogger(profile.NewLogger(viper.GetString("log-level")))

		// A fresh context is empty, so there is no subscription to select.
		// Check Changed, not viper, to ignore the exported AZCTX_SUBSCRIPTION.
		if cmd.Flags().Changed("fresh") && cmd.Flags().Changed("subscription") {
			return fmt.Errorf("--fresh and --subscription cannot be used together: a fresh context is empty, with no subscription to select")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		sweepOrphans()

		// Refused inside an isolated shell, with a message per mode.
		if isolation.IsActive() {
			// --unset/--in-place would hit the temp copy, not master ~/.azure.
			if viper.GetBool("unset") || viper.GetBool("in-place") {
				return fmt.Errorf(
					"cannot modify the master ~/.azure from inside an azctx isolated shell: this shell is scoped to a temporary copy; exit it first")
			}
			// A re-pick can't update the exported AZCTX_SUBSCRIPTION.
			return fmt.Errorf(
				"already inside an azctx isolated shell (AZCTX_SUBSCRIPTION=%q); exit it and re-run azctx, or use azctx exec for a one-off command in another context",
				os.Getenv("AZCTX_SUBSCRIPTION"))
		}

		// Unset mode: clear the default subscription in ~/.azure, no picker,
		// no subshell.
		if viper.GetBool("unset") {
			storage := storage.FileAdapter{}
			if err := storage.FetchDefaultPath("azureProfile.json"); err != nil {
				return pkgerrors.ErrFileOperation("fetching default profile path", err)
			}
			adapter := profile.NewConfigurationAdapter(&storage, profile.NewLogger(viper.GetString("log-level")).SetQuiet(viper.GetBool("quiet")))
			return adapter.ClearContext()
		}

		// In-place mode: mutate the master ~/.azure directly, no subshell.
		if viper.GetBool("in-place") {
			_, err := pickContext(args)
			return err
		}

		// Fresh mode: an empty isolated context, nothing copied, no picker.
		// az behaves as never-logged-in inside; everything done there (e.g.
		// an az login) vanishes when the shell exits.
		if viper.GetBool("fresh") {
			tmpDir, err := isolation.SetupEmpty()
			if err != nil {
				return pkgerrors.ErrOperation("setting up fresh config", err)
			}
			defer func() { _ = os.RemoveAll(tmpDir) }()
			fmt.Fprintln(os.Stderr, "fresh empty Azure context; run `az login` inside to use it")
			return isolation.SpawnShell()
		}

		// Isolated mode (the default): copy ~/.azure to a private tempdir,
		// pick inside it, then drop into a subshell scoped to the copy.
		tmpDir, err := isolation.Setup()
		if err != nil {
			return pkgerrors.ErrOperation("setting up isolated config", err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		picked, err := pickContext(args)
		if err != nil || picked == "" {
			return err
		}
		_ = os.Setenv("AZCTX_SUBSCRIPTION", picked)
		return isolation.SpawnShell()
	},
}

// pickContext runs the subscription/tenant picker against the active Azure
// config dir (honoring AZURE_CONFIG_DIR) and returns the name of the picked
// subscription. It returns "" when the user aborted the fuzzy finder without
// picking anything.
func pickContext(args []string) (string, error) {
	finder.Configure(viper.GetStringSlice("picker.options"), viper.GetBool("picker.preview"))

	logger := profile.NewLogger(viper.GetString("log-level")).SetQuiet(viper.GetBool("quiet"))
	aliases := configuredAliases()
	if configFile == "" {
		logger.Debug("no config file")
	} else {
		logger.Debug("config file %s, %d alias(es)", configFile, len(aliases))
	}
	if opts := viper.GetStringSlice("picker.options"); len(opts) > 0 {
		logger.Debug("picker options %v", opts)
	}

	stateManager := state.NewFileStateManager()
	storage := storage.FileAdapter{}
	if err := storage.FetchDefaultPath("azureProfile.json"); err != nil {
		return "", pkgerrors.ErrFileOperation("fetching default profile path", err)
	}
	logger.Debug("profile %s (AZURE_CONFIG_DIR=%q)", storage.Path, os.Getenv("AZURE_CONFIG_DIR"))

	cfg, err := storage.ReadConfig()
	if err != nil {
		return "", pkgerrors.ErrReadingConfiguration(err)
	}
	logger.Debug("%d subscription(s) in profile", len(cfg.Subscriptions))
	for _, alias := range (&subscription.Manager{
		BaseManager: types.BaseManager{Configuration: cfg}, Aliases: aliases,
	}).DanglingAliases() {
		logger.Debug("alias %q points at %q, which matches no subscription", alias, aliases[alias])
	}

	// setContext switches to the given subscription and records the switch
	// for `azctx -`. State-write failures are not worth failing the switch
	// over; they only degrade the previous-context feature.
	setContext := func(id uuid.UUID, name string) (string, error) {
		adapter := profile.NewConfigurationAdapter(&storage, logger)
		if err := adapter.SetContext(id); err != nil {
			return "", pkgerrors.ErrOperation("setting context", err)
		}
		if err := stateManager.RecordSwitch(id.String(), name); err != nil {
			logger.Warn("failed to record context switch: %v", err)
		}
		return name, nil
	}

	// Non-interactive selection by subscription name or ID
	if query := viper.GetString("subscription"); query != "" {
		subManager := subscription.Manager{BaseManager: types.BaseManager{Configuration: cfg}, Aliases: aliases}
		sub, err := subManager.FindSubscriptionByNameOrID(query)
		if err != nil {
			return "", pkgerrors.ErrOperation(fmt.Sprintf("finding subscription %q", query), err)
		}
		logger.Debug("%q resolved to %s (%s)", query, sub.Name, sub.ID)
		return setContext(sub.ID, sub.Name)
	}

	if len(args) > 0 && args[0] == "-" {
		targetID, targetName := stateManager.GetCurrentContext()
		if targetID == "" {
			return "", pkgerrors.ErrSettingPreviousContext(pkgerrors.ErrNoPreviousContext)
		}
		// If the active profile is already on the most recent pick (e.g.
		// in-place usage), "-" means the one before it — cd - toggling.
		// Otherwise "-" re-enters the most recent pick.
		reason := "re-entering the most recent pick"
		for _, s := range cfg.Subscriptions {
			if s.IsDefault && s.ID.String() == targetID {
				lastID, lastName := stateManager.GetLastContext()
				if lastID == "" {
					return "", pkgerrors.ErrSettingPreviousContext(pkgerrors.ErrNoPreviousContext)
				}
				targetID, targetName = lastID, lastName
				reason = "profile is already on the most recent pick, toggling back"
				break
			}
		}
		logger.Debug("- resolves to %s (%s): %s", targetName, targetID, reason)
		id, err := uuid.Parse(targetID)
		if err != nil {
			return "", pkgerrors.WrapError("parsing previous subscription ID", err)
		}
		return setContext(id, targetName)
	}

	// Check if tenant selection is requested
	if viper.GetBool("by-tenant") {
		tenantManager := tenant.Manager{BaseManager: types.BaseManager{Configuration: cfg}}
		selectedTenant, err := tenantManager.FindTenantIndex()
		if err != nil {
			if errors.Is(err, finder.ErrAbort) {
				return "", nil
			}
			return "", pkgerrors.ErrTenantOperation("selecting tenant", err)
		}

		subManager := subscription.Manager{BaseManager: types.BaseManager{Configuration: cfg}, Aliases: aliases}
		sub, err := subManager.FindSubscriptionIndexByTenant(selectedTenant.ID)
		if err != nil {
			if errors.Is(err, finder.ErrAbort) {
				return "", nil
			}
			return "", pkgerrors.ErrSelectingSubscription(err)
		}

		return setContext(sub.ID, sub.Name)
	}

	// Default subscription selection
	adapter := profile.NewConfigurationAdapter(&storage, logger).WithAliases(aliases)
	sub, err := adapter.SelectWithFinder()
	if err != nil {
		if errors.Is(err, finder.ErrAbort) {
			return "", nil
		}
		return "", pkgerrors.ErrSelectingSubscription(err)
	}

	return setContext(sub.ID, sub.Name)
}

// SetVersion sets the version reported by the --version flag.
func SetVersion(v string) {
	rootCmd.Version = v
}

// Execute adds all child commands to the root command and sets flags appropriately.
// It is called by main.main() and only needs to happen once to the rootCmd.
// Returns an error if the command execution fails.
func Execute() error {
	return rootCmd.Execute()
}

// init initializes the command configuration by setting up flags and binding them to viper.
// It is automatically called by cobra during command initialization.
func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().String("log-level", "info", "Set log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().Bool("by-tenant", false, "Select tenant before choosing subscription")
	rootCmd.PersistentFlags().String("subscription", "", "Select subscription by configured alias, name, or ID without the interactive picker")
	rootCmd.Flags().Bool("in-place", false, "Mutate the master ~/.azure directly instead of spawning an isolated subshell")
	rootCmd.Flags().Bool("unset", false, "Clear the default subscription in the master ~/.azure and exit")
	rootCmd.PersistentFlags().Bool("fresh", false, "Start from an empty Azure config (skip copying ~/.azure, no picker) for ephemeral workflows")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress the \"switched context to\" confirmation message")

	// Bind flags to viper and check for errors
	if err := viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level")); err != nil {
		logger := profile.NewLogger("error")
		logger.Error("Failed to bind log-level flag: %v", err)
		os.Exit(1)
	}
	if err := viper.BindPFlag("by-tenant", rootCmd.PersistentFlags().Lookup("by-tenant")); err != nil {
		logger := profile.NewLogger("error")
		logger.Error("Failed to bind by-tenant flag: %v", err)
		os.Exit(1)
	}
	if err := viper.BindPFlag("subscription", rootCmd.PersistentFlags().Lookup("subscription")); err != nil {
		logger := profile.NewLogger("error")
		logger.Error("Failed to bind subscription flag: %v", err)
		os.Exit(1)
	}
	if err := viper.BindPFlag("in-place", rootCmd.Flags().Lookup("in-place")); err != nil {
		logger := profile.NewLogger("error")
		logger.Error("Failed to bind in-place flag: %v", err)
		os.Exit(1)
	}
	if err := viper.BindPFlag("unset", rootCmd.Flags().Lookup("unset")); err != nil {
		logger := profile.NewLogger("error")
		logger.Error("Failed to bind unset flag: %v", err)
		os.Exit(1)
	}
	if err := viper.BindPFlag("fresh", rootCmd.PersistentFlags().Lookup("fresh")); err != nil {
		logger := profile.NewLogger("error")
		logger.Error("Failed to bind fresh flag: %v", err)
		os.Exit(1)
	}
	if err := viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet")); err != nil {
		logger := profile.NewLogger("error")
		logger.Error("Failed to bind quiet flag: %v", err)
		os.Exit(1)
	}

	registerCompletions()
}

// configFile is the config that was loaded, "" when none was found. Only
// used for debug output.
var configFile string

// initConfig loads the config, exiting 1 on any problem with it.
func initConfig() {
	if err := loadConfig(); err != nil {
		profile.NewLogger("error").Error("%v", err)
		os.Exit(1)
	}
}

// loadConfig reads ~/.azctx.yml (or AZCTX_CONFIG_FILE) and sets up the
// AZCTX_* environment. The file is read once here and handed to viper, so
// validation and viper see the same bytes. It is optional and never created.
func loadConfig() error {
	viper.SetConfigType("yml")
	viper.SetEnvPrefix("AZCTX")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	path, err := resolveConfigFile()
	if err != nil {
		return err
	}
	explicit := path != ""
	if !explicit {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolving home directory: %w", err)
		}
		path = filepath.Join(home, ".azctx.yml")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		// A missing default config is the normal case. Auto-creating it
		// would snapshot whatever flags the first run passed, e.g.
		// `--fresh` forever.
		if os.IsNotExist(err) && !explicit {
			return nil
		}
		return fmt.Errorf("reading config: %w", err)
	}

	if err := validateAliasKeys(data, path); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	if err := viper.ReadConfig(bytes.NewReader(data)); err != nil {
		return fmt.Errorf("reading config %s: %w", path, err)
	}
	configFile = path
	return nil
}
