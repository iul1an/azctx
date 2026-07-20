package cmd

import (
	"github.com/riweston/aztx/pkg/storage"
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
	names := make([]string, 0, len(cfg.Subscriptions))
	for _, s := range cfg.Subscriptions {
		names = append(names, s.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
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
