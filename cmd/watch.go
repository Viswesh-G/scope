// cmd/watch.go implements "scp watch": continuously watch a directory for file
// changes and re-run the search every time a file is modified.
//
// This is the "live grep" experience — you run it in one terminal pane while
// you edit code in another, and the matches update automatically.
package cmd

import (
	"github.com/Viswesh-G/scope/internal/watch"
	"github.com/spf13/cobra"
)

var (
	watchPattern    string
	watchPath       string
	watchWorkers    int
	watchIgnoreCase bool
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch a directory and re-run the search on file changes",
	Long: `Watch a directory for file changes and re-run the search automatically.

Press Ctrl+C to stop watching.

Examples:
  scp watch -p "TODO"
  scp watch -p "func main" --path ./cmd -i
  scp watch -p "error" --workers 4`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := watch.Config{
			Pattern:    watchPattern,
			Path:       watchPath,
			Workers:    watchWorkers,
			IgnoreCase: watchIgnoreCase,
		}
		return watch.Run(cfg)
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)

	f := watchCmd.Flags()
	f.StringVarP(&watchPattern, "pattern", "p", "", "regex pattern to watch for (required)")
	f.StringVar(&watchPath, "path", ".", "directory to watch")
	f.IntVarP(&watchWorkers, "workers", "w", 4, "number of parallel search workers")
	f.BoolVarP(&watchIgnoreCase, "ignore-case", "i", false, "case-insensitive matching")

	_ = watchCmd.MarkFlagRequired("pattern")
}
