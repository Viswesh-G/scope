// cmd/version.go prints the current version of scp so users can quickly check
// what they have installed. It follows the same pattern as other commands in this
// folder: define a cobra.Command, add it to the root command in init().
package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version is the current release tag. We use a simple string constant here
// so it's easy to update before cutting a release.
const Version = "v0.5.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the scp version",
	Long:  "Print the scp version number along with the Go version and OS/architecture it was built for.",
	Run: func(cmd *cobra.Command, args []string) {
		// runtime.Version() comes from the Go standard library and returns something like "go1.25.6".
		// runtime.GOOS and runtime.GOARCH tell us the operating system and CPU architecture.
		fmt.Printf("scp %s (built with %s on %s/%s)\n",
			Version,
			runtime.Version(),
			runtime.GOOS,
			runtime.GOARCH,
		)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
