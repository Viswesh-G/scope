// internal/output/base_color.go
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
	cName := viper.GetString("colors." + element)
	attr := MapColor(cName)
	if attr == 0 && len(defaultAttrs) > 0 {
		return color.New(defaultAttrs...)
	}

	var attrs []color.Attribute
	attrs = append(attrs, attr)
	for _, da := range defaultAttrs {
		if da == color.Bold {
			attrs = append(attrs, color.Bold)
		}
	}

	return color.New(attrs...)
}

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
