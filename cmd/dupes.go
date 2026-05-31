package cmd

import (
	"runtime"

	"github.com/Viswesh-G/scope/internal/analysis"
	"github.com/spf13/cobra"
)

var dupesCmd = &cobra.Command{
	Use:   "dupes",
	Short: "Find duplicate files",
	Long:  `Find duplicate files by hashing their contents (SHA256).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("path")
		workers, _ := cmd.Flags().GetInt("workers")
		return analysis.RunDupes(path, workers)
	},
}

func init() {
	rootCmd.AddCommand(dupesCmd)
	dupesCmd.Flags().String("path", ".", "search path")
	dupesCmd.Flags().IntP("workers", "w", runtime.NumCPU(), "number of workers")
}
