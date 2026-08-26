// Global color variables used everywhere in the output package.
// All colors are user-configurable via `scope config set-color` and saved
// in ~/.scope-config.yaml. InitColors() reads the config and sets these up.
package output

import (
	"github.com/fatih/color"
	"github.com/spf13/viper"
)

var (
	TitleColor   *color.Color
	SectionColor *color.Color
	SuccessColor *color.Color
	WarningColor *color.Color
	ErrorColor   *color.Color
	DimColor     *color.Color
	FileColor    *color.Color
	MatchColor   *color.Color
	TimeColor    *color.Color
)

// InitColors reads user config from viper and sets all the color variables.
// Falls back to the hardcoded defaults if a config value is missing.
func InitColors() {
	TitleColor = getColor("title", color.FgCyan, color.Bold)
	SectionColor = getColor("section", color.FgBlue, color.Bold)
	SuccessColor = getColor("success", color.FgGreen)
	WarningColor = getColor("warning", color.FgYellow)
	ErrorColor = getColor("error", color.FgRed)
	DimColor = getColor("dim", color.Faint)
	FileColor = getColor("file", color.FgHiWhite)
	MatchColor = getColor("match", color.FgMagenta)
	TimeColor = getColor("time", color.FgHiGreen)
}

func getColor(element string, defaultAttrs ...color.Attribute) *color.Color {
	colorName := viper.GetString("colors." + element)
	attr := MapColor(colorName)

	if attr == 0 && len(defaultAttrs) > 0 {
		return color.New(defaultAttrs...)
	}

	// preserve Bold from the defaults even when the user picks a custom color
	// (so section headers stay bold regardless of color choice)
	var attrs []color.Attribute
	attrs = append(attrs, attr)
	for _, da := range defaultAttrs {
		if da == color.Bold {
			attrs = append(attrs, color.Bold)
		}
	}
	return color.New(attrs...)
}

// MapColor converts a config color name (like "cyan" or "higreen") to a
// color.Attribute. Returns 0 if the name isn't recognised.
func MapColor(name string) color.Attribute {
	switch name {
	case "black":
		return color.FgBlack
	case "red":
		return color.FgRed
	case "green":
		return color.FgGreen
	case "yellow":
		return color.FgYellow
	case "blue":
		return color.FgBlue
	case "magenta":
		return color.FgMagenta
	case "cyan":
		return color.FgCyan
	case "white":
		return color.FgWhite
	case "hiblack":
		return color.FgHiBlack
	case "hired":
		return color.FgHiRed
	case "higreen":
		return color.FgHiGreen
	case "hiyellow":
		return color.FgHiYellow
	case "hiblue":
		return color.FgHiBlue
	case "himagenta":
		return color.FgHiMagenta
	case "hicyan":
		return color.FgHiCyan
	case "hiwhite":
		return color.FgHiWhite
	case "faint":
		return color.Faint
	case "bold":
		return color.Bold
	default:
		return 0
	}
}
