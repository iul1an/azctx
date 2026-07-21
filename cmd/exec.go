package cmd

import (
	"fmt"
	"os"

	pkgerrors "github.com/riweston/aztx/pkg/errors"
	"github.com/riweston/aztx/pkg/isolation"
	"github.com/spf13/cobra"
)

// ExitCodeError carries a child process's exit code up to main so aztx can
// exit with the same status without printing an error of its own.
type ExitCodeError struct{ Code int }

func (e ExitCodeError) Error() string {
	return fmt.Sprintf("command exited with code %d", e.Code)
}

var execCmd = &cobra.Command{
	Use:   "exec -- <command> [args...]",
	Short: "Pick a subscription and run a command in the isolated context",
	Long: `exec creates a fresh isolated Azure context (a private copy of ~/.azure with
AZURE_CONFIG_DIR pointing at it), runs the subscription picker, executes the
given command inside that context, and cleans the context up when the command
exits. The command's exit code is propagated.

Similar to aws-vault exec, e.g.:

  aztx exec -- kubectl get pods
  aztx exec --by-tenant -- kubie ctx my-aks-cluster`,
	Args:          cobra.MinimumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		sweepOrphans()

		tmpDir, err := isolation.Setup()
		if err != nil {
			return pkgerrors.ErrOperation("setting up isolated config", err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		picked, err := pickContext(nil)
		if err != nil {
			return err
		}
		if picked == "" {
			return ExitCodeError{Code: 1}
		}
		_ = os.Setenv("AZTX_SUBSCRIPTION", picked)

		code, err := isolation.RunCommand(args)
		if err != nil {
			return err
		}
		if code != 0 {
			return ExitCodeError{Code: code}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(execCmd)
}
