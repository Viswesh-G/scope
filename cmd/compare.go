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
	Short: "Benchmark Scope against ripgrep",
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
	f.StringVarP(&comparePattern, "pattern", "p", "", "regex pattern")
	f.StringVar(&comparePath, "path", ".", "search path")
	f.IntVar(&compareRuns, "runs", 20, "number of timed benchmark runs")
	f.IntVar(&compareWarmup, "warmup", 3, "number of warmup runs to discard before timing")

	_ = compareCmd.MarkFlagRequired("pattern")
}