package cmd

import (
	"fmt"
	"github.com/iul1an/azctx/pkg/storage"
	"github.com/spf13/cobra"
)

// completeSubscriptions completes --subscription values with the
// subscription names from the active Azure config dir.
func completeSubscriptions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	storage := storage.FileAdapter{}
	if err := storage.FetchDefaultPath("azureProfile.json"); err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	cfg, err := storage.ReadConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	// Complete IDs, with the name shown as the menu description. Names
	// often contain spaces, which cobra's zsh script cannot re-parse when
	// continuing a partially completed word (Tab goes dead after the
	// escaped space); IDs are space-free and --subscription accepts them.
	entries := make([]string, 0, len(cfg.Subscriptions))
	for _, s := range cfg.Subscriptions {
		entries = append(entries, fmt.Sprintf("%s\t%s", s.ID, s.Name))
	}
	return entries, cobra.ShellCompDirectiveNoFileComp
}

// registerCompletions attaches flag value completions; called from root.go's
// init after the flags are defined (file init order would run this first).
func registerCompletions() {
	if err := rootCmd.RegisterFlagCompletionFunc("subscription", completeSubscriptions); err != nil {
		panic(err)
	}
	if err := rootCmd.RegisterFlagCompletionFunc("log-level", cobra.FixedCompletions(
		[]string{"debug", "info", "warn", "error"}, cobra.ShellCompDirectiveNoFileComp)); err != nil {
		panic(err)
	}
}
