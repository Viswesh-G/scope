// This file defines `scope stats` — shows a breakdown of file extensions
// found in the repository (e.g. 42 .go files, 12 .md files, etc.).
package cmd

import (
	"github.com/Viswesh-G/scope/internal/analysis"
	"github.com/spf13/cobra"
)

// statsCmd scans the repository and counts files by their extension.
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show file extension statistics",
	Long: `Walk the repository and count how many files exist per file extension.
Respects .scope-ignore rules and built-in defaults (skips .git, node_modules, etc.)

Example:
  scope stats
  scope stats --path ./src`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("path")
		return analysis.RunExtStats(path)
	},
}

// init registers the stats command.
func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.Flags().String("path", ".", "directory to analyse")
}
