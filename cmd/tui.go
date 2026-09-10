package cmd

import (
	"github.com/Viswesh-G/scope/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Start the interactive terminal UI",
	Long: `Launch the interactive bubbletea terminal user interface for SCP.
	
This interface lets you configure and run searches directly from a terminal dashboard
without having to remember CLI flags.

You can also pipe JSON results directly into the TUI to view them interactively:
  scp search -p "error" --json | scp tui`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.StartTUI()
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
