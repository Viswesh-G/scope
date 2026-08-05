// Reusable print helpers used across all commands.
// All output should go through these so styling stays consistent.
package output

import (
	"fmt"
	"strings"
)

func PrintHeader(title, subtitle string) {
	fmt.Println()
	TitleColor.Println(title)
	DimColor.Println("────────────────────────────────────────────────────────────────────")
	if subtitle != "" {
		fmt.Println(subtitle)
	}
	fmt.Println()
}

func PrintSection(title string) {
	fmt.Println()
	SectionColor.Println(title)
	DimColor.Println("────────────────────")
	fmt.Println()
}

// PrintKeyValue prints a key: value pair with the key padded to a fixed width
// so values line up in a column.
func PrintKeyValue(key, value string) {
	fmt.Printf("%-20s : %s\n", key, value)
}

func PrintSuccess(msg string) { SuccessColor.Println("✓ " + msg) }
func PrintWarning(msg string) { WarningColor.Println("⚠ " + msg) }
func PrintError(err error)    { ErrorColor.Println("✗ Error: " + err.Error()) }
func PrintDivider()           { DimColor.Println("────────────────────────────────────────────────────────────────────") }

// PrintTable prints a table with auto-sized columns and optional alignment.
// alignments can be "left", "right", or "center" per column (defaults to "left").
func PrintTable(headers []string, rows [][]string, alignments []string) {
	if len(headers) == 0 || len(rows) == 0 {
		return
	}

	// figure out the max width needed for each column
	colWidths := make([]int, len(headers))
	for i, h := range headers {
		colWidths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// header row
	for i, h := range headers {
		fmt.Print(TitleColor.Sprint(alignString(h, colWidths[i], getAlignment(alignments, i))))
		if i < len(headers)-1 {
			DimColor.Print(" │ ")
		}
	}
	fmt.Println()

	// separator
	for i, w := range colWidths {
		DimColor.Print(strings.Repeat("─", w))
		if i < len(colWidths)-1 {
			DimColor.Print("─┼─")
		}
	}
	fmt.Println()

	// data rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(headers) {
				fmt.Print(alignString(cell, colWidths[i], getAlignment(alignments, i)))
				if i < len(headers)-1 {
					DimColor.Print(" │ ")
				}
			}
		}
		fmt.Println()
	}
	fmt.Println()
}

func getAlignment(alignments []string, index int) string {
	if index < len(alignments) {
		return alignments[index]
	}
	return "left"
}

func alignString(s string, width int, alignment string) string {
	padding := width - len(s)
	if padding <= 0 {
		return s
	}
	switch alignment {
	case "right":
		return strings.Repeat(" ", padding) + s
	case "center":
		left := padding / 2
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", padding-left)
	default:
		return s + strings.Repeat(" ", padding)
	}
}
