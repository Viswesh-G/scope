package cmd

import (
	"runtime"

	"github.com/Viswesh-G/scope/internal/search"
	"github.com/spf13/cobra"
)

var (
	flagPattern    string
	flagPath       string
	flagRecursive  bool
	flagWorkers    int
	flagFilenameOnly   bool
	flagHotspots       bool
	flagIgnoreCase bool
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search files for regex pattern",
	Long: `Search files for a specific regular expression pattern within a given directory.
By default, this will search recursively starting from the current directory, utilizing
all available CPU cores to speed up the process. It will automatically respect any
local '.scope-ignore' files as well as default ignores (like .git, node_modules).`,
	RunE: runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)

	f := searchCmd.Flags()

	f.StringVarP(&flagPattern, "pattern", "p", "", "regex pattern")
	f.BoolVarP(&flagFilenameOnly, "fname", "f", false, "filename to search for")
	f.StringVar(&flagPath, "path", ".", "search path")

	f.BoolVarP(
		&flagRecursive,
		"recursive",
		"r",
		true,
		"search recursively",
	)

	f.BoolVar(
		&flagHotspots,
		"hotspots",
		false,
		"find files with most matches",
	)

	f.BoolVarP(
		&flagIgnoreCase,
		"ignore-case",
		"i",
		false,
		"Case-insensitive search",
	)

	f.IntVarP(
		&flagWorkers,
		"workers",
		"w",
		runtime.NumCPU(),
		"number of workers",
	)

	_ = searchCmd.MarkFlagRequired("pattern")
}

func runSearch(cmd *cobra.Command, args []string) error {

	cfg := search.Config{
		Pattern:    flagPattern,
		Path:       flagPath,
		Recursive:  flagRecursive,
		Workers:    flagWorkers,
		FilenameOnly:   flagFilenameOnly,
		Hotspots:   flagHotspots,
		IgnoreCase: flagIgnoreCase,
	}

	return search.Run(cfg)
}
