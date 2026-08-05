// Package config manages user settings stored in ~/.scope-config.yaml.
// Right now the only setting is colors - one per output element.
// Using viper so we get YAML read/write for free.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// single source of truth for all default colors.
// both Init() and Reset() use this map - no duplication.
var defaultColors = map[string]string{
	"title":   "cyan",
	"section": "blue",
	"success": "green",
	"warning": "yellow",
	"error":   "red",
	"dim":     "faint",
	"file":    "hiwhite",
	"match":   "magenta",
	"time":    "higreen",
}

// Init loads ~/.scope-config.yaml (or just uses defaults if the file doesn't exist yet).
// Called once on startup before every command.
func Init() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	viper.AddConfigPath(home)
	viper.SetConfigName(".scope-config")
	viper.SetConfigType("yaml")

	for element, colorName := range defaultColors {
		viper.SetDefault("colors."+element, colorName)
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err // unexpected error - surface it
		}
		// file not found is fine - we'll just use defaults
	}

	return nil
}

func Save() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return viper.WriteConfigAs(filepath.Join(home, ".scope-config.yaml"))
}

func SetColor(element, colorName string) error {
	if !GetValidColors()[colorName] {
		return fmt.Errorf("invalid color %q - run 'scope config list' to see options", colorName)
	}
	viper.Set("colors."+element, colorName)
	return Save()
}

func Reset() error {
	for element, colorName := range defaultColors {
		viper.Set("colors."+element, colorName)
	}
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
	current := make(map[string]string, len(defaultColors))
	for element := range defaultColors {
		current[element] = viper.GetString("colors." + element)
	}
	return current
}
