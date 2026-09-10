// This file defines `scp deps` — scans Go source files and counts
// how many times each import package appears across the codebase.
package cmd

import (
	"github.com/Viswesh-G/scope/internal/analysis"
	"github.com/spf13/cobra"
)

// depsCmd walks Go source files and counts imports per package.
var depsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Analyse Go import dependencies",
	Long: `Walk all .go files in the repository and count how many times
each import package appears. Useful for understanding which packages
your codebase relies on most heavily.

Example:
  scp deps
  scp deps --path ./internal`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("path")
		return analysis.RunDeps(path)
	},
}

// init registers the deps command.
func init() {
	rootCmd.AddCommand(depsCmd)
	depsCmd.Flags().String("path", ".", "directory to analyse")
}
