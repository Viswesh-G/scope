// the search command - parse flags, build a config, hand off to the engine
package cmd

import (
	"runtime"

	"github.com/Viswesh-G/scope/internal/search"
	"github.com/spf13/cobra"
)

// cobra fills these in from whatever the user typed on the command line
var (
	flagPattern      string
	flagPath         string
	flagRecursive    bool
	flagWorkers      int
	flagFilenameOnly bool
	flagHotspots     bool
	flagIgnoreCase   bool
	flagNoHistory    bool
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search files for a regex pattern",
	Long: `Search files for a regular expression pattern within a directory.

Searches recursively from the current directory by default, using all CPU cores.
Respects .scope-ignore and skips binaries, .git, node_modules, etc.

Examples:
  scope search -p "TODO"
  scope search -p "func main" --path ./cmd -i
  scope search -p "github" --hotspots`,
	RunE: runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)

	f := searchCmd.Flags()
	f.StringVarP(&flagPattern, "pattern", "p", "", "regex pattern to search for (required)")
	f.StringVar(&flagPath, "path", ".", "directory to search in")
	f.BoolVarP(&flagRecursive, "recursive", "r", true, "search subdirectories")
	f.BoolVarP(&flagIgnoreCase, "ignore-case", "i", false, "case-insensitive")
	f.IntVarP(&flagWorkers, "workers", "w", runtime.NumCPU(), "parallel workers")
	f.BoolVarP(&flagFilenameOnly, "fname", "f", false, "match filenames instead of contents")
	f.BoolVar(&flagHotspots, "hotspots", false, "rank files by match count")

	// hidden flag used by the benchmark so its runs don't pollute history
	f.BoolVar(&flagNoHistory, "no-history", false, "skip saving to history")
	_ = f.MarkHidden("no-history")

	_ = searchCmd.MarkFlagRequired("pattern")
}

func runSearch(cmd *cobra.Command, args []string) error {
	cfg := search.Config{
		Pattern:      flagPattern,
		Path:         flagPath,
		Recursive:    flagRecursive,
		Workers:      flagWorkers,
		FilenameOnly: flagFilenameOnly,
		Hotspots:     flagHotspots,
		IgnoreCase:   flagIgnoreCase,
		SkipHistory:  flagNoHistory,
	}
	return search.Run(cfg)
}
