package cmd

import (
	"github.com/Viswesh-G/scope/internal/serve"
	"github.com/spf13/cobra"
)

var flagPort int

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Live Dashboard server",
	Long: `Start a local HTTP server that provides a live, interactive dashboard.
The dashboard automatically updates to show your search history and metrics.

Example:
  scp serve
  scp serve --port 3000`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return serve.StartServer(flagPort)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().IntVarP(&flagPort, "port", "P", 8080, "Port to run the dashboard on")
}
