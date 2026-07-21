package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	pkgerrors "github.com/iul1an/azctx/pkg/errors"
	"github.com/iul1an/azctx/pkg/isolation"
	"github.com/iul1an/azctx/pkg/storage"
	"github.com/iul1an/azctx/pkg/types"
	"github.com/spf13/cobra"
)

// sweepOrphans garbage-collects contexts whose owning process is gone,
// telling the user when it did something. Best-effort.
func sweepOrphans() {
	if n := isolation.Sweep(); n > 0 {
		fmt.Fprintf(os.Stderr, "cleaned up %d orphaned azctx context(s)\n", n)
	}
}

var listCmd = &cobra.Command{
	Use:           "list",
	Aliases:       []string{"ls"},
	Short:         "List subscriptions and isolated azctx contexts",
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
			return listJSON()
		}
		if err := listSubscriptions(); err != nil {
			return err
		}
		fmt.Println()
		return listIsolatedContexts()
	},
}

type listSubscriptionEntry struct {
	Name      string `json:"name"`
	ID        string `json:"id"`
	TenantID  string `json:"tenantId"`
	IsDefault bool   `json:"isDefault"`
}

func listJSON() error {
	cfg, err := readActiveConfig()
	if err != nil {
		return err
	}
	ctxs, err := isolation.ListContexts()
	if err != nil {
		return err
	}
	for i := range ctxs {
		ctxs[i].Started = ctxs[i].Started.Truncate(time.Second)
	}

	out := struct {
		Subscriptions    []listSubscriptionEntry `json:"subscriptions"`
		IsolatedContexts []isolation.Context     `json:"isolatedContexts"`
	}{
		Subscriptions:    []listSubscriptionEntry{},
		IsolatedContexts: ctxs,
	}
	if out.IsolatedContexts == nil {
		out.IsolatedContexts = []isolation.Context{}
	}
	for _, s := range cfg.Subscriptions {
		out.Subscriptions = append(out.Subscriptions, listSubscriptionEntry{
			Name: s.Name, ID: s.ID.String(), TenantID: s.TenantID.String(), IsDefault: s.IsDefault,
		})
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// readActiveConfig reads the active config dir's profile. A missing
// azureProfile.json is a valid empty state (e.g. a --fresh context before
// the first az login), not an error.
func readActiveConfig() (*types.Configuration, error) {
	storage := storage.FileAdapter{}
	if err := storage.FetchDefaultPath("azureProfile.json"); err != nil {
		return nil, pkgerrors.ErrFileOperation("fetching default profile path", err)
	}
	cfg, err := storage.ReadConfig()
	if err != nil {
		if errors.Is(err, pkgerrors.ErrFileDoesNotExist) {
			return &types.Configuration{}, nil
		}
		return nil, pkgerrors.ErrReadingConfiguration(err)
	}
	return cfg, nil
}

// listSubscriptions prints the subscriptions of the active Azure config dir
// (honoring AZURE_CONFIG_DIR); '*' marks the default.
func listSubscriptions() error {
	cfg, err := readActiveConfig()
	if err != nil {
		return err
	}

	fmt.Println("SUBSCRIPTIONS")
	if len(cfg.Subscriptions) == 0 {
		fmt.Println("  none")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, " \tNAME\tID\tTENANT")
	for _, s := range cfg.Subscriptions {
		marker := " "
		if s.IsDefault {
			marker = "*"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", marker, s.Name, s.ID, s.TenantID)
	}
	return w.Flush()
}

func listIsolatedContexts() error {
	ctxs, err := isolation.ListContexts()
	if err != nil {
		return err
	}

	fmt.Println("ISOLATED CONTEXTS")
	if len(ctxs) == 0 {
		fmt.Println("  none")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, " \tCONFIG DIR\tSUBSCRIPTION\tPID\tAGE\tSTATE")
	for _, c := range ctxs {
		marker := " "
		if c.Active {
			marker = "*"
		}
		state := "live"
		if !c.Alive {
			state = "orphaned"
		}
		sub := c.Subscription
		if sub == "" {
			sub = "-"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
			marker, c.Dir, sub, c.PID, formatAge(time.Since(c.Started)), state)
	}
	return w.Flush()
}

func formatAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	default:
		return fmt.Sprintf("%dd%02dh", int(d.Hours())/24, int(d.Hours())%24)
	}
}

func init() {
	listCmd.Flags().Bool("json", false, "Output as indented JSON")
	rootCmd.AddCommand(listCmd)
}
