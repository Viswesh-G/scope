package cmd

import (
	"github.com/Viswesh-G/scope/internal/analysis"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "ext-stats",
	Short: "Show file extension statistics",
	Long:  `Show statistics for file extensions across the repository.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("path")
		return analysis.RunExtStats(path)
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.Flags().String("path", ".", "search path")
}
