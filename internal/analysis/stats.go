// This file implements `scope stats` — counting files by extension.
package analysis

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Viswesh-G/scope/internal/ignore"
	"github.com/Viswesh-G/scope/internal/output"
)

// RunExtStats walks path and counts how many files have each file extension.
// Results are sorted by count (most common extensions first).
func RunExtStats(path string) error {
	ig, err := ignore.LoadIgnoreFile(filepath.Join(path, ".scope-ignore"))
	if err != nil {
		return fmt.Errorf("loading .scope-ignore: %w", err)
	}

	// Map from extension (e.g. ".go") to count.
	extCounts := make(map[string]int)

	_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if ignore.ShouldSkipDir(d.Name(), ig) {
				return filepath.SkipDir
			}
			return nil
		}
		if ignore.ShouldSkipFile(p, ig) {
			return nil
		}

		// Get the lowercase extension (e.g. ".GO" → ".go").
		ext := strings.ToLower(filepath.Ext(p))
		if ext == "" {
			ext = "no extension" // group extensionless files together
		}
		extCounts[ext]++
		return nil
	})

	// Sort by count descending, then alphabetically for ties.
	type extStat struct {
		ext   string
		count int
	}
	var stats []extStat
	for ext, count := range extCounts {
		stats = append(stats, extStat{ext, count})
	}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].count == stats[j].count {
			return stats[i].ext < stats[j].ext
		}
		return stats[i].count > stats[j].count
	})

	output.PrintHeader("File Extensions", "")

	var rows [][]string
	for _, s := range stats {
		// Grammatically correct label: "1 file" vs "N files".
		label := "files"
		if s.count == 1 {
			label = "file"
		}
		rows = append(rows, []string{s.ext, fmt.Sprintf("%d", s.count), label})
	}

	output.PrintTable(
		[]string{"Extension", "Count", ""},
		rows,
		[]string{"left", "right", "left"},
	)
	return nil
}
