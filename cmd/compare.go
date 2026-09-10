// benchmarks scp vs ripgrep - runs both tools many times and computes
// real statistics (not just a single timing)
package cmd

import (
	"github.com/Viswesh-G/scope/internal/analysis"
	"github.com/spf13/cobra"
)

var (
	comparePattern string
	comparePath    string
	compareRuns    int
	compareWarmup  int
)

var compareCmd = &cobra.Command{
	Use:   "compare",
	Short: "Benchmark scp against ripgrep",
	Long: `Run scp and ripgrep head-to-head on the same search.

Both tools alternate who goes first each round (to cancel out cache warming effects).
A few throwaway warmup rounds run first, then the real timing starts.

Requires ripgrep (rg) on your PATH.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		scopeResult, rgResult, err := analysis.Compare(comparePattern, comparePath, compareRuns, compareWarmup)
		if err != nil {
			return err
		}
		analysis.PrintCompare(compareRuns, compareWarmup, scopeResult, rgResult)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(compareCmd)

	f := compareCmd.Flags()
	f.StringVarP(&comparePattern, "pattern", "p", "", "regex pattern (required)")
	f.StringVar(&comparePath, "path", ".", "directory to search")
	f.IntVar(&compareRuns, "runs", 20, "timed benchmark rounds")
	f.IntVar(&compareWarmup, "warmup", 3, "warmup rounds to discard")

	_ = compareCmd.MarkFlagRequired("pattern")
}
