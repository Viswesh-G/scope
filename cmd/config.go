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
	Short: "Manage CLI configuration and looks",
	Long: `Manage your scope CLI configuration, including colors and aesthetic preferences.
Changes made here are saved globally to your ~/.scope-config.yaml file.`,
}

var configSetColorCmd = &cobra.Command{
	Use:   "set-color [element] [color]",
	Short: "Set a color for a specific output element",
	Long: `Set a color for a specific output element (e.g., title, success, match, etc).
For a list of all available elements and valid colors, run 'scope config list'.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		element := args[0]
		colorName := args[1]
		if err := config.SetColor(element, colorName); err != nil {
			return err
		}
		fmt.Printf("Successfully set %s color to %s\n", element, colorName)
		return nil
	},
}

var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset configuration to defaults",
	Long:  "Reset all personalized aesthetics and colors back to their original defaults.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Reset(); err != nil {
			return err
		}
		fmt.Println("Successfully reset configuration to defaults")
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all colors and configuration elements",
	Long:  "Display all available elements that can be styled, current color settings, and all valid colors.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("--- Current Configuration ---")
		current := config.GetCurrentColors()
		for k, v := range current {
			attr := output.MapColor(v)
			c := color.New(attr)
			fmt.Printf("  %s: %s\n", k, c.Sprint(v))
		}
		
		fmt.Println("\n--- Valid Colors ---")
		valid := config.GetValidColors()
		for k := range valid {
			attr := output.MapColor(k)
			c := color.New(attr)
			fmt.Printf("  %s\n", c.Sprint(k))
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetColorCmd)
	configCmd.AddCommand(configResetCmd)
	configCmd.AddCommand(configListCmd)
}
