// internal/search/engine.go
package search

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Viswesh-G/scope/internal/metrics"
	"github.com/Viswesh-G/scope/internal/output"
)

func Run(cfg Config) error {
	registry := metrics.NewRegistry(cfg.Workers)
	totalStart := time.Now()

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
	ignore, err := LoadIgnoreFile(filepath.Join(cfg.Path, ".scope-ignore"))
	if err != nil {
		return fmt.Errorf("loading .scope-ignore: %w", err)
	}

	fileCh := make(chan string, cfg.Workers*4)
	matchCh := make(chan Match, 256)

	// Walker
	go func() {
		start := time.Now()
		walkFiles(cfg, ignore, fileCh, registry)
		registry.WalkDuration = time.Since(start)
	}()

	// Workers
	wg := startWorkers(cfg.Workers, re, fileCh, matchCh, registry)

	// Close match channel
	go func() {
		wg.Wait()
		close(matchCh)
	}()

	// Collector
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

	registry.TotalDuration = time.Since(totalStart)
	report := metrics.BuildReport(registry)

	renderer := output.ConsoleRenderer{}
	renderer.Render(report)

	return nil
}

// --------------------------------------------------------------------
// WALKER
// --------------------------------------------------------------------

func walkFiles(cfg Config, ignore *IgnoreMatcher, fileCh chan<- string, registry *metrics.Registry) {
	defer close(fileCh)

	_ = filepath.WalkDir(cfg.Path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			atomic.AddInt64(&registry.DirsScanned, 1)
			if ShouldSkipDir(d.Name(), ignore) {
				return filepath.SkipDir
			}
			if !cfg.Recursive && path != cfg.Path {
				return filepath.SkipDir
			}
			return nil
		}

		if ShouldSkipFile(path, ignore) {
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

func startWorkers(workers int, re *regexp.Regexp, fileCh <-chan string, matchCh chan<- Match, registry *metrics.Registry) *sync.WaitGroup {
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		workerID := i

		go func() {
			defer wg.Done()
			workerStats := registry.Workers[workerID]

			for path := range fileCh {
				atomic.AddInt64(&workerStats.FilesScanned, 1)

				workerStats.Mu.Lock()
				workerStats.Files = append(workerStats.Files, path)
				workerStats.Mu.Unlock()

				searchStart := time.Now()
				matches := searchFile(path, re)
				elapsed := time.Since(searchStart)

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

func searchFile(path string, re *regexp.Regexp) []Match {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var matches []Match
	lines := strings.Split(string(data), "\n")

	for linePos, line := range lines {
		if re.MatchString(line) {
			matches = append(matches, Match{
				File:    path,
				LineNum: linePos + 1,
				Line:    line,
			})
		}
	}

	return matches
}
