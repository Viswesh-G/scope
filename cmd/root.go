// cmd/ is the CLI layer. each file = one command (or group of subcommands).
// the commands are thin - they just parse flags and call into internal/.
//
// using cobra for the CLI framework. the pattern is:
//   var someCmd = &cobra.Command{ ... }
//   func init() { rootCmd.AddCommand(someCmd) }
// cobra wires everything together automatically.
package cmd

import (
	"os"

	"github.com/Viswesh-G/scope/internal/config"
	"github.com/Viswesh-G/scope/internal/output"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "scope",
	Short: "A fast, self-profiling grep",
	Long: `Scope is a fast, concurrent, and customizable grep-like tool.
It searches files for regex patterns and shows detailed performance metrics
(worker stats, throughput, load balance) after every search.

Customize its colors with 'scope config set-color'.
Control which files are skipped with 'scope ignore'.`,

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

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// colors might not be initialized yet if the error happened early
		if output.ErrorColor == nil {
			output.InitColors()
		}
		output.PrintError(err)
		os.Exit(1)
	}
}
