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

func RunStats(path string) error {
	ig, err := ignore.LoadIgnoreFile(filepath.Join(path, ".scope-ignore"))
	if err != nil {
		return fmt.Errorf("loading .scope-ignore: %w", err)
	}

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

		ext := strings.ToLower(filepath.Ext(p))
		if ext == "" {
			ext = "no extension"
		}
		extCounts[ext]++

		return nil
	})

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

	fmt.Println()
	fmt.Println(output.TitleColor.Sprint("Extensions"))
	fmt.Println(output.DimColor.Sprint("────────────────────"))
	fmt.Println()

	for _, s := range stats {
		label := "files"
		if s.count == 1 {
			label = "file"
		}
		fmt.Printf("%-20s %d %s\n", output.FileColor.Sprint(s.ext), s.count, label)
	}
	fmt.Println()

	return nil
}
