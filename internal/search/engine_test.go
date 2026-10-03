package search

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Viswesh-G/scope/internal/analysis"
	"github.com/Viswesh-G/scope/internal/output"
)

// the engine prints colored output, so we need the global color variables
// set up before any test calls Run(). normally cmd/root.go does this.
func TestMain(m *testing.M) {
	output.InitColors()
	os.Exit(m.Run())
}

func TestIsLiteralPattern(t *testing.T) {
	literal := []string{"TODO", "func main", "hello world"}
	for _, p := range literal {
		if !isLiteralPattern(p) {
			t.Errorf("%q should count as a literal pattern", p)
		}
	}
	regex := []string{"a.*b", "^func", "(foo|bar)", "[a-z]+", "v1.2"}
	for _, p := range regex {
		if isLiteralPattern(p) {
			t.Errorf("%q has metacharacters, should not be literal", p)
		}
	}
}

func TestSearchFileContents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.txt")
	content := "the quick brown fox\njumps over the lazy dog\nanother quick line\nnothing here\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	re := regexp.MustCompile(`quick`)
	bytesScanned, matches := searchFileContents(path, re.MatchString, 0, 0)

	if bytesScanned != int64(len(content)) {
		t.Errorf("bytes scanned = %d, want %d", bytesScanned, len(content))
	}
	if len(matches) != 2 {
		t.Fatalf("got %d matches, want 2: %+v", len(matches), matches)
	}
	if matches[0].LineNum != 1 || matches[1].LineNum != 3 {
		t.Errorf("line numbers = %d and %d, want 1 and 3", matches[0].LineNum, matches[1].LineNum)
	}
	for _, m := range matches {
		if m.File != path {
			t.Errorf("match file = %q, want %q", m.File, path)
		}
	}
}

func TestSearchFileContentsMissingFile(t *testing.T) {
	bytes, matches := searchFileContents("does/not/exist.txt", func(string) bool { return true }, 0, 0)
	if bytes != 0 || matches != nil {
		t.Error("a missing file should give 0 bytes and no matches, not an error")
	}
}

func TestSearchFilenameMatch(t *testing.T) {
	re := regexp.MustCompile(`engine`)
	matches := searchFilename("internal/search/engine.go", re.MatchString)
	if len(matches) != 1 {
		t.Fatalf("got %d matches, want 1", len(matches))
	}
	m := matches[0]
	if m.Mode != FilenameSearch {
		t.Errorf("mode = %v, want FilenameSearch", m.Mode)
	}
	if m.LineNum != 0 {
		t.Errorf("filename matches should have LineNum 0, got %d", m.LineNum)
	}
	if m.Line != "engine.go" {
		t.Errorf("matched name = %q, want engine.go", m.Line)
	}
}

func TestSearchFilenameNoMatch(t *testing.T) {
	re := regexp.MustCompile(`engine`)
	matches := searchFilename("cmd/root.go", re.MatchString)
	if len(matches) != 0 {
		t.Errorf("expected no matches, got %+v", matches)
	}
}

// full end-to-end run of the pipeline on a small temp directory.
// we write matches to an output file so we can check them without stdout noise.
func TestRunEndToEnd(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("fruits.txt", "apple pie\nbanana bread\napple juice\n")
	write("notes.md", "# shopping list\n- apples\n- pears\n")

	outPath := filepath.Join(dir, "results.txt")

	cfg := Config{
		Pattern:     "apple",
		Path:        dir,
		Recursive:   true,
		Workers:     2,
		Quiet:       true,
		SkipHistory: true,
		OutputFile:  outPath,
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	output := string(data)

	if !strings.Contains(output, "fruits.txt") || !strings.Contains(output, "notes.md") {
		t.Errorf("both files should appear in the results:\n%s", output)
	}
	if strings.Count(output, "apple") < 3 {
		t.Errorf("expected 3 lines containing 'apple':\n%s", output)
	}
}

func TestRunInvalidPattern(t *testing.T) {
	cfg := Config{
		Pattern:     "(unclosed",
		Path:        t.TempDir(),
		Workers:     1,
		SkipHistory: true,
		Quiet:       true,
	}
	if err := Run(cfg); err == nil {
		t.Error("an invalid regex should return an error")
	}
}

func TestRunNormalizesInvalidWorkerCount(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "input.txt")
	if err := os.WriteFile(path, []byte("target\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		Pattern: "target", Path: dir, Recursive: true, Workers: 0,
		Quiet: true, SkipHistory: true, OutputFile: filepath.Join(dir, "out.txt"),
	}
	if err := Run(cfg); err != nil {
		t.Fatalf("zero workers should be handled safely: %v", err)
	}
}

func TestRunReportsWalkErrors(t *testing.T) {
	cfg := Config{
		Pattern: "target", Path: filepath.Join(t.TempDir(), "missing"),
		Workers: 1, Quiet: true, SkipHistory: true,
	}
	if err := Run(cfg); err == nil {
		t.Fatal("expected an inaccessible search path to return an error")
	}
}

func TestRunRespectsMaxResults(t *testing.T) {
	dir := t.TempDir()
	content := strings.Repeat("needle here\n", 10)
	for i := 0; i < 5; i++ {
		name := filepath.Join(dir, "file"+string(rune('0'+i))+".txt")
		if err := os.WriteFile(name, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := Config{
		Pattern:     "needle",
		Path:        dir,
		Recursive:   true,
		Workers:     2,
		MaxResults:  3,
		Quiet:       true,
		SkipHistory: true,
		OutputFile:  filepath.Join(dir, "out.txt"),
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	data, err := os.ReadFile(cfg.OutputFile)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Count(string(data), "needle here")
	if got != 3 {
		t.Errorf("--max-results printed %d matches, want exactly 3", got)
	}
}

func TestRunIgnoresScopeIgnorePatterns(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".scope-ignore"), []byte("skipme.txt\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("target\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skipme.txt"), []byte("target\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Pattern:     "target",
		Path:        dir,
		Recursive:   true,
		Workers:     1,
		Quiet:       true,
		SkipHistory: true,
		OutputFile:  filepath.Join(dir, "out.txt"),
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	data, err := os.ReadFile(cfg.OutputFile)
	if err != nil {
		t.Fatal(err)
	}
	output := string(data)
	if !strings.Contains(output, "keep.txt") {
		t.Errorf("keep.txt should be searched:\n%s", output)
	}
	if strings.Contains(output, "skipme.txt") {
		t.Errorf("skipme.txt should have been ignored:\n%s", output)
	}
}

func TestContextLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "context.txt")
	content := "line 1\nline 2\ntarget\nline 4\nline 5\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		OriginalPattern: "target",
		Pattern:         "target",
		Path:            dir,
		Workers:         1,
		BeforeContext:   1,
		AfterContext:    1,
		Quiet:           true,
		SkipHistory:     true,
		OutputFile:      filepath.Join(dir, "out.txt"),
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	data, err := os.ReadFile(cfg.OutputFile)
	if err != nil {
		t.Fatal(err)
	}
	output := string(data)

	if !strings.Contains(output, "line 2") {
		t.Errorf("expected before context 'line 2', got:\n%s", output)
	}
	if !strings.Contains(output, "line 4") {
		t.Errorf("expected after context 'line 4', got:\n%s", output)
	}
}

func TestGlobFiltering(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "match.go"), []byte("target\n"), 0644)
	os.WriteFile(filepath.Join(dir, "skip.go"), []byte("target\n"), 0644)
	os.WriteFile(filepath.Join(dir, "match_test.go"), []byte("target\n"), 0644)
	os.WriteFile(filepath.Join(dir, "other.txt"), []byte("target\n"), 0644)

	cfg := Config{
		OriginalPattern: "target",
		Pattern:         "target",
		Path:            dir,
		Workers:         1,
		Globs:           []string{"*.go", "!skip.go"},
		Quiet:           true,
		SkipHistory:     true,
		OutputFile:      filepath.Join(dir, "out.txt"),
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	data, err := os.ReadFile(cfg.OutputFile)
	if err != nil {
		t.Fatal(err)
	}
	output := string(data)

	if !strings.Contains(output, "match.go") {
		t.Errorf("match.go should be included")
	}
	if !strings.Contains(output, "match_test.go") {
		t.Errorf("match_test.go should be included")
	}
	if strings.Contains(output, "skip.go") {
		t.Errorf("skip.go should be excluded by negative glob")
	}
	if strings.Contains(output, "other.txt") {
		t.Errorf("other.txt should be excluded as it does not match *.go")
	}
}

func TestBinaryFileSkip(t *testing.T) {
	dir := t.TempDir()
	txtPath := filepath.Join(dir, "text.txt")
	binPath := filepath.Join(dir, "bin.dat")

	os.WriteFile(txtPath, []byte("valid target here\n"), 0644)
	os.WriteFile(binPath, []byte("target \x00 some binary data\n"), 0644)

	cfg := Config{
		OriginalPattern: "target",
		Pattern:         "target",
		Path:            dir,
		Workers:         1,
		Quiet:           true,
		SkipHistory:     true,
		OutputFile:      filepath.Join(dir, "out.txt"),
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	data, err := os.ReadFile(cfg.OutputFile)
	if err != nil {
		t.Fatal(err)
	}
	output := string(data)

	if !strings.Contains(output, "text.txt") {
		t.Errorf("expected text.txt to be searched")
	}
	if strings.Contains(output, "bin.dat") {
		t.Errorf("expected bin.dat to be skipped as binary")
	}
}

func TestStdinSearch(t *testing.T) {
	// Temporarily replace os.Stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	dir := t.TempDir()
	stdinFile := filepath.Join(dir, "stdin_mock.txt")
	os.WriteFile(stdinFile, []byte("stdin target\n"), 0644)

	f, err := os.Open(stdinFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	os.Stdin = f

	cfg := Config{
		OriginalPattern: "target",
		Pattern:         "target",
		Path:            "-", // Signals stdin search
		Workers:         1,
		Quiet:           true,
		SkipHistory:     true,
		OutputFile:      filepath.Join(dir, "out.txt"),
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	data, err := os.ReadFile(cfg.OutputFile)
	if err != nil {
		t.Fatal(err)
	}
	output := string(data)

	if !strings.Contains(output, "<stdin>") {
		t.Errorf("expected <stdin> in output, got: %s", output)
	}
}

// makeManyFiles fills a temp directory with a lot of small files, so a search
// has enough work to still be running when we cancel it.
func makeManyFiles(t *testing.T, count int) string {
	t.Helper()
	dir := t.TempDir()
	content := strings.Repeat("needle on a line\n", 200)
	for i := 0; i < count; i++ {
		name := filepath.Join(dir, fmt.Sprintf("dir%02d", i%8), fmt.Sprintf("file%03d.txt", i))
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestRunStopsWhenContextIsCancelled(t *testing.T) {
	// Ctrl+C should end the search early, and Run should report that it was
	// cancelled rather than pretending everything completed.
	dir := makeManyFiles(t, 300)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel up front, so the search stops almost immediately

	cfg := Config{
		Context:     ctx,
		Pattern:     "needle",
		Path:        dir,
		Recursive:   true,
		Workers:     4,
		Quiet:       true,
		SkipHistory: true,
		OutputFile:  filepath.Join(t.TempDir(), "out.txt"),
	}

	err := Run(cfg)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run should return context.Canceled after an interrupt, got %v", err)
	}
}

func TestRunWithNilContextStillWorks(t *testing.T) {
	// Tests and the benchmark harness leave Config.Context nil, and that has
	// to keep working rather than panicking.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("needle\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Pattern:     "needle",
		Path:        dir,
		Recursive:   true,
		Workers:     1,
		Quiet:       true,
		SkipHistory: true,
		OutputFile:  filepath.Join(t.TempDir(), "out.txt"),
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run with no context should succeed, got %v", err)
	}

	data, err := os.ReadFile(cfg.OutputFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "needle") {
		t.Errorf("expected the match in the output file, got:\n%s", data)
	}
}

func TestRunCancelsMidwayAndKeepsEarlyMatches(t *testing.T) {
	// Cancel from another goroutine while the search is running. Whatever was
	// found before the cancel should still reach the output file, because
	// throwing away real results would be worse than stopping early.
	dir := makeManyFiles(t, 400)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		time.Sleep(15 * time.Millisecond)
		cancel()
	}()

	out := filepath.Join(t.TempDir(), "out.txt")
	cfg := Config{
		Context:     ctx,
		Pattern:     "needle",
		Path:        dir,
		Recursive:   true,
		Workers:     2,
		Quiet:       true,
		SkipHistory: true,
		OutputFile:  out,
	}

	// Either outcome is acceptable: it finished before the cancel landed, or
	// it stopped early. What must not happen is a hang or a panic.
	if err := Run(cfg); err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Run returned an unexpected error: %v", err)
	}

	// The output file is created even if the search was cancelled, and it
	// should contain valid text, not a half-written stream of nothing.
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("output file should exist even for a cancelled search: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if line != "" && !strings.Contains(line, "needle on a line") {
			t.Errorf("output file contains a line that is not a match: %q", line)
		}
	}
}

func TestRunStillRecordsHistoryAfterCancel(t *testing.T) {
	// We keep the record even though it was cut short, because the user did
	// run the search and "replay last search" should still work. The stored
	// numbers describe what was actually found before the cancel, so it is
	// not misleading.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("needle\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := Config{
		Context:    ctx,
		Pattern:    "needle",
		Path:       dir,
		Recursive:  true,
		Workers:    1,
		Quiet:      true,
		OutputFile: filepath.Join(t.TempDir(), "out.txt"),
	}
	// SkipHistory is left off on purpose here.
	_ = Run(cfg)

	records, err := analysis.Load()
	if err != nil {
		t.Fatalf("reading history failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("got %d history records, want 1", len(records))
	}
	if records[0].Pattern != "needle" {
		t.Errorf("history pattern = %q, want %q", records[0].Pattern, "needle")
	}
}
