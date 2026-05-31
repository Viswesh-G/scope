package cmd

import (
	"github.com/Viswesh-G/scope/internal/analysis"
	"github.com/spf13/cobra"
)

var depsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Analyze Go dependencies",
	Long:  `Analyze and count Go imports across the repository.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("path")
		return analysis.RunDeps(path)
	},
}

func init() {
	rootCmd.AddCommand(depsCmd)
	depsCmd.Flags().String("path", ".", "search path")
}
