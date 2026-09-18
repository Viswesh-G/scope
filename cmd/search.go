// the search command - parse flags, build a config, hand off to the engine
package cmd

import (
	"runtime"

	"github.com/Viswesh-G/scope/internal/search"
	"github.com/spf13/cobra"
)

// cobra fills these in from whatever the user typed on the command line
var (
	flagPattern       string
	flagPath          string
	flagRecursive     bool
	flagWorkers       int
	flagFilenameOnly  bool
	flagHotspots      bool
	flagIgnoreCase    bool
	flagNoHistory     bool
	flagNoConfig      bool
	flagProfile       bool
	flagQuiet         bool     // --quiet: skip the big metrics table
	flagCount         bool     // --count: print only the total match count
	flagMaxResults    int      // --max-results: stop after N matches
	flagOutputFile    string   // --output: write matches to a file instead of stdout
	flagHTMLFile      string   // --html: write a beautiful HTML report to this file
	flagJSONOutput    bool     // --json: emit matches as JSON array
	flagBeforeContext int      // -B: lines of leading context
	flagAfterContext  int      // -A: lines of trailing context
	flagContext       int      // -C: shorthand for setting both -A and -B to the same value
	flagGlobs         []string // -g: glob patterns to include/exclude files
	flagParallelProf  bool     // --parallel-profile: show worker execution timeline
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search files for a regex pattern",
	Long: `Search files for a regular expression pattern within a directory.

Searches recursively from the current directory by default, using all CPU cores.
Respects .scope-ignore and skips binaries, .git, node_modules, etc.

Examples:
  scp search -p "TODO"
  scp search -p "func main" --path ./cmd -i
  scp search -p "github" --hotspots
  scp search -p "TODO" --count
  scp search -p "error" --max-results 20
  scp search -p "TODO" -o matches.txt -q
  scp search -p "func" -A 2 -B 2
  scp search -p "func" -C 2
  scp search -p "TODO" -g "*.go" -g "!*_test.go"
  scp search -p "func" --json
  scp search -p "error" --path -`,
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

	// context lines (like grep -A / -B / -C)
	f.IntVarP(&flagAfterContext, "after-context", "A", 0, "lines of context after each match")
	f.IntVarP(&flagBeforeContext, "before-context", "B", 0, "lines of context before each match")
	f.IntVarP(&flagContext, "context", "C", 0, "lines of context on both sides of each match (sets -A and -B)")

	// glob filtering
	f.StringSliceVarP(&flagGlobs, "glob", "g", nil, "file glob patterns to include/exclude (e.g. *.go, !*_test.go)")

	// output control flags
	f.BoolVarP(&flagQuiet, "quiet", "q", false, "suppress the metrics table (just show matches)")
	f.BoolVar(&flagCount, "count", false, "print only the total match count, not each line")
	f.IntVarP(&flagMaxResults, "max-results", "m", 0, "stop after this many matches (0 = unlimited)")
	f.StringVarP(&flagOutputFile, "output", "o", "", "write matches to a file instead of stdout")
	f.StringVar(&flagHTMLFile, "html", "", "write a beautiful HTML report to this file")
	f.BoolVar(&flagJSONOutput, "json", false, "emit matches as a JSON array")

	// profile flag - writes .scope/cpu.pprof and .scope/mem.pprof
	f.BoolVar(&flagProfile, "profile", false, "write CPU + memory profiles to .scope/")

	// hidden flags used internally
	f.BoolVar(&flagNoHistory, "no-history", false, "skip saving to history")
	f.BoolVar(&flagNoConfig, "no-config", false, "skip loading config file")
	f.BoolVar(&flagParallelProf, "parallel-profile", false, "print ASCII timeline of parallel workers")

	_ = f.MarkHidden("no-history")
	_ = f.MarkHidden("no-config")
	_ = f.MarkHidden("parallel-profile")

	_ = searchCmd.MarkFlagRequired("pattern")
}

func runSearch(cmd *cobra.Command, args []string) error {
	// -C sets both sides to the same value, but explicit -A or -B take priority.
	// Only apply -C when the user didn't already set -A/-B manually.
	if flagContext > 0 {
		if !cmd.Flags().Changed("after-context") {
			flagAfterContext = flagContext
		}
		if !cmd.Flags().Changed("before-context") {
			flagBeforeContext = flagContext
		}
	}

	cfg := search.Config{
		// store the original pattern before the engine prepends (?i)
		OriginalPattern: flagPattern,
		Pattern:         flagPattern,
		Path:            flagPath,
		Recursive:       flagRecursive,
		Workers:         flagWorkers,
		FilenameOnly:    flagFilenameOnly,
		Hotspots:        flagHotspots,
		IgnoreCase:      flagIgnoreCase,
		SkipHistory:     flagNoHistory,
		Profile:         flagProfile,
		Quiet:           flagQuiet,
		Count:           flagCount,
		MaxResults:      flagMaxResults,
		OutputFile:      flagOutputFile,
		HTMLFile:        flagHTMLFile,
		JSONOutput:      flagJSONOutput,
		BeforeContext:   flagBeforeContext,
		AfterContext:    flagAfterContext,
		Globs:           flagGlobs,
		ParallelProfile: flagParallelProf,
	}
	return search.Run(cfg)
}
