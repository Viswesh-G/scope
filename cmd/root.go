package cmd

import (
	"os"

	"github.com/Viswesh-G/scope/internal/config"
	"github.com/Viswesh-G/scope/internal/output"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "scope",
	Short: "A fast self profiling grep",
	Long: `Scope is a fast, concurrent, and highly customizable grep-like utility.
It is built with speed in mind and automatically profiles itself. You can personalize
its appearance through the 'config' command and easily manage ignore files with
the 'ignore' command.`,
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
		// If colors aren't initialized yet (e.g. error before PersistentPreRunE), 
		// output.PrintError will safely fall back to uncolored or default colored text.
		if output.ErrorColor == nil {
			output.InitColors()
		}
		output.PrintError(err)
		os.Exit(1)
	}
}
