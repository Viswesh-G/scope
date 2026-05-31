package cmd

import (
	"github.com/Viswesh-G/scope/internal/analysis"
	"github.com/spf13/cobra"
)

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Build directory tree graph",
	Long:  `Build a directory tree graph of the repository, respecting ignore rules.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("path")
		dot, _ := cmd.Flags().GetBool("dot")
		return analysis.RunGraph(path, dot)
	},
}

func init() {
	rootCmd.AddCommand(graphCmd)
	graphCmd.Flags().String("path", ".", "search path")
	graphCmd.Flags().Bool("dot", false, "export to Graphviz DOT format")
}
