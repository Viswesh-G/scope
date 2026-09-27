// config subcommands - let users change the color of any output element.
// settings live in ~/.scope-config.yaml (managed by viper)
package cmd

import (
	"fmt"

	"github.com/Viswesh-G/scope/internal/config"
	"github.com/Viswesh-G/scope/internal/output"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration and appearance",
	Long: `Manage your scope CLI configuration, including colors and aesthetic preferences.
Changes are saved to ~/.scope-config.yaml and persist across sessions.`,
}

var configSetColorCmd = &cobra.Command{
	Use:   "set-color [element] [color]",
	Short: "Set the color of an output element",
	Long: `Set the color of a specific output element (e.g. title, match, file).
Run 'scope config list' to see all elements and valid color names.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		element, colorName := args[0], args[1]
		if err := config.SetColor(element, colorName); err != nil {
			return err
		}
		output.PrintSuccess(fmt.Sprintf("Set %s color to %s", element, colorName))
		return nil
	},
}

var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset all colors back to defaults",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Reset(); err != nil {
			return err
		}
		output.PrintSuccess("Configuration reset to defaults")
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show current colors and all valid color names",
	Run: func(cmd *cobra.Command, args []string) {
		output.PrintHeader("Current Configuration", "")
		for element, colorName := range config.GetCurrentColors() {
			// render each color name in its own color as a live preview
			c := color.New(output.MapColor(colorName))
			output.PrintKeyValue(element, c.Sprint(colorName))
		}

		output.PrintHeader("Valid Colors", "")
		for colorName := range config.GetValidColors() {
			c := color.New(output.MapColor(colorName))
			fmt.Printf("  %s\n", c.Sprint(colorName))
		}
		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetColorCmd)
	configCmd.AddCommand(configResetCmd)
	configCmd.AddCommand(configListCmd)
}
