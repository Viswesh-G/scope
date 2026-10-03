// benchmarks scope vs ripgrep - runs both tools many times and computes
// real statistics (not just a single timing)
package cmd

import (
	"runtime"

	"github.com/Viswesh-G/scope/internal/analysis"
	"github.com/spf13/cobra"
)

var (
	comparePattern string
	comparePath    string
	compareRuns    int
	compareWarmup  int
	compareWorkers int
	compareGlobs   []string
	compareSave    string
)

var compareCmd = &cobra.Command{
	Use:   "compare",
	Short: "Benchmark scope against ripgrep",
	Long: `Run scope and ripgrep head-to-head on the same search.

Both tools alternate who goes first each round. Warmups try to warm the OS file
cache, but cold-cache measurements are not supported. Results include tool
versions, environment details, repository metadata, and match counts.

Scope and ripgrep apply different ignore rules. The report calls out that
difference and warns when result counts differ. Use --save to write the full
measurement record and raw per-run samples as JSON.

Requires ripgrep (rg) on your PATH.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		report, err := analysis.Compare(analysis.CompareOptions{
			Pattern: comparePattern,
			Path:    comparePath,
			Runs:    compareRuns,
			Warmup:  compareWarmup,
			Workers: compareWorkers,
			Globs:   compareGlobs,
		})
		if err != nil {
			return err
		}
		analysis.PrintCompare(report)
		if compareSave != "" {
			if err := analysis.SaveCompareReport(compareSave, report); err != nil {
				return err
			}
			cmd.Printf("Saved benchmark record to %s\n", compareSave)
		}
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
	f.IntVarP(&compareWorkers, "workers", "w", runtime.NumCPU(), "worker/thread count for both tools")
	f.StringSliceVarP(&compareGlobs, "glob", "g", nil, "include/exclude file glob (repeatable)")
	f.StringVar(&compareSave, "save", "", "write a JSON benchmark record to this path")

	_ = compareCmd.MarkFlagRequired("pattern")
}
