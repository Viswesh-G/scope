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

	switch {
	case report.Parallelism < 1.2:
		WarningColor.Println("Concurrency: Low")
	case report.Parallelism < 3:
		SuccessColor.Println("Concurrency: Good")
	default:
		SuccessColor.Println("Concurrency: Excellent")
	}

	// ----------------------------------------------------------------
	// Per-worker stats
	// ----------------------------------------------------------------
	fmt.Println()
	SectionColor.Println("Worker Stats:")

	for _, worker := range report.Workers {
		pct := 0.0
		if report.FilesScanned > 0 {
			pct = float64(worker.FilesScanned) / float64(report.FilesScanned) * 100
		}
		barLength := int(pct / 100 * 20)
		bar := strings.Repeat("█", barLength) + strings.Repeat("░", 20-barLength)

		// Header row: bar + percentage
		SuccessColor.Printf("  Worker %-2d  ", worker.ID)
		MatchColor.Printf("%s", bar)
		DimColor.Printf("  %3.0f%% files scanned\n", pct)

		// Stat rows
		TimeColor.Printf("  work: %-10v  ", worker.WorkDuration)
		FileColor.Printf("files: %-3d  ", worker.FilesScanned)
		WarningColor.Printf("matches: %-3d  ", worker.MatchesFound)
		TitleColor.Printf("size: %s\n", formatBytes(worker.BytesScanned))

		if worker.WorkDuration > 0 {
			DimColor.Printf("  throughput: %s/s\n", formatBytes(int64(worker.Throughput)))
		}

		// Indent file list under each worker
		for _, f := range worker.Files {
			DimColor.Printf("             ↳ %s\n", f)
		}
	}

	// ----------------------------------------------------------------
	// Load Balance Report
	// ----------------------------------------------------------------
	b := report.Balance
	fmt.Println()
	SectionColor.Println("Load Balance")
	SectionColor.Println("────────────────────")
	fmt.Printf("Max Worker Load : %d files  /  %s\n", b.MaxFiles, formatBytes(b.MaxBytes))
	fmt.Printf("Min Worker Load : %d files  /  %s\n", b.MinFiles, formatBytes(b.MinBytes))
	fmt.Printf("Average Load    : %.1f files  /  %s\n", b.AvgFiles, formatBytes(int64(b.AvgBytes)))
	if b.Imbalance >= 3.0 {
		WarningColor.Printf("Imbalance       : %.2fx\n", b.Imbalance)
	} else {
		SuccessColor.Printf("Imbalance       : %.2fx\n", b.Imbalance)
	}

	// ----------------------------------------------------------------
	// Top Workers (ranked by bytes scanned)
	// ----------------------------------------------------------------
	fmt.Println()
	SectionColor.Println("Top Workers")
	SectionColor.Println("────────────────────")

	for rank, w := range b.Ranked {
		TitleColor.Printf("#%d Worker-%d\n", rank+1, w.ID)
		fmt.Printf("   Files     : %d\n", w.FilesScanned)
		fmt.Printf("   Size      : %s\n", formatBytes(w.BytesScanned))
		WarningColor.Printf("   Matches   : %d\n", w.MatchesFound)
		TimeColor.Printf("   Time      : %v\n", w.WorkDuration)
		if w.WorkDuration > 0 {
			DimColor.Printf("   Throughput: %s/s\n", formatBytes(int64(w.Throughput)))
		}
	}

	fmt.Println()
}

// formatBytes returns a human-readable byte size (B / KB / MB / GB).
func formatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.2f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.2f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
