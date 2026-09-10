// This file defines `scp graph` — prints the repository directory tree,
// optionally as a Graphviz DOT file for visualization.
package cmd

import (
	"github.com/Viswesh-G/scope/internal/analysis"
	"github.com/spf13/cobra"
)

// graphCmd renders the directory structure as a tree or DOT graph.
var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Print the directory tree",
	Long: `Walk the repository and print its directory structure as a tree.
Respects .scope-ignore rules (same as search).

Use --dot to export a Graphviz DOT file instead of an ASCII tree.
You can then visualise it with: dot -Tsvg graph.dot > graph.svg

Examples:
  scp graph
  scp graph --path ./internal
  scp graph --dot > graph.dot`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("path")
		dot, _ := cmd.Flags().GetBool("dot")
		return analysis.RunGraph(path, dot)
	},
}

// init registers the graph command.
func init() {
	rootCmd.AddCommand(graphCmd)
	graphCmd.Flags().String("path", ".", "directory to graph")
	graphCmd.Flags().Bool("dot", false, "output Graphviz DOT format instead of ASCII tree")
}
