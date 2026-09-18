package search

// Benchmarks for the hot paths in the search engine.
// Run with: go test -bench=. -benchmem ./internal/search/
//
// The goal is to prove:
//   1. The literal fast-path (strings.Contains) is meaningfully faster than the regex engine.
//   2. Worker scaling gives real throughput gains up to the number of cores.
//   3. isBinaryFile is cheap enough to call on every file.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeLargeFile creates a temp file with `lines` lines of text.
// It returns the file path. The caller is responsible for cleanup.
func makeLargeFile(b *testing.B, lines int) string {
	b.Helper()
	dir := b.TempDir()
	path := filepath.Join(dir, "large.txt")

	// Each line is 80 chars. Every 10th line contains "needle" so we always get matches.
	var sb strings.Builder
	for i := 0; i < lines; i++ {
		if i%10 == 0 {
			sb.WriteString("this line contains needle and some padding text to fill it out properly\n")
		} else {
			sb.WriteString("this is a regular line of text without the special word, just filler data\n")
		}
	}

	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		b.Fatal(err)
	}
	return path
}

// BenchmarkLiteralSearch measures the literal fast-path (strings.Contains).
// isLiteralPattern("needle") == true, so the engine skips the regex entirely.
func BenchmarkLiteralSearch(b *testing.B) {
	dir := b.TempDir()
	// Write 20 files of 5000 lines each (~7MB total)
	for i := 0; i < 20; i++ {
		path := filepath.Join(dir, fmt.Sprintf("file%d.txt", i))
		var sb strings.Builder
		for j := 0; j < 5000; j++ {
			if j%10 == 0 {
				sb.WriteString("this line contains needle\n")
			} else {
				sb.WriteString("this is a regular line of text\n")
			}
		}
		if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
			b.Fatal(err)
		}
	}

	cfg := Config{
		OriginalPattern: "needle",
		Pattern:         "needle",
		Path:            dir,
		Recursive:       true,
		Workers:         4,
		Quiet:           true,
		SkipHistory:     true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := Run(cfg); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRegexSearch measures the full regex engine on the same workload.
// "need[l]e" has a character class, so isLiteralPattern returns false.
func BenchmarkRegexSearch(b *testing.B) {
	dir := b.TempDir()
	for i := 0; i < 20; i++ {
		path := filepath.Join(dir, fmt.Sprintf("file%d.txt", i))
		var sb strings.Builder
		for j := 0; j < 5000; j++ {
			if j%10 == 0 {
				sb.WriteString("this line contains needle\n")
			} else {
				sb.WriteString("this is a regular line of text\n")
			}
		}
		if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
			b.Fatal(err)
		}
	}

	cfg := Config{
		OriginalPattern: "need[l]e", // character class forces the regex engine
		Pattern:         "need[l]e",
		Path:            dir,
		Recursive:       true,
		Workers:         4,
		Quiet:           true,
		SkipHistory:     true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := Run(cfg); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkWorkerScaling runs the same search at 1, 2, 4, and 8 workers
// so you can see the actual speedup curve on your machine.
func BenchmarkWorkerScaling(b *testing.B) {
	dir := b.TempDir()
	for i := 0; i < 40; i++ {
		path := filepath.Join(dir, fmt.Sprintf("file%d.txt", i))
		var sb strings.Builder
		for j := 0; j < 2500; j++ {
			if j%10 == 0 {
				sb.WriteString("needle found here\n")
			} else {
				sb.WriteString("nothing to find on this line, just filler text\n")
			}
		}
		if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
			b.Fatal(err)
		}
	}

	for _, workers := range []int{1, 2, 4, 8} {
		b.Run(fmt.Sprintf("workers=%d", workers), func(b *testing.B) {
			cfg := Config{
				OriginalPattern: "needle",
				Pattern:         "needle",
				Path:            dir,
				Recursive:       true,
				Workers:         workers,
				Quiet:           true,
				SkipHistory:     true,
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := Run(cfg); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkSearchFileContents measures a single file scan, no concurrency.
// Useful for isolating just the I/O + matching cost.
func BenchmarkSearchFileContents(b *testing.B) {
	path := makeLargeFile(b, 10000)
	matchFn := func(line string) bool {
		return strings.Contains(line, "needle")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		searchFileContents(path, matchFn, 0, 0)
	}
}
