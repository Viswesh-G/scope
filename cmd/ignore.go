package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var ignoreCmd = &cobra.Command{
	Use:   "ignore",
	Short: "Manage .scope-ignore file",
	Long: `Manage the .scope-ignore file in the current directory.
The ignore file is used to specify which directories or files should be skipped during a search.
Commands allow you to initialize, add, remove, or list patterns within this file.`,
}

var ignoreInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a .scope-ignore file with defaults",
	Long: `Creates a new .scope-ignore file in the current directory.
It automatically seeds the file with standard default directories to ignore,
such as .git, node_modules, vendor, build, and dist.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat(".scope-ignore"); err == nil {
			return fmt.Errorf(".scope-ignore already exists in the current directory")
		}

		defaults := []string{
			"# scope-ignore",
			".git",
			"node_modules",
			"vendor",
			"build",
			"dist",
		}

		if err := os.WriteFile(".scope-ignore", []byte(strings.Join(defaults, "\n")+"\n"), 0644); err != nil {
			return err
		}

		fmt.Println("Initialized .scope-ignore with defaults")
		return nil
	},
}

var ignoreAddCmd = &cobra.Command{
	Use:   "add [pattern]",
	Short: "Add a pattern to .scope-ignore",
	Long: `Add a specific file, directory, or wildcard pattern to the .scope-ignore file.
Examples:
  scope ignore add "build/"
  scope ignore add "*.log"`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		f, err := os.OpenFile(".scope-ignore", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()

		pattern := args[0]
		if _, err := f.WriteString(pattern + "\n"); err != nil {
			return err
		}

		fmt.Printf("Added %s to .scope-ignore\n", pattern)
		return nil
	},
}

var ignoreRemoveCmd = &cobra.Command{
	Use:   "remove [pattern]",
	Short: "Remove a pattern from .scope-ignore",
	Long:  "Safely removes a previously added pattern from the local .scope-ignore file.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		b, err := os.ReadFile(".scope-ignore")
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf(".scope-ignore does not exist")
			}
			return err
		}

		lines := strings.Split(string(b), "\n")
		var newLines []string
		found := false

		pattern := args[0]
		for _, line := range lines {
			if strings.TrimSpace(line) == pattern {
				found = true
				continue
			}
			if line != "" || len(newLines) > 0 { // keep non-empty, but avoid double empty line at end if we trim
				newLines = append(newLines, line)
			}
		}

		if !found {
			fmt.Printf("Pattern %s not found in .scope-ignore\n", pattern)
			return nil
		}

		if err := os.WriteFile(".scope-ignore", []byte(strings.Join(newLines, "\n")), 0644); err != nil {
			return err
		}

		fmt.Printf("Removed %s from .scope-ignore\n", pattern)
		return nil
	},
}

var ignoreListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all patterns in .scope-ignore",
	Long:  "Display all the ignore patterns currently configured in the local .scope-ignore file.",
	RunE: func(cmd *cobra.Command, args []string) error {
		b, err := os.ReadFile(".scope-ignore")
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No .scope-ignore found")
				return nil
			}
			return err
		}

		fmt.Println("Patterns in .scope-ignore:")
		fmt.Print(string(b))
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
