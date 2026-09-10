package search

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

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
