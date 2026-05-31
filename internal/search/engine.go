// internal/search/engine.go
package search

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Viswesh-G/scope/internal/ignore"
	"github.com/Viswesh-G/scope/internal/metrics"
	"github.com/Viswesh-G/scope/internal/output"
)

func Run(cfg Config) error {
	registry := metrics.NewRegistry(cfg.Workers)
	totalStart := time.Now()

	// Detect literal pattern (no regex metacharacters) for strings.Contains fast-path.
	// Must be checked before we prepend (?i), which is our own addition.
	literal := ""
	if isLiteralPattern(cfg.Pattern) {
		if cfg.IgnoreCase {
			literal = strings.ToLower(cfg.Pattern)
		} else {
			literal = cfg.Pattern
		}
	}

	// Ignore case support
	if cfg.IgnoreCase {
		cfg.Pattern = "(?i)" + cfg.Pattern
	}

	// Compile regex once
	re, err := regexp.Compile(cfg.Pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern %q: %w", cfg.Pattern, err)
	}

	// Load .scope-ignore
	ig, err := ignore.LoadIgnoreFile(filepath.Join(cfg.Path, ".scope-ignore"))
	if err != nil {
		return fmt.Errorf("loading .scope-ignore: %w", err)
	}

	fileCh := make(chan string, cfg.Workers*4)
	matchCh := make(chan Match, 256)

	// Walker
	go func() {
		start := time.Now()
		walkFiles(cfg, ig, fileCh, registry)
		registry.WalkDuration = time.Since(start)
	}()

	// Workers
	wg := startWorkers(cfg, re, literal, fileCh, matchCh, registry)

	// Close match channel
	go func() {
		wg.Wait()
		close(matchCh)
	}()

	// Collector
	if cfg.Hotspots {
		collectHotspots(matchCh)
	} else {
		for m := range matchCh {
			highlighted := re.ReplaceAllStringFunc(m.Line, func(match string) string {
				return output.MatchColor.Sprint(match)
			})
			fmt.Printf("%s:%s %s\n",
				output.FileColor.Sprint(m.File),
				output.SuccessColor.Sprint(m.LineNum),
				highlighted,
			)
		}
	}

	registry.TotalDuration = time.Since(totalStart)
	report := metrics.BuildReport(registry)

	renderer := output.ConsoleRenderer{}
	renderer.Render(report)

	return nil
}

// --------------------------------------------------------------------
// WALKER
// --------------------------------------------------------------------

func walkFiles(cfg Config, ig *ignore.IgnoreMatcher, fileCh chan<- string, registry *metrics.Registry) {
	defer close(fileCh)

	_ = filepath.WalkDir(cfg.Path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			atomic.AddInt64(&registry.DirsScanned, 1)
			if ignore.ShouldSkipDir(d.Name(), ig) {
				return filepath.SkipDir
			}
			if !cfg.Recursive && path != cfg.Path {
				return filepath.SkipDir
			}
			return nil
		}

		if ignore.ShouldSkipFile(path, ig) {
			atomic.AddInt64(&registry.FilesIgnored, 1)
			return nil
		}

		atomic.AddInt64(&registry.FilesScanned, 1)
		fileCh <- path
		return nil
	})
}

// --------------------------------------------------------------------
// WORKERS
// --------------------------------------------------------------------

func startWorkers(cfg Config, re *regexp.Regexp, literal string, fileCh <-chan string, matchCh chan<- Match, registry *metrics.Registry) *sync.WaitGroup {
	var wg sync.WaitGroup

	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		workerID := i

		go func() {
			defer wg.Done()
			workerStats := registry.Workers[workerID]

			// Build per-worker match function.
			// Literal path: strings.Contains (5–10× faster, no NFA overhead).
			// Regex path: re.Copy() gives each worker its own state machine.
			var matchFn func(string) bool
			if literal != "" {
				if cfg.IgnoreCase {
					matchFn = func(line string) bool {
						return strings.Contains(strings.ToLower(line), literal)
					}
				} else {
					matchFn = func(line string) bool {
						return strings.Contains(line, literal)
					}
				}
			} else {
				workerRe := re.Copy()
				matchFn = workerRe.MatchString
			}

			for path := range fileCh {
				atomic.AddInt64(&workerStats.FilesScanned, 1)

				workerStats.Mu.Lock()
				workerStats.Files = append(workerStats.Files, path)
				workerStats.Mu.Unlock()

				searchStart := time.Now()
				var bytesScanned int64
				var matches []Match
				if cfg.FilenameOnly {
					matches = searchFilename(path, matchFn)
				} else {
					bytesScanned, matches = searchFileWithData(path, matchFn)
				}
				elapsed := time.Since(searchStart)

				atomic.AddInt64(&workerStats.BytesScanned, bytesScanned)
				atomic.AddInt64(&workerStats.WorkTimeNs, elapsed.Nanoseconds())
				atomic.AddInt64(&registry.SearchTimeNs, elapsed.Nanoseconds())
				atomic.AddInt64(&registry.MatchesFound, int64(len(matches)))
				atomic.AddInt64(&workerStats.MatchesFound, int64(len(matches)))

				for _, m := range matches {
					matchCh <- m
				}
			}
		}()
	}

	return &wg
}

// --------------------------------------------------------------------
// FILE SEARCH
// --------------------------------------------------------------------

func searchFileWithData(path string, matchFn func(string) bool) (int64, []Match) {
	file, err := os.Open(path)
	if err != nil {
		return 0, nil
	}
	defer file.Close()

	var size int64
	if stat, err := file.Stat(); err == nil {
		size = stat.Size()
	}

	var matches []Match
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1<<20), 1<<20) // 1 MB cap: handles minified JS, long log lines

	lineNum := 1
	for scanner.Scan() {
		line := scanner.Text()
		if matchFn(line) {
			matches = append(matches, Match{
				File:    path,
				LineNum: lineNum,
				Line:    line,
			})
		}
		lineNum++
	}

	return size, matches
}

func searchFilename(path string, matchFn func(string) bool) []Match {
	name := filepath.Base(path)

	if !matchFn(name) {
		return nil
	}

	return []Match{
		{
			Mode: FilenameSearch,
			Line: name,
			File: path,
		},
	}
}

// --------------------------------------------------------------------
// HOTSPOTS
// --------------------------------------------------------------------

func collectHotspots(matchCh <-chan Match) {
	counts := make(map[string]int)
	for m := range matchCh {
		counts[m.File]++
	}

	type hotspot struct {
		file  string
		count int
	}
	var spots []hotspot
	for f, c := range counts {
		spots = append(spots, hotspot{f, c})
	}

	sort.Slice(spots, func(i, j int) bool {
		return spots[i].count > spots[j].count
	})

	fmt.Println()
	fmt.Println(output.TitleColor.Sprint("Hotspots"))
	fmt.Println(output.DimColor.Sprint("────────────────────"))
	fmt.Println()

	for _, s := range spots {
		fmt.Printf("%-30s %d matches\n", output.FileColor.Sprint(s.file), s.count)
	}
	fmt.Println()
}

// --------------------------------------------------------------------
// HELPERS
// --------------------------------------------------------------------

// isLiteralPattern returns true when s contains no regex metacharacters.
// When true, strings.Contains is used instead of the NFA engine — 5–10x faster
// for the most common search case (plain words, identifiers, etc.).
func isLiteralPattern(s string) bool {
	return regexp.QuoteMeta(s) == s
}