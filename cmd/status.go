package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	pkgerrors "github.com/riweston/aztx/pkg/errors"
	"github.com/riweston/aztx/pkg/isolation"
	"github.com/riweston/aztx/pkg/storage"
	"github.com/spf13/cobra"
)

type statusSubscription struct {
	Name     string `json:"name"`
	ID       string `json:"id"`
	TenantID string `json:"tenantId"`
}

type statusOutput struct {
	Isolated         bool                `json:"isolated"`
	ConfigDir        string              `json:"configDir,omitempty"`
	Subscription     *statusSubscription `json:"subscription"`
	EnvSubscription  string              `json:"aztxSubscriptionEnv,omitempty"`
	EnvMatchesConfig *bool               `json:"envMatchesConfig,omitempty"`
	PID              int                 `json:"pid,omitempty"`
	Started          *time.Time          `json:"started,omitempty"`
}

var statusCmd = &cobra.Command{
	Use:           "status",
	Short:         "Show the current shell's Azure context as JSON",
	Long:          "Prints the active context as indented JSON. Exits 1 when not inside an aztx isolated shell.",
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		out := statusOutput{Isolated: isolation.IsActive()}

		storage := storage.FileAdapter{}
		if err := storage.FetchDefaultPath("azureProfile.json"); err != nil {
			return pkgerrors.ErrFileOperation("fetching default profile path", err)
		}
		out.ConfigDir = filepath.Dir(storage.Path)

		cfg, err := readActiveConfig()
		if err != nil {
			return err
		}
		for _, s := range cfg.Subscriptions {
			if s.IsDefault {
				out.Subscription = &statusSubscription{Name: s.Name, ID: s.ID.String(), TenantID: s.TenantID.String()}
				break
			}
		}

		if out.Isolated {
			out.EnvSubscription = os.Getenv("AZTX_SUBSCRIPTION")
			// Both empty (e.g. a --fresh context) is agreement too.
			matches := (out.Subscription == nil && out.EnvSubscription == "") ||
				(out.Subscription != nil && out.Subscription.Name == out.EnvSubscription)
			out.EnvMatchesConfig = &matches
			for _, c := range mustListContexts() {
				if c.Active {
					out.PID = c.PID
					started := c.Started.Truncate(time.Second)
					out.Started = &started
					break
				}
			}
		}

		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))

		if !out.Isolated {
			return ExitCodeError{Code: 1}
		}
		return nil
	},
}

func mustListContexts() []isolation.Context {
	ctxs, err := isolation.ListContexts()
	if err != nil {
		return nil
	}
	return ctxs
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
