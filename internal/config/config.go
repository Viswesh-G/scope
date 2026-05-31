package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

func Init() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	viper.AddConfigPath(home)
	viper.SetConfigName(".scope-config")
	viper.SetConfigType("yaml")

	// Set defaults
	viper.SetDefault("colors.title", "cyan")
	viper.SetDefault("colors.section", "blue")
	viper.SetDefault("colors.success", "green")
	viper.SetDefault("colors.warning", "yellow")
	viper.SetDefault("colors.error", "red")
	viper.SetDefault("colors.dim", "faint")
	viper.SetDefault("colors.file", "hiwhite")
	viper.SetDefault("colors.match", "magenta")
	viper.SetDefault("colors.time", "higreen")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error and use defaults
		} else {
			return err
		}
	}

	return nil
}

func Save() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configPath := filepath.Join(home, ".scope-config.yaml")
	return viper.WriteConfigAs(configPath)
}

func SetColor(element, colorName string) error {
	validColors := GetValidColors()

	if !validColors[colorName] {
		return fmt.Errorf("invalid color: %s", colorName)
	}

	viper.Set(fmt.Sprintf("colors.%s", element), colorName)
	return Save()
}

func Reset() error {
	// Re-initialize defaults
	viper.Set("colors.title", "cyan")
	viper.Set("colors.section", "blue")
	viper.Set("colors.success", "green")
	viper.Set("colors.warning", "yellow")
	viper.Set("colors.error", "red")
	viper.Set("colors.dim", "faint")
	viper.Set("colors.file", "hiwhite")
	viper.Set("colors.match", "magenta")
	viper.Set("colors.time", "higreen")

	return Save()
}

func GetValidColors() map[string]bool {
	return map[string]bool{
		"black": true, "red": true, "green": true, "yellow": true,
		"blue": true, "magenta": true, "cyan": true, "white": true,
		"hiblack": true, "hired": true, "higreen": true, "hiyellow": true,
		"hiblue": true, "himagenta": true, "hicyan": true, "hiwhite": true,
		"faint": true, "bold": true,
	}
}

func GetCurrentColors() map[string]string {
	return map[string]string{
		"title":   viper.GetString("colors.title"),
		"section": viper.GetString("colors.section"),
		"success": viper.GetString("colors.success"),
		"warning": viper.GetString("colors.warning"),
		"error":   viper.GetString("colors.error"),
		"dim":     viper.GetString("colors.dim"),
		"file":    viper.GetString("colors.file"),
		"match":   viper.GetString("colors.match"),
		"time":    viper.GetString("colors.time"),
	}
}
