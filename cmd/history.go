// This file defines the `scope history` command and all its subcommands.
//
// Every time you run `scope search`, the search details (pattern, path,
// duration, match count) are saved to .scope/history.json. The history
// commands let you explore that data: see recent searches, find the slowest
// ones, filter by pattern, export to JSON, and more.
//
// Subcommands:
//
//	scope history           — show all recent searches
//	scope history stats     — aggregate stats (total, avg time, fastest, slowest)
//	scope history top       — most frequently searched patterns
//	scope history slowest   — top 10 slowest searches
//	scope history fastest   — top 10 fastest searches
//	scope history recent    — N most recent searches
//	scope history pattern   — filter history by pattern text
//	scope history path      — filter history by search path
//	scope history export    — export history to a JSON file
//	scope history prune     — keep only the newest N records
//	scope history clear     — delete all history
package cmd

import (
	"fmt"
	"strconv"

	"github.com/Viswesh-G/scope/internal/analysis"
	"github.com/spf13/cobra"
)

// Flag values for history subcommands.
var (
	recentLimit int    // --limit : how many recent entries to show
	exportFile  string // -o / --output : file to write exported history to
	replayIndex int    // --nth : which entry to replay (1 = most recent)
)

// historyCmd is the parent `scope history` command.
// Running it with no subcommand shows a full chronological list.
var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "View and manage search history",
	Long: `Explore your past searches. Every scope search is recorded to .scope/history.json.

Run with no subcommand to see all recent searches in a table.
Use subcommands for filtering, statistics, and management.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return analysis.RunHistory()
	},
}

// historyStatsCmd shows aggregate metrics across all recorded searches.
var historyStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show aggregate statistics over all history",
	RunE: func(cmd *cobra.Command, args []string) error {
		return analysis.RunStats()
	},
}

// historyTopCmd lists the most frequently searched patterns.
var historyTopCmd = &cobra.Command{
	Use:   "top",
	Short: "Show most frequently searched patterns",
	RunE: func(cmd *cobra.Command, args []string) error {
		return analysis.RunTop()
	},
}

// historySlowestCmd shows the 10 slowest searches you have run.
var historySlowestCmd = &cobra.Command{
	Use:   "slowest",
	Short: "Show the 10 slowest searches",
	RunE: func(cmd *cobra.Command, args []string) error {
		return analysis.RunSlowest()
	},
}

// historyFastestCmd shows the 10 fastest searches you have run.
var historyFastestCmd = &cobra.Command{
	Use:   "fastest",
	Short: "Show the 10 fastest searches",
	RunE: func(cmd *cobra.Command, args []string) error {
		return analysis.RunFastest()
	},
}

// historyRecentCmd shows the N most recent searches (default: 10).
var historyRecentCmd = &cobra.Command{
	Use:   "recent",
	Short: "Show the most recent N searches",
	RunE: func(cmd *cobra.Command, args []string) error {
		return analysis.RunRecent(recentLimit)
	},
}

// historyPatternCmd filters history to only show entries matching a pattern string.
var historyPatternCmd = &cobra.Command{
	Use:   "pattern <text>",
	Short: "Filter history by pattern text",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return analysis.RunPattern(args[0])
	},
}

// historyPathCmd filters history to only show entries for a specific search path.
var historyPathCmd = &cobra.Command{
	Use:   "path <path>",
	Short: "Filter history by search path",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return analysis.RunPath(args[0])
	},
}

// historyExportCmd dumps the full history to a JSON file.
var historyExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export history to a JSON file",
	RunE: func(cmd *cobra.Command, args []string) error {
		return analysis.Export(exportFile)
	},
}

// historyPruneCmd trims history to keep only the newest N records.
// Useful for keeping the history file from growing unbounded.
var historyPruneCmd = &cobra.Command{
	Use:   "prune <count>",
	Short: "Keep only the newest N history entries",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid count %q: must be a whole number", args[0])
		}
		if limit <= 0 {
			return fmt.Errorf("count must be greater than 0")
		}
		return analysis.Prune(limit)
	},
}

// historyClearCmd deletes all stored search history.
var historyClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Delete all search history",
	RunE: func(cmd *cobra.Command, args []string) error {
		return analysis.Clear()
	},
}

// historyReplayCmd re-runs the most recent (or Nth) search from history.
// Useful for quickly repeating a past search without retyping the pattern.
var historyReplayCmd = &cobra.Command{
	Use:   "replay",
	Short: "Re-run the most recent (or Nth most recent) search",
	Long: `Re-runs a past search from your history.

By default replays the most recent search. Use --nth to pick an older one.
The replayed search IS saved to history again (as a new entry).

Examples:
  scope history replay          # re-run the last search
  scope history replay --nth 3  # re-run the 3rd most recent search`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return analysis.Replay(replayIndex)
	},
}

// init wires all history subcommands into the command tree and defines flags.
func init() {
	rootCmd.AddCommand(historyCmd)

	historyCmd.AddCommand(historyStatsCmd)
	historyCmd.AddCommand(historyTopCmd)
	historyCmd.AddCommand(historySlowestCmd)
	historyCmd.AddCommand(historyFastestCmd)
	historyCmd.AddCommand(historyRecentCmd)
	historyCmd.AddCommand(historyPatternCmd)
	historyCmd.AddCommand(historyPathCmd)
	historyCmd.AddCommand(historyExportCmd)
	historyCmd.AddCommand(historyPruneCmd)
	historyCmd.AddCommand(historyClearCmd)
	historyCmd.AddCommand(historyReplayCmd)

	// --limit flag for `scope history recent`
	historyRecentCmd.Flags().IntVarP(
		&recentLimit, "limit", "n", 10,
		"number of recent searches to show",
	)

	// -o / --output flag for `scope history export`
	historyExportCmd.Flags().StringVarP(
		&exportFile, "output", "o", "history_export.json",
		"output JSON file path",
	)

	// --nth flag for `scope history replay`
	historyReplayCmd.Flags().IntVar(
		&replayIndex, "nth", 1,
		"which past search to replay (1 = most recent)",
	)
}
