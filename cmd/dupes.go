// This file defines `scp dupes` — finds files with identical contents
// by computing a SHA256 hash of every file and grouping matches.
package cmd

import (
	"runtime"

	"github.com/Viswesh-G/scope/internal/analysis"
	"github.com/spf13/cobra"
)

// dupesCmd hashes all files in the repo and groups any that are identical.
var dupesCmd = &cobra.Command{
	Use:   "dupes",
	Short: "Find duplicate files by content",
	Long: `Scan all files in the repository and detect duplicates.
Two files are considered duplicates if their SHA256 hashes match —
meaning they have byte-for-byte identical contents.

Uses parallel workers (same concurrency model as search) for speed.

Example:
  scp dupes
  scp dupes --path ./assets -w 4`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("path")
		workers, _ := cmd.Flags().GetInt("workers")
		return analysis.RunDupes(path, workers)
	},
}

// init registers the dupes command.
func init() {
	rootCmd.AddCommand(dupesCmd)
	dupesCmd.Flags().String("path", ".", "directory to scan")
	dupesCmd.Flags().IntP("workers", "w", runtime.NumCPU(), "number of parallel workers")
}
