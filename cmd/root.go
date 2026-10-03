// cmd/ is the CLI layer. each file = one command (or group of subcommands).
// the commands are thin - they just parse flags and call into internal/.
//
// using cobra for the CLI framework. the pattern is:
//
//	var someCmd = &cobra.Command{ ... }
//	func init() { rootCmd.AddCommand(someCmd) }
//
// cobra wires everything together automatically.
package cmd

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/Viswesh-G/scope/internal/config"
	"github.com/Viswesh-G/scope/internal/output"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "scope",
	Aliases: []string{"scp"},
	Short:   "A fast, local-first codebase intelligence and profiling engine",
	Long: `scope is a fast, local-first codebase intelligence tool.
It searches files for regex patterns, parses Go AST structure, and displays
useful performance metrics like throughput, parallelism, and load balance.

  scope search -p "TODO"          # search files
  scope ast --type func           # search Go AST functions
  scope refs --name Run           # find Go symbol references
  scope tui                       # open interactive terminal interface
  scope serve                     # launch live web dashboard
  scope compare -p "func"         # benchmark against ripgrep

You can also run commands using the short alias 'scp'.`,

	// runs before every subcommand - loads config + sets up colors
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Init(); err != nil {
			return err
		}
		output.InitColors()
		return nil
	},

	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	// this generates the "completion" subcommand for bash/zsh/fish/powershell automatically
	rootCmd.InitDefaultCompletionCmd()
}

func Execute() {
	// Ctrl+C (or a `kill`) cancels this context instead of the process just
	// being torn down from the outside. Any long running work that watches it
	// can stop at the next safe point, so we still print results and tidy up
	// after ourselves instead of leaving half-written files behind.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rootCmd.SetContext(ctx)

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		// if the user hit Ctrl+C there is nothing to report, they already know
		if errors.Is(err, context.Canceled) {
			os.Exit(130)
		}

		// colors might not be initialized yet if the error happened early
		if output.ErrorColor == nil {
			output.InitColors()
		}
		output.PrintError(err)
		os.Exit(1)
	}
}
