package output

import (
	"fmt"
	"strings"
)

// PrintHeader prints a standardized top-level section header
func PrintHeader(title, subtitle string) {
	fmt.Println()
	TitleColor.Println(title)
	DimColor.Println("────────────────────────────────────────────────────────────────────")
	if subtitle != "" {
		fmt.Println(subtitle)
		fmt.Println()
	} else {
		fmt.Println()
	}
}

// PrintSection prints a standardized sub-section header
func PrintSection(title string) {
	fmt.Println()
	SectionColor.Println(title)
	DimColor.Println("────────────────────")
	fmt.Println()
}

// PrintKeyValue prints a standardized key-value pair, padding the key to a specific length
func PrintKeyValue(key, value string) {
	fmt.Printf("%-20s : %s\n", key, value)
}

// PrintSuccess prints a success message in standard coloring
func PrintSuccess(msg string) {
	SuccessColor.Println("✓ " + msg)
}

// PrintWarning prints a warning message in standard coloring
func PrintWarning(msg string) {
	WarningColor.Println("⚠ " + msg)
}

// PrintError prints an error message in standard coloring
func PrintError(err error) {
	ErrorColor.Println("✗ Error: " + err.Error())
}

// PrintDivider prints a reusable separator line
func PrintDivider() {
	DimColor.Println("────────────────────────────────────────────────────────────────────")
}

// PrintTable prints tabulated data with specified alignments (left, right, center)
func PrintTable(headers []string, rows [][]string, alignments []string) {
	if len(headers) == 0 || len(rows) == 0 {
		return
	}

	// Calculate max column widths
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

	// Print Headers
	for i, h := range headers {
		fmt.Print(TitleColor.Sprint(alignString(h, colWidths[i], getAlignment(alignments, i))))
		if i < len(headers)-1 {
			DimColor.Print(" │ ")
		}
	}
	fmt.Println()

	// Print Separator
	for i, w := range colWidths {
		DimColor.Print(strings.Repeat("─", w))
		if i < len(colWidths)-1 {
			DimColor.Print("─┼─")
		}
	}
	fmt.Println()

	// Print Rows
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
	return "left" // Default alignment
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
		leftPad := padding / 2
		rightPad := padding - leftPad
		return strings.Repeat(" ", leftPad) + s + strings.Repeat(" ", rightPad)
	default: // "left"
		return s + strings.Repeat(" ", padding)
	}
}
