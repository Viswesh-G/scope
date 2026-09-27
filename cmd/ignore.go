// .scope-ignore works like .gitignore - one pattern per line, # for comments
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Viswesh-G/scope/internal/output"
	"github.com/spf13/cobra"
)

var ignoreCmd = &cobra.Command{
	Use:   "ignore",
	Short: "Manage .scope-ignore file",
	Long: `Manage the .scope-ignore file in the current directory.
Works like .gitignore: one pattern per line, # for comments, wildcards supported.`,
}

var ignoreInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a .scope-ignore with default patterns",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat(".scope-ignore"); err == nil {
			return fmt.Errorf(".scope-ignore already exists")
		}

		defaults := []string{
			"# scope-ignore - patterns to skip during searches",
			".git",
			"node_modules",
			"vendor",
			"build",
			"dist",
		}

		content := strings.Join(defaults, "\n") + "\n"
		if err := os.WriteFile(".scope-ignore", []byte(content), 0644); err != nil {
			return err
		}
		output.PrintSuccess("Created .scope-ignore with default patterns")
		return nil
	},
}

var ignoreAddCmd = &cobra.Command{
	Use:   "add [pattern]",
	Short: "Add a pattern to .scope-ignore",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// O_APPEND so we don't overwrite existing content, O_CREATE if file doesn't exist yet
		f, err := os.OpenFile(".scope-ignore", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()

		pattern := args[0]
		if _, err := f.WriteString(pattern + "\n"); err != nil {
			return err
		}
		output.PrintSuccess(fmt.Sprintf("Added %q to .scope-ignore", pattern))
		return nil
	},
}

var ignoreRemoveCmd = &cobra.Command{
	Use:   "remove [pattern]",
	Short: "Remove a pattern from .scope-ignore",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		b, err := os.ReadFile(".scope-ignore")
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf(".scope-ignore doesn't exist - run 'scope ignore init' first")
			}
			return err
		}

		pattern := args[0]
		lines := strings.Split(string(b), "\n")

		var newLines []string
		found := false
		for _, line := range lines {
			if strings.TrimSpace(line) == pattern {
				found = true
				continue
			}
			if line != "" || len(newLines) > 0 {
				newLines = append(newLines, line)
			}
		}

		if !found {
			output.PrintWarning(fmt.Sprintf("%q not found in .scope-ignore", pattern))
			return nil
		}

		if err := os.WriteFile(".scope-ignore", []byte(strings.Join(newLines, "\n")), 0644); err != nil {
			return err
		}
		output.PrintSuccess(fmt.Sprintf("Removed %q from .scope-ignore", pattern))
		return nil
	},
}

var ignoreListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all patterns in .scope-ignore",
	RunE: func(cmd *cobra.Command, args []string) error {
		b, err := os.ReadFile(".scope-ignore")
		if err != nil {
			if os.IsNotExist(err) {
				output.PrintWarning("No .scope-ignore found. Run 'scope ignore init' to create one.")
				return nil
			}
			return err
		}
		output.PrintHeader("Patterns in .scope-ignore", "")
		fmt.Print(string(b))
		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(ignoreCmd)
	ignoreCmd.AddCommand(ignoreInitCmd)
	ignoreCmd.AddCommand(ignoreAddCmd)
	ignoreCmd.AddCommand(ignoreRemoveCmd)
	ignoreCmd.AddCommand(ignoreListCmd)
}
