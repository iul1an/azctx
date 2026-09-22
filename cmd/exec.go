package cmd

import (
	"fmt"
	"os"

	pkgerrors "github.com/iul1an/azctx/pkg/errors"
	"github.com/iul1an/azctx/pkg/isolation"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ExitCodeError carries a child process's exit code up to main so azctx can
// exit with the same status without printing an error of its own.
type ExitCodeError struct{ Code int }

func (e ExitCodeError) Error() string {
	return fmt.Sprintf("command exited with code %d", e.Code)
}

var execCmd = &cobra.Command{
	Use:   "exec [-- <command> [args...]]",
	Short: "Pick a subscription and run a command (or a shell) in the isolated context",
	Long: `exec creates a fresh isolated Azure context (a private copy of ~/.azure with
AZURE_CONFIG_DIR pointing at it), runs the subscription picker, executes the
given command inside that context, and cleans the context up when the command
exits. The command's exit code is propagated.

With no command, exec drops into an isolated subshell instead, so it doubles
as a non-interactive way into bare azctx, e.g. skipping the picker with
--subscription.

Similar to aws-vault exec, e.g.:

  azctx exec -- kubectl get pods
  azctx exec --by-tenant -- kubie ctx my-aks-cluster
  azctx exec --subscription "My Subscription"`,
	Args:          cobra.ArbitraryArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		sweepOrphans()

		// Refuse to nest inside an existing isolated shell, like bare azctx
		// does: nesting is confusing and buys nothing. Exit first and re-run.
		if isolation.IsActive() {
			return fmt.Errorf(
				"already inside an azctx isolated shell (AZCTX_SUBSCRIPTION=%q); exit it first and re-run azctx exec",
				os.Getenv("AZCTX_SUBSCRIPTION"))
		}

		// Fresh mode: run the command in an empty context, no picker.
		if viper.GetBool("fresh") {
			tmpDir, err := isolation.SetupEmpty()
			if err != nil {
				return pkgerrors.ErrOperation("setting up fresh config", err)
			}
			defer func() { _ = os.RemoveAll(tmpDir) }()
			if len(args) == 0 {
				fmt.Fprintln(os.Stderr, "fresh empty Azure context; run `az login` inside to use it")
			}
			return runOrShell(args)
		}

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
		_ = os.Setenv("AZCTX_SUBSCRIPTION", picked)

		return runOrShell(args)
	},
}

// runOrShell runs argv in the active isolated context, propagating its exit
// code, or drops into an isolated subshell when no command was given. A
// non-zero command exit is surfaced as ExitCodeError so main can mirror the
// status; a shell's own exit status is not treated as an azctx error.
func runOrShell(args []string) error {
	if len(args) == 0 {
		return isolation.SpawnShell()
	}
	code, err := isolation.RunCommand(args)
	if err != nil {
		return err
	}
	if code != 0 {
		return ExitCodeError{Code: code}
	}
	return nil
}

func init() {
	rootCmd.AddCommand(execCmd)
}
