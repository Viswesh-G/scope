package cmd

import (
	"fmt"
	"strconv"

	"github.com/Viswesh-G/scope/internal/analysis"
	"github.com/spf13/cobra"
)

var (
	recentLimit int
	exportFile  string
)

var historyCmd = &cobra.Command{
    Use:   "history",
    Short: "Search history and analytics",
    Long: `Inspect previous searches, performance statistics,
popular patterns, exports and history management.`,
    RunE: func(
        cmd *cobra.Command,
        args []string,
    ) error {
        return analysis.RunHistory()
    },
}
// =====================================================
// history stats
// =====================================================

var historyStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show search history statistics",
	RunE: func(
		cmd *cobra.Command,
		args []string,
	) error {

		return analysis.RunStats()
	},
}

// =====================================================
// history top
// =====================================================

var historyTopCmd = &cobra.Command{
	Use:   "top",
	Short: "Show most frequently searched patterns",
	RunE: func(
		cmd *cobra.Command,
		args []string,
	) error {

		return analysis.RunTop()
	},
}

// =====================================================
// history slowest
// =====================================================

var historySlowestCmd = &cobra.Command{
	Use:   "slowest",
	Short: "Show slowest searches",
	RunE: func(
		cmd *cobra.Command,
		args []string,
	) error {

		return analysis.RunSlowest()
	},
}

// =====================================================
// history fastest
// =====================================================

var historyFastestCmd = &cobra.Command{
	Use:   "fastest",
	Short: "Show fastest searches",
	RunE: func(
		cmd *cobra.Command,
		args []string,
	) error {

		return analysis.RunFastest()
	},
}

// =====================================================
// history recent
// =====================================================

var historyRecentCmd = &cobra.Command{
	Use:   "recent",
	Short: "Show most recent searches",
	RunE: func(
		cmd *cobra.Command,
		args []string,
	) error {

		return analysis.RunRecent(recentLimit)
	},
}

// =====================================================
// history pattern
// =====================================================

var historyPatternCmd = &cobra.Command{
	Use:   "pattern <text>",
	Short: "Filter history by pattern",
	Args:  cobra.ExactArgs(1),
	RunE: func(
		cmd *cobra.Command,
		args []string,
	) error {

		return analysis.RunPattern(args[0])
	},
}

// =====================================================
// history path
// =====================================================

var historyPathCmd = &cobra.Command{
	Use:   "path <path>",
	Short: "Filter history by search path",
	Args:  cobra.ExactArgs(1),
	RunE: func(
		cmd *cobra.Command,
		args []string,
	) error {

		return analysis.RunPath(args[0])
	},
}

// =====================================================
// history export
// =====================================================

var historyExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export history to JSON file",
	RunE: func(
		cmd *cobra.Command,
		args []string,
	) error {

		return analysis.Export(exportFile)
	},
}

// =====================================================
// history prune
// =====================================================

var historyPruneCmd = &cobra.Command{
	Use:   "prune <count>",
	Short: "Keep only the newest N history entries",
	Args:  cobra.ExactArgs(1),
	RunE: func(
		cmd *cobra.Command,
		args []string,
	) error {

		limit, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf(
				"invalid count %q",
				args[0],
			)
		}

		if limit <= 0 {
			return fmt.Errorf(
				"count must be > 0",
			)
		}

		return analysis.Prune(limit)
	},
}
// =====================================================
// history clear
// =====================================================

var historyClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Delete all search history",
	RunE: func(
		cmd *cobra.Command,
		args []string,
	) error {

		return analysis.Clear()
	},
}

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

	historyRecentCmd.Flags().IntVarP(
		&recentLimit,
		"limit",
		"n",
		10,
		"number of recent searches to show",
	)

	historyExportCmd.Flags().StringVarP(
		&exportFile,
		"output",
		"o",
		"history_export.json",
		"output JSON file",
	)

	
}