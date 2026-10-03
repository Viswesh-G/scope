// Integration tests that run the real scope binary as a separate process.
//
// The unit tests in internal/ call the functions directly, which is quick but
// it never checks the things that only go wrong in a real process: flag
// parsing, exit codes, what actually lands on stdout, and whether two
// processes writing history at the same time lose data.
//
// So these tests compile the binary once and then run it.
package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Viswesh-G/scope/internal/analysis"
)

var (
	buildOnce   sync.Once
	builtBinary string
	buildErr    error
)

// scopeBin compiles the binary the first time a test needs it and reuses it
// for the rest, because building takes longer than most of these tests.
func scopeBin(t *testing.T) string {
	t.Helper()

	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "scope-cli-test")
		if err != nil {
			buildErr = err
			return
		}

		name := "scope-under-test"
		if os.PathSeparator == '\\' {
			name += ".exe"
		}
		builtBinary = filepath.Join(dir, name)

		// These tests live in cmd/, so the module root is one folder up.
		cmd := exec.Command("go", "build", "-o", builtBinary, ".")
		cmd.Dir = ".."
		if out, err := cmd.CombinedOutput(); err != nil {
			buildErr = err
			t.Logf("build output:\n%s", out)
		}
	})

	if buildErr != nil {
		t.Fatalf("could not build the scope binary: %v", buildErr)
	}
	return builtBinary
}

// result holds what a finished command produced, so tests can check the output
// and the exit code without repeating the plumbing.
type result struct {
	stdout   string
	stderr   string
	exitCode int
}

// run executes scope in dir and captures both streams.
func run(t *testing.T, dir string, stdin string, args ...string) result {
	t.Helper()

	cmd := exec.Command(scopeBin(t), args...)
	cmd.Dir = dir
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	var out, errOut strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	err := cmd.Run()

	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if ok := asExitError(err, &exitErr); ok {
			code = exitErr.ExitCode()
		} else {
			t.Fatalf("could not run scope: %v", err)
		}
	}

	return result{stdout: out.String(), stderr: errOut.String(), exitCode: code}
}

// asExitError is errors.As for exec.ExitError, kept separate so run() reads
// cleanly.
func asExitError(err error, target **exec.ExitError) bool {
	e, ok := err.(*exec.ExitError)
	if ok {
		*target = e
	}
	return ok
}

// sampleRepo makes a tiny repository for the tests to search.
func sampleRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	files := map[string]string{
		"main.go":                 "package main\n\nfunc main() {\n\t// TODO: wire this up\n}\n",
		"internal/util.go":        "package internal\n\n// Helper does work.\nfunc Helper() {}\n",
		"notes.md":                "# notes\n\nTODO: write the docs\n",
		"internal/helper_test.go": "package internal\n\nfunc TestHelper() {}\n",
	}
	for name, body := range files {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestCLIVersionPrintsSomething(t *testing.T) {
	got := run(t, t.TempDir(), "", "version")

	if got.exitCode != 0 {
		t.Fatalf("version exited with %d\nstderr: %s", got.exitCode, got.stderr)
	}
	if !strings.Contains(got.stdout, "scope") {
		t.Errorf("version output should name the tool, got:\n%s", got.stdout)
	}
}

func TestCLISearchFindsMatches(t *testing.T) {
	dir := sampleRepo(t)

	got := run(t, dir, "", "search", "-p", "TODO", "-q")

	if got.exitCode != 0 {
		t.Fatalf("search exited with %d\nstderr: %s", got.exitCode, got.stderr)
	}
	if !strings.Contains(got.stdout, "TODO") {
		t.Errorf("expected TODO matches on stdout, got:\n%s", got.stdout)
	}
	// -q means no metrics table, so "Scope Metrics" must not show up.
	if strings.Contains(got.stdout, "Scope Metrics") {
		t.Errorf("--quiet should suppress the metrics table, got:\n%s", got.stdout)
	}
}

func TestCLISearchJSONIsValid(t *testing.T) {
	// This is the contract other tools rely on when they pipe scope output
	// into jq, so the JSON has to parse and the fields have to be right.
	dir := sampleRepo(t)

	got := run(t, dir, "", "search", "-p", "TODO", "--json", "-q")

	if got.exitCode != 0 {
		t.Fatalf("search exited with %d\nstderr: %s", got.exitCode, got.stderr)
	}

	var matches []struct {
		File    string `json:"file"`
		LineNum int    `json:"line"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(got.stdout), &matches); err != nil {
		t.Fatalf("stdout was not a valid JSON array: %v\noutput:\n%s", err, got.stdout)
	}
	if len(matches) == 0 {
		t.Fatal("expected at least one match in the JSON output")
	}
	for _, m := range matches {
		if m.File == "" || m.LineNum < 1 || !strings.Contains(m.Content, "TODO") {
			t.Errorf("suspicious match entry: %+v", m)
		}
	}
}

func TestCLISearchJSONWithNoMatchesIsEmptyArray(t *testing.T) {
	// An empty result has to be `[]`, not `null`. jq and most other tools
	// treat null as a missing field and fall over.
	dir := sampleRepo(t)

	got := run(t, dir, "", "search", "-p", "this-string-is-not-in-the-repo", "--json", "-q")

	if got.exitCode != 0 {
		t.Fatalf("search exited with %d\nstderr: %s", got.exitCode, got.stderr)
	}
	if strings.TrimSpace(got.stdout) != "[]" {
		t.Errorf("expected [] for no matches, got:\n%s", got.stdout)
	}
}

func TestCLISearchReadsStandardInput(t *testing.T) {
	dir := t.TempDir()
	log := "starting up\nFATAL could not connect\nretrying\nFATAL gave up\n"

	got := run(t, dir, log, "search", "-p", "FATAL", "--path", "-", "-q")

	if got.exitCode != 0 {
		t.Fatalf("stdin search exited with %d\nstderr: %s", got.exitCode, got.stderr)
	}
	if strings.Count(got.stdout, "FATAL") != 2 {
		t.Errorf("expected 2 FATAL lines on stdout, got:\n%s", got.stdout)
	}
}

func TestCLIInvalidRegexFailsCleanly(t *testing.T) {
	// A broken pattern should produce a friendly message and a non-zero exit
	// code, not a panic or a Go stack trace.
	got := run(t, t.TempDir(), "", "search", "-p", "(unclosed", "-q")

	if got.exitCode == 0 {
		t.Error("an invalid regex should exit with a non-zero code")
	}
	if !strings.Contains(got.stderr, "invalid pattern") {
		t.Errorf("stderr should explain the pattern is invalid, got:\n%s", got.stderr)
	}
	if strings.Contains(got.stderr, "panic:") {
		t.Errorf("an invalid pattern must not panic, got:\n%s", got.stderr)
	}
}

func TestCLIMissingPathFailsCleanly(t *testing.T) {
	got := run(t, t.TempDir(), "", "search", "-p", "x", "--path", "no/such/dir", "-q")

	if got.exitCode == 0 {
		t.Error("searching a directory that does not exist should fail")
	}
	if got.stderr == "" {
		t.Error("expected an explanation on stderr")
	}
}

func TestCLIMissingPatternFails(t *testing.T) {
	// --pattern is marked required, so cobra should stop us before we start.
	got := run(t, t.TempDir(), "", "search")

	if got.exitCode == 0 {
		t.Error("search without --pattern should fail")
	}
}

func TestCLIGlobFiltering(t *testing.T) {
	dir := sampleRepo(t)

	got := run(t, dir, "", "search", "-p", "TODO", "-g", "*.md", "-q")

	if got.exitCode != 0 {
		t.Fatalf("search exited with %d\nstderr: %s", got.exitCode, got.stderr)
	}
	if !strings.Contains(got.stdout, "notes.md") {
		t.Errorf("expected notes.md in the output, got:\n%s", got.stdout)
	}
	if strings.Contains(got.stdout, "main.go") {
		t.Errorf("--glob '*.md' should have excluded main.go, got:\n%s", got.stdout)
	}
}

func TestCLINegativeGlobExcludesTests(t *testing.T) {
	dir := sampleRepo(t)

	got := run(t, dir, "", "search", "-p", "func", "-g", "*.go", "-g", "!*_test.go", "-q")

	if got.exitCode != 0 {
		t.Fatalf("search exited with %d\nstderr: %s", got.exitCode, got.stderr)
	}
	if strings.Contains(got.stdout, "_test.go") {
		t.Errorf("test files should be excluded, got:\n%s", got.stdout)
	}
}

func TestCLICountOnly(t *testing.T) {
	dir := sampleRepo(t)

	got := run(t, dir, "", "search", "-p", "TODO", "--count", "-q")

	if got.exitCode != 0 {
		t.Fatalf("search exited with %d\nstderr: %s", got.exitCode, got.stderr)
	}
	if !strings.Contains(got.stdout, "2 matches") {
		t.Errorf("expected '2 matches', got:\n%s", got.stdout)
	}
}

func TestCLIOutputToFile(t *testing.T) {
	dir := sampleRepo(t)
	outFile := filepath.Join(dir, "matches.txt")

	got := run(t, dir, "", "search", "-p", "TODO", "-o", outFile, "-q")
	if got.exitCode != 0 {
		t.Fatalf("search exited with %d\nstderr: %s", got.exitCode, got.stderr)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	if !strings.Contains(string(data), "TODO") {
		t.Errorf("output file has no matches:\n%s", data)
	}
}

func TestCLIHistoryRecordsTheSearch(t *testing.T) {
	dir := sampleRepo(t)
	t.Chdir(dir)

	if got := run(t, dir, "", "search", "-p", "TODO", "-q"); got.exitCode != 0 {
		t.Fatalf("search exited with %d\nstderr: %s", got.exitCode, got.stderr)
	}

	records, err := analysis.Load()
	if err != nil {
		t.Fatalf("loading history: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("got %d history records, want 1", len(records))
	}
	if records[0].Pattern != "TODO" {
		t.Errorf("history pattern = %q, want TODO", records[0].Pattern)
	}
}

func TestCLINoHistorySkipsRecording(t *testing.T) {
	dir := sampleRepo(t)
	t.Chdir(dir)

	got := run(t, dir, "", "search", "-p", "TODO", "--no-history", "-q")
	if got.exitCode != 0 {
		t.Fatalf("search exited with %d\nstderr: %s", got.exitCode, got.stderr)
	}

	records, err := analysis.Load()
	if err != nil {
		t.Fatalf("loading history: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("--no-history should not record anything, got %d records", len(records))
	}
}

func TestCLIParallelSearchesDoNotLoseHistory(t *testing.T) {
	// This is the race that used to silently drop records: the dashboard and
	// the watcher both run searches as real processes, and they all append to
	// the same history file. Without a lock, records go missing.
	dir := sampleRepo(t)
	t.Chdir(dir)

	const runs = 8
	errs := make(chan error, runs)

	for i := 0; i < runs; i++ {
		go func() {
			// No cmd.Dir override, so every process shares one working
			// directory and therefore one .scope/history.json.
			cmd := exec.Command(scopeBin(t), "search", "-p", "TODO", "-q")
			cmd.Dir = dir
			errs <- cmd.Run()
		}()
	}
	for i := 0; i < runs; i++ {
		if err := <-errs; err != nil {
			t.Errorf("parallel search failed: %v", err)
		}
	}

	records, err := analysis.Load()
	if err != nil {
		t.Fatalf("loading history: %v", err)
	}
	if len(records) != runs {
		t.Errorf("got %d records after %d parallel processes, want %d - records were lost to a write race", len(records), runs, runs)
	}
}

func TestCLIAstSearch(t *testing.T) {
	dir := sampleRepo(t)

	got := run(t, dir, "", "ast", "--type", "func", "--name", "Help*")

	if got.exitCode != 0 {
		t.Fatalf("ast exited with %d\nstderr: %s", got.exitCode, got.stderr)
	}
	if !strings.Contains(got.stdout, "Helper") {
		t.Errorf("expected to find the Helper function, got:\n%s", got.stdout)
	}
}

func TestCLIAstRejectsUnknownNodeType(t *testing.T) {
	got := run(t, t.TempDir(), "", "ast", "--type", "banana")

	if got.exitCode == 0 {
		t.Error("an unknown node type should fail")
	}
	if got.stderr == "" {
		t.Error("expected an explanation on stderr")
	}
}

func TestCLIAuditRuns(t *testing.T) {
	dir := sampleRepo(t)

	got := run(t, dir, "", "audit")

	if got.exitCode != 0 {
		t.Fatalf("audit exited with %d\nstderr: %s", got.exitCode, got.stderr)
	}
	for _, want := range []string{"File Stats", "Go Dependencies"} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("audit output is missing the %q section:\n%s", want, got.stdout)
		}
	}
}

func TestCLIStatsRuns(t *testing.T) {
	dir := sampleRepo(t)

	got := run(t, dir, "", "stats")

	if got.exitCode != 0 {
		t.Fatalf("stats exited with %d\nstderr: %s", got.exitCode, got.stderr)
	}
	if !strings.Contains(got.stdout, ".go") {
		t.Errorf("stats should mention the .go extension, got:\n%s", got.stdout)
	}
}

func TestCLIHistorySubcommandPrintsTable(t *testing.T) {
	dir := sampleRepo(t)
	t.Chdir(dir)

	if got := run(t, dir, "", "search", "-p", "TODO", "-q"); got.exitCode != 0 {
		t.Fatalf("setup search exited with %d\nstderr: %s", got.exitCode, got.stderr)
	}

	got := run(t, dir, "", "history")
	if got.exitCode != 0 {
		t.Fatalf("history exited with %d\nstderr: %s", got.exitCode, got.stderr)
	}
	if !strings.Contains(got.stdout, "TODO") {
		t.Errorf("history table should list the saved pattern, got:\n%s", got.stdout)
	}
}

func TestCLIHelpListsCommands(t *testing.T) {
	got := run(t, t.TempDir(), "", "--help")

	if got.exitCode != 0 {
		t.Fatalf("--help exited with %d\nstderr: %s", got.exitCode, got.stderr)
	}
	for _, name := range []string{"search", "ast", "audit", "history", "serve", "compare"} {
		if !strings.Contains(got.stdout, name) {
			t.Errorf("help output is missing the %q command:\n%s", name, got.stdout)
		}
	}
}

func TestCLIScpAliasStillWorks(t *testing.T) {
	// We renamed the binary to scope so it stops colliding with the ssh
	// command, but the old name has to keep working for existing scripts.
	bin := scopeBin(t)
	dir := sampleRepo(t)

	cmd := exec.Command(bin, "search", "-p", "TODO", "-q")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("search via the binary failed: %v\n%s", err, out)
	}

	// The alias lives in cobra, so it is checked through the help text rather
	// than by renaming the file, which is awkward to do portably.
	help := exec.Command(bin, "--help")
	helpOut, err := help.Output()
	if err != nil {
		t.Fatalf("--help failed: %v", err)
	}
	if !strings.Contains(string(helpOut), "scp") {
		t.Errorf("help should still mention the scp alias:\n%s", helpOut)
	}
}
