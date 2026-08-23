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
	flagProfile      bool
	flagQuiet        bool   // --quiet: skip the big metrics table
	flagCount        bool   // --count: print only the total match count
	flagMaxResults   int    // --max-results: stop after N matches
	flagOutputFile   string // --output: write matches to a file instead of stdout
	flagHTMLFile     string // --html: write a beautiful HTML report to this file
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
  scope search -p "github" --hotspots
  scope search -p "TODO" --count
  scope search -p "error" --max-results 20
  scope search -p "TODO" -o matches.txt -q`,
	RunE: runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)

	f := searchCmd.Flags()
	f.StringVarP(&flagPattern, "pattern", "p", "", "regex pattern to search for (required)")
	f.StringVar(&flagPath, "path", ".", "directory to search in")
	f.BoolVarP(&flagRecursive, "recursive", "r", true, "search subdirectories")
	f.BoolVarP(&flagIgnoreCase, "ignore-case", "i", false, "case-insensitive search")
	f.IntVarP(&flagWorkers, "workers", "w", runtime.NumCPU(), "number of parallel workers")
	f.BoolVarP(&flagFilenameOnly, "fname", "f", false, "match filenames instead of file contents")
	f.BoolVar(&flagHotspots, "hotspots", false, "rank files by match count instead of printing lines")

	// output control flags
	f.BoolVarP(&flagQuiet, "quiet", "q", false, "suppress the metrics table (just show matches)")
	f.BoolVar(&flagCount, "count", false, "print only the total match count, not each line")
	f.IntVarP(&flagMaxResults, "max-results", "m", 0, "stop after this many matches (0 = unlimited)")
	f.StringVarP(&flagOutputFile, "output", "o", "", "write matches to a file instead of stdout")
	f.StringVar(&flagHTMLFile, "html", "", "write a beautiful HTML report to this file")

	// profile flag - writes .scope/cpu.pprof and .scope/mem.pprof
	f.BoolVar(&flagProfile, "profile", false, "write CPU + memory profiles to .scope/")

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
		Profile:      flagProfile,
		Quiet:        flagQuiet,
		Count:        flagCount,
		MaxResults:   flagMaxResults,
		OutputFile:   flagOutputFile,
		HTMLFile:     flagHTMLFile,
	}
	return search.Run(cfg)
}
