package output

import (
	"fmt"
	"strings"

	"github.com/Viswesh-G/scope/internal/metrics"
)

type ConsoleRenderer struct{}

func (ConsoleRenderer) Render(report metrics.Report) {
	fmt.Println()
	TitleColor.Println("--- Legend ---")
	fmt.Printf("  %s : %s : %s\n\n",
		FileColor.Sprint("File Path"),
		SuccessColor.Sprint("Line Number"),
		MatchColor.Sprint("Matched Text"),
	)

	TitleColor.Println("Scope Metrics")

	fmt.Printf("Dirs Scanned  : %d\n", report.DirsScanned)
	fmt.Printf("Files Scanned : %d\n", report.FilesScanned)
	fmt.Printf("Files Ignored : %d\n", report.FilesIgnored)
	fmt.Printf("Matches Found : %d\n", report.MatchesFound)
	TimeColor.Printf("Walk Time     : %v\n", report.WalkDuration)
	TimeColor.Printf("Search Time   : %v\n", report.SearchDuration)
	TimeColor.Printf("Total Time    : %v\n", report.TotalDuration)
	SuccessColor.Printf("Parallelism   : %.2fx\n", report.Parallelism)

	fmt.Println()
	SectionColor.Println("Worker Stats:")
	for _, worker := range report.Workers {
		pct := 0.0
		if report.FilesScanned > 0 {
			pct = float64(worker.FilesScanned) / float64(report.FilesScanned) * 100
		}
		barLength := int(pct / 100 * 20)
		bar := strings.Repeat("█", barLength) + strings.Repeat("░", 20-barLength)

		// Header row: bar + summary stats
		SuccessColor.Printf("  Worker %-2d  ", worker.ID)
		MatchColor.Printf("%s", bar)
		DimColor.Printf("  %3.0f%%  ", pct)
		TimeColor.Printf("work: %-10v  ", worker.WorkDuration)
		FileColor.Printf("files: %-2d  ", worker.FilesScanned)
		WarningColor.Printf("matches: %d\n", worker.MatchesFound)

		// Indent file list under each worker
		for _, f := range worker.Files {
			DimColor.Printf("             ↳ %s\n", f)
		}
	}
}

