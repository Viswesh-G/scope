// This is where the actual searching happens.
// The high-level flow: walker → fileCh → workers → matchCh → collector → output
package search

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Viswesh-G/scope/internal/analysis"
	"github.com/Viswesh-G/scope/internal/ignore"
	"github.com/Viswesh-G/scope/internal/metrics"
	"github.com/Viswesh-G/scope/internal/output"
	"github.com/Viswesh-G/scope/internal/profiler"
)

// JSONMatch is the struct we emit when --json is used.
// We keep it separate so the JSON keys are clean and user-facing.
type JSONMatch struct {
	File    string `json:"file"`
	LineNum int    `json:"line"`
	Content string `json:"content"`
}

func Run(cfg Config) error {
	// if --profile was passed, start collecting CPU samples right away.
	// the deferred Stop() will flush both cpu.pprof and mem.pprof when Run() returns.
	if cfg.Profile {
		session, err := profiler.Start(".scope")
		if err != nil {
			return fmt.Errorf("profiler: %w", err)
		}
		defer session.Stop()
		defer profiler.PrintSummary(".scope")
	}

	registry := metrics.NewRegistry(cfg.Workers)
	totalStart := time.Now()

	// -- literal fast-path --
	// if the pattern is plain text (no regex metacharacters like . * + etc),
	// strings.Contains is 5-10x faster than running the full regex engine.
	// check this BEFORE prepending (?i), which would break the literal check.
	literal := ""
	if isLiteralPattern(cfg.Pattern) {
		if cfg.IgnoreCase {
			literal = strings.ToLower(cfg.Pattern)
		} else {
			literal = cfg.Pattern
		}
	}

	if cfg.IgnoreCase {
		cfg.Pattern = "(?i)" + cfg.Pattern
	}

	// OriginalPattern is used when saving to history so the user never sees (?i) in their history.
	// If it wasn't set by the caller (e.g. in tests), fall back to the current pattern.
	if cfg.OriginalPattern == "" {
		cfg.OriginalPattern = cfg.Pattern
	}

	re, err := regexp.Compile(cfg.Pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern %q: %w", cfg.Pattern, err)
	}

	ig, err := ignore.LoadIgnoreFile(filepath.Join(cfg.Path, ".scope-ignore"))
	if err != nil {
		return fmt.Errorf("loading .scope-ignore: %w", err)
	}

	fileCh := make(chan string, cfg.Workers*4)
	matchCh := make(chan Match, 256)

	// stage 1: walk the filesystem, send paths into fileCh
	go func() {
		start := time.Now()
		walkFiles(cfg, ig, fileCh, registry)
		registry.WalkDuration = time.Since(start)
	}()

	// stage 2: workers pull paths from fileCh, scan files, push matches into matchCh
	wg := startWorkers(cfg, re, literal, fileCh, matchCh, registry, totalStart)

	// close matchCh once all workers are done so the collector knows to stop
	go func() {
		wg.Wait()
		close(matchCh)
	}()

	// stage 3: drain matchCh - collect or print each match
	//
	// where to write: stdout by default, or a file if --output was given
	var matchWriter io.Writer = os.Stdout
	var outFile *os.File
	if cfg.OutputFile != "" {
		f, err := os.Create(cfg.OutputFile)
		if err != nil {
			return fmt.Errorf("opening output file: %w", err)
		}
		defer f.Close()
		matchWriter = f
		outFile = f
	}
	_ = outFile // used indirectly through matchWriter

	var allMatches []output.HTMLMatch
	var jsonMatches []JSONMatch

	if cfg.Hotspots {
		collectHotspots(matchCh)
	} else if cfg.Count {
		// --count: just drain the channel silently, then print the total
		for range matchCh {
		}
		// the atomic counter in registry already tracked the total
	} else {
		collected := 0
		for m := range matchCh {
			if m.GroupSep {
				if !cfg.JSONOutput {
					fmt.Fprintln(matchWriter, output.DimColor.Sprint("--"))
				}
				continue
			}

			if cfg.HTMLFile != "" {
				allMatches = append(allMatches, output.HTMLMatch{
					File:    m.File,
					LineNum: m.LineNum,
					Line:    m.Line,
				})
			}

			if cfg.JSONOutput {
				// collect quietly - we print the JSON array at the end
				if !m.IsContext {
					jsonMatches = append(jsonMatches, JSONMatch{
						File:    m.File,
						LineNum: m.LineNum,
						Content: m.Line,
					})
					collected++
				}
			} else if m.IsContext {
				fmt.Fprintf(matchWriter, "%s-%s- %s\n",
					output.FileColor.Sprint(m.File),
					output.DimColor.Sprint(m.LineNum),
					m.Line,
				)
			} else {
				highlighted := re.ReplaceAllStringFunc(m.Line, func(match string) string {
					return output.MatchColor.Sprint(match)
				})
				fmt.Fprintf(matchWriter, "%s:%s %s\n",
					output.FileColor.Sprint(m.File),
					output.SuccessColor.Sprint(m.LineNum),
					highlighted,
				)
				collected++
			}

			// --max-results: stop printing after N matches but let workers finish
			if cfg.MaxResults > 0 && collected >= cfg.MaxResults {
				// drain the rest silently so the pipeline can shut down cleanly
				go func() {
					for range matchCh {
					}
				}()
				break
			}
		}
	}

	registry.TotalDuration = time.Since(totalStart)

	// --json: dump all collected matches as a JSON array
	if cfg.JSONOutput {
		enc := json.NewEncoder(matchWriter)
		enc.SetIndent("", "  ")
		_ = enc.Encode(jsonMatches)
	}

	// --count: print just the number, no big table
	if cfg.Count {
		fmt.Printf("%d matches\n", registry.MatchesFound)
	} else if !cfg.Quiet && !cfg.JSONOutput {
		// normal mode: print the full metrics table
		output.ConsoleRenderer{}.Render(metrics.BuildReport(registry))

		// If the user wants to see the parallel profile, draw it now!
		if cfg.ParallelProfile {
			renderParallelProfile(registry)
		}
	}

	if cfg.HTMLFile != "" {
		report := metrics.BuildReport(registry)
		err := output.WriteHTMLReport(cfg.HTMLFile, report, allMatches)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to write HTML report: %v\n", err)
		} else {
			fmt.Println()
			output.SuccessColor.Printf("✨ HTML report saved to %s\n", cfg.HTMLFile)
		}
	}

	if !cfg.SkipHistory {
		analysis.Save(analysis.SearchRecord{
			Timestamp:  time.Now().Format(time.RFC3339),
			Pattern:    cfg.OriginalPattern, // save the original, not the (?i)-prefixed version
			Path:       cfg.Path,
			Workers:    cfg.Workers,
			Matches:    registry.MatchesFound,
			DurationMs: registry.TotalDuration.Seconds() * 1000,
		})
	}

	return nil
}

// -- walker --

func walkFiles(cfg Config, ig *ignore.IgnoreMatcher, fileCh chan<- string, registry *metrics.Registry) {
	defer close(fileCh)

	// If the path is "-", we read directly from standard input instead of directories
	if cfg.Path == "-" {
		atomic.AddInt64(&registry.FilesScanned, 1)
		fileCh <- "<stdin>"
		return
	}

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

		// Apply glob filtering if --glob was specified
		if len(cfg.Globs) > 0 && !matchesGlobs(path, cfg.Globs) {
			atomic.AddInt64(&registry.FilesIgnored, 1)
			return nil
		}

		atomic.AddInt64(&registry.FilesScanned, 1)
		fileCh <- path
		return nil
	})
}

// -- workers --

func startWorkers(cfg Config, re *regexp.Regexp, literal string, fileCh <-chan string, matchCh chan<- Match, registry *metrics.Registry, totalStart time.Time) *sync.WaitGroup {
	var wg sync.WaitGroup

	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		workerID := i

		go func() {
			defer wg.Done()
			workerStats := registry.Workers[workerID]

			// each worker gets its own match function.
			// for plain text: strings.Contains (fast, no regex overhead).
			// for regex: re.Copy() gives each goroutine its own state machine
			// to avoid data races on the internal regex state.
			var matchFn func(string) bool
			if literal != "" {
				if cfg.IgnoreCase {
					matchFn = func(line string) bool {
						return strings.Contains(strings.ToLower(line), literal)
					}
				} else {
					matchFn = func(line string) bool { return strings.Contains(line, literal) }
				}
			} else {
				workerRe := re.Copy()
				matchFn = workerRe.MatchString
			}

			for path := range fileCh {
				atomic.AddInt64(&workerStats.FilesScanned, 1)

				// Find out exactly when we started processing this file relative to the overall start time
				fileStart := time.Now()
				startOffsetNs := fileStart.Sub(totalStart).Nanoseconds()

				// (Binary files are skipped directly in searchFileContents now)

				var bytesScanned int64
				var matches []Match

				if cfg.FilenameOnly {
					matches = searchFilename(path, matchFn)
				} else {
					bytesScanned, matches = searchFileContents(path, matchFn, cfg.BeforeContext, cfg.AfterContext)
				}

				elapsed := time.Since(fileStart)

				// We lock the mutex so we can safely add to the slices without data races
				workerStats.Mu.Lock()
				workerStats.Files = append(workerStats.Files, path)
				workerStats.Events = append(workerStats.Events, metrics.WorkerEvent{
					StartOffsetNs: startOffsetNs,
					DurationNs:    elapsed.Nanoseconds(),
				})
				workerStats.Mu.Unlock()

				// Calculate the actual match count, excluding context lines and separators
				var actualMatchCount int64
				for _, m := range matches {
					if !m.IsContext && !m.GroupSep {
						actualMatchCount++
					}
				}

				// all these counters are shared across goroutines,
				// so we use atomic operations instead of a mutex
				atomic.AddInt64(&workerStats.BytesScanned, bytesScanned)
				atomic.AddInt64(&workerStats.WorkTimeNs, elapsed.Nanoseconds())
				atomic.AddInt64(&registry.SearchTimeNs, elapsed.Nanoseconds())
				atomic.AddInt64(&registry.MatchesFound, actualMatchCount)
				atomic.AddInt64(&workerStats.MatchesFound, actualMatchCount)

				for _, m := range matches {
					matchCh <- m
				}
			}
		}()
	}

	return &wg
}

// -- file scanning --

func searchFileContents(path string, matchFn func(string) bool, before, after int) (int64, []Match) {
	var r io.Reader
	var size int64

	if path == "<stdin>" {
		r = os.Stdin
	} else {
		file, err := os.Open(path)
		if err != nil {
			return 0, nil
		}
		defer file.Close()

		if stat, err := file.Stat(); err == nil {
			size = stat.Size()
		}

		// check binary in the first 1024 bytes
		buf := make([]byte, 1024)
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return size, nil
		}
		for i := 0; i < n; i++ {
			if buf[i] == 0x00 {
				// is binary, skip
				return size, nil
			}
		}

		// stitch back the chunk
		r = io.MultiReader(strings.NewReader(string(buf[:n])), file)
	}

	var matches []Match
	var window []Match // keeps up to `before` lines
	var afterRemaining int
	lastPrintedIdx := -1

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1<<20), 1<<20) // 1MB buffer

	lineNum := 1
	for scanner.Scan() {
		line := scanner.Text()
		isMatch := matchFn(line)

		if isMatch {
			// Find the first line in the before-window that we haven't printed yet
			var firstToPrint *Match
			for i := range window {
				if window[i].LineNum > lastPrintedIdx {
					firstToPrint = &window[i]
					break
				}
			}

			var nextLineNum int
			if firstToPrint != nil {
				nextLineNum = firstToPrint.LineNum
			} else {
				nextLineNum = lineNum
			}

			// If there's a gap between the last printed line and the next one we print, insert separator
			// (Only do this if context lines are enabled)
			if (before > 0 || after > 0) && lastPrintedIdx != -1 && nextLineNum > lastPrintedIdx+1 {
				matches = append(matches, Match{Mode: ContentSearch, File: path, GroupSep: true})
			}

			// Output the before context
			for _, m := range window {
				if m.LineNum > lastPrintedIdx {
					m.IsContext = true
					matches = append(matches, m)
					lastPrintedIdx = m.LineNum
				}
			}

			// Output the match itself
			matches = append(matches, Match{Mode: ContentSearch, File: path, LineNum: lineNum, Line: line})
			lastPrintedIdx = lineNum
			afterRemaining = after

		} else {
			if afterRemaining > 0 {
				matches = append(matches, Match{Mode: ContentSearch, File: path, LineNum: lineNum, Line: line, IsContext: true})
				lastPrintedIdx = lineNum
				afterRemaining--
			}
		}

		// Keep the sliding window updated
		if before > 0 {
			window = append(window, Match{Mode: ContentSearch, File: path, LineNum: lineNum, Line: line})
			if len(window) > before {
				window = window[1:]
			}
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
	return []Match{{Mode: FilenameSearch, File: path, Line: name}}
}

// -- hotspots --

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
	sort.Slice(spots, func(i, j int) bool { return spots[i].count > spots[j].count })

	fmt.Println()
	fmt.Println(output.TitleColor.Sprint("Hotspots"))
	fmt.Println(output.DimColor.Sprint("────────────────────"))
	fmt.Println()
	for _, s := range spots {
		fmt.Printf("%-30s %d matches\n", output.FileColor.Sprint(s.file), s.count)
	}
	fmt.Println()
}

// -- helpers --

// matchesGlobs checks if the filename of a path matches any glob patterns
// supporting standard negative globs (prefixed with "!") and positive patterns.
func matchesGlobs(path string, globs []string) bool {
	if len(globs) == 0 {
		return true
	}
	base := filepath.Base(path)
	hasPositive := false
	for _, g := range globs {
		if !strings.HasPrefix(g, "!") {
			hasPositive = true
			break
		}
	}

	// First verify negative patterns
	for _, g := range globs {
		if strings.HasPrefix(g, "!") {
			pattern := g[1:]
			if matched, _ := filepath.Match(pattern, base); matched {
				return false
			}
		}
	}

	// Next, verify positive patterns if they exist
	if hasPositive {
		for _, g := range globs {
			if !strings.HasPrefix(g, "!") {
				if matched, _ := filepath.Match(g, base); matched {
					return true
				}
			}
		}
		return false
	}

	return true
}


// isLiteralPattern returns true if the string has no regex special characters.
// trick: regexp.QuoteMeta escapes everything, so if the result equals the input,
// there was nothing to escape.
func isLiteralPattern(s string) bool {
	return regexp.QuoteMeta(s) == s
}

// renderParallelProfile draws a Gantt-style ASCII chart showing when each worker was active.
// It is intended for students to visually understand parallel processing!
func renderParallelProfile(registry *metrics.Registry) {
	fmt.Println()
	fmt.Println(output.TitleColor.Sprint("Parallel Execution Profile"))
	fmt.Println(output.DimColor.Sprint("──────────────────────────"))

	totalTimeNs := registry.TotalDuration.Nanoseconds()
	if totalTimeNs <= 0 {
		fmt.Println("Search was too fast to profile!")
		return
	}

	// We'll use 50 columns for our timeline visualization
	const columns = 50
	nsPerColumn := float64(totalTimeNs) / float64(columns)

	for _, w := range registry.Workers {
		w.Mu.Lock() // lock while reading the events
		
		// Create an empty timeline string of spaces
		timeline := make([]rune, columns)
		for i := range timeline {
			timeline[i] = ' '
		}

		// Fill in the timeline where the worker was active
		for _, ev := range w.Events {
			startCol := int(float64(ev.StartOffsetNs) / nsPerColumn)
			endCol := int(float64(ev.StartOffsetNs+ev.DurationNs) / nsPerColumn)

			// bounds check
			if startCol < 0 { startCol = 0 }
			if endCol >= columns { endCol = columns - 1 }
			if startCol >= columns { startCol = columns - 1 }

			for i := startCol; i <= endCol; i++ {
				timeline[i] = '█' // Use a block character to show activity
			}
		}

		w.Mu.Unlock()

		// Print the worker ID and its visual timeline
		workerLabel := fmt.Sprintf("Worker %2d", w.ID)
		fmt.Printf("%s │ %s │\n", output.TitleColor.Sprint(workerLabel), string(timeline))
	}
	fmt.Println(output.DimColor.Sprint("          └" + strings.Repeat("─", columns+2) + "┘"))
	fmt.Println(output.DimColor.Sprint("           0ms" + strings.Repeat(" ", columns-6) + fmt.Sprintf("%dms", registry.TotalDuration.Milliseconds())))
	fmt.Println()
}
