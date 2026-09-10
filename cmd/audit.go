// scp audit chains several analysis commands together into one shot.
// Instead of running stats + deps + dupes separately, one command gives you
// a full health picture of the codebase: what files exist, what they depend on,
// and whether any files are accidentally duplicated.
package cmd

import (
	"github.com/Viswesh-G/scope/internal/analysis"
	"github.com/Viswesh-G/scope/internal/output"
	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Run a full codebase health check (stats + deps + dupes)",
	Long: `Run a full audit of the codebase in one command.

Combines three analysis passes:
  1. File extension stats     (scp stats)
  2. Go dependency counts     (scp deps)
  3. Duplicate file detection (scp dupes)

Useful as a single command to get a complete picture of the repo.

Examples:
  scp audit
  scp audit --path ./internal
  scp audit --skip-dupes    (skip the slower duplicate scan)`,
	RunE: runAudit,
}

var (
	auditPath      string
	auditSkipDupes bool
)

func init() {
	rootCmd.AddCommand(auditCmd)
	auditCmd.Flags().StringVar(&auditPath, "path", ".", "directory to audit")
	auditCmd.Flags().BoolVar(&auditSkipDupes, "skip-dupes", false, "skip the duplicate file scan")
}

func runAudit(cmd *cobra.Command, args []string) error {
	output.PrintHeader("Audit: "+auditPath, "")

	// 1. file extension stats
	output.SectionColor.Println("\n── File Stats ──")
	if err := analysis.RunExtStats(auditPath); err != nil {
		output.PrintWarning("stats failed: " + err.Error())
	}

	// 2. Go dependency counts
	output.SectionColor.Println("\n── Go Dependencies ──")
	if err := analysis.RunDeps(auditPath); err != nil {
		output.PrintWarning("deps failed: " + err.Error())
	}

	// 3. duplicate detection (can be slow on big repos, so it's skippable)
	if !auditSkipDupes {
		output.SectionColor.Println("\n── Duplicate Files ──")
		if err := analysis.RunDupes(auditPath, 4); err != nil {
			output.PrintWarning("dupes failed: " + err.Error())
		}
	}

	return nil
}
