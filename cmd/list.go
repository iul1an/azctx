package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/riweston/aztx/pkg/isolation"
	"github.com/spf13/cobra"
)

// sweepOrphans garbage-collects contexts whose owning process is gone,
// telling the user when it did something. Best-effort.
func sweepOrphans() {
	if n := isolation.Sweep(); n > 0 {
		fmt.Fprintf(os.Stderr, "cleaned up %d orphaned aztx context(s)\n", n)
	}
}

var listCmd = &cobra.Command{
	Use:           "list",
	Aliases:       []string{"ls"},
	Short:         "List isolated aztx contexts",
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctxs, err := isolation.ListContexts()
		if err != nil {
			return err
		}
		if len(ctxs) == 0 {
			fmt.Println("no isolated contexts")
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
	},
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
	rootCmd.AddCommand(listCmd)
}
