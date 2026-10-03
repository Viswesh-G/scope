// cmd/version.go prints the current version of scope so users can quickly check
// what they have installed. It follows the same pattern as other commands in this
// folder: define a cobra.Command, add it to the root command in init().
package cmd

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// Version is set by GoReleaser for release binaries.
var Version = "dev"

func versionString() string {
	if Version != "dev" {
		return Version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return Version
	}
	return info.Main.Version
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the scope version",
	Long:  "Print the scope version number along with the Go version and OS/architecture it was built for.",
	Run: func(cmd *cobra.Command, args []string) {
		// runtime.Version() comes from the Go standard library and returns something like "go1.25.6".
		// runtime.GOOS and runtime.GOARCH tell us the operating system and CPU architecture.
		fmt.Printf("scope %s (built with %s on %s/%s)\n",
			versionString(),
			runtime.Version(),
			runtime.GOOS,
			runtime.GOARCH,
		)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
