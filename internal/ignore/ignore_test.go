package ignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldIgnoreExactName(t *testing.T) {
	m := &IgnoreMatcher{Patterns: []string{"Makefile"}}
	if !m.ShouldIgnore("some/folder/Makefile") {
		t.Error("expected exact basename match to be ignored")
	}
	// note: the substring fallback means anything containing the pattern
	// text is ignored too - so Makefile.txt also gets caught
	if !m.ShouldIgnore("docs/Makefile.txt") {
		t.Error("substring fallback should catch files containing the pattern")
	}
	if m.ShouldIgnore("folder/Make") {
		t.Error("paths without the full pattern text should not match")
	}
}

func TestShouldIgnoreGlob(t *testing.T) {
	m := &IgnoreMatcher{Patterns: []string{"*.log"}}
	if !m.ShouldIgnore("logs/app.log") {
		t.Error("expected glob *.log to match app.log")
	}
	if m.ShouldIgnore("logs/app.txt") {
		t.Error("*.log should not match app.txt")
	}
}

func TestShouldIgnoreSubstring(t *testing.T) {
	m := &IgnoreMatcher{Patterns: []string{"vendor/"}}
	if !m.ShouldIgnore("project/vendor/pkg/file.go") {
		t.Error("expected substring vendor/ to match a nested path")
	}
}

func TestShouldIgnoreNoMatch(t *testing.T) {
	m := &IgnoreMatcher{Patterns: []string{"*.log", "tmp/"}}
	if m.ShouldIgnore("src/main.go") {
		t.Error("main.go should not be ignored")
	}
}

func TestLoadIgnoreFileMissing(t *testing.T) {
	m, err := LoadIgnoreFile(filepath.Join(t.TempDir(), ".scope-ignore"))
	if err != nil {
		t.Fatalf("missing file should not be an error, got %v", err)
	}
	if m != nil {
		t.Error("expected nil matcher when the file doesn't exist")
	}
}

func TestLoadIgnoreFileParsesPatterns(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".scope-ignore")
	content := "# a comment line\n\n*.log\n  tmp/  \nsecrets.yaml\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadIgnoreFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"*.log", "tmp/", "secrets.yaml"}
	if len(m.Patterns) != len(want) {
		t.Fatalf("got %d patterns, want %d: %v", len(m.Patterns), len(want), m.Patterns)
	}
	for i, p := range want {
		if m.Patterns[i] != p {
			t.Errorf("pattern[%d] = %q, want %q", i, m.Patterns[i], p)
		}
	}
}

func TestShouldSkipDirDefaults(t *testing.T) {
	// these come from defaults.yaml which is baked into the binary
	for _, name := range []string{".git", "node_modules", "vendor"} {
		if !ShouldSkipDir(name, nil) {
			t.Errorf("%s should be skipped by default (even with a nil matcher)", name)
		}
	}
	if ShouldSkipDir("src", nil) {
		t.Error("src should not be skipped")
	}
}

func TestShouldSkipFileDefaults(t *testing.T) {
	if !ShouldSkipFile("assets/logo.png", nil) {
		t.Error(".png files should be skipped by default")
	}
	if !ShouldSkipFile("tools/scope.exe", nil) {
		t.Error(".exe files should be skipped by default")
	}
	// the scope binary itself is always skipped so we don't search inside it
	if !ShouldSkipFile("bin/scope", nil) {
		t.Error("the scope binary itself should be skipped")
	}
	if ShouldSkipFile("main.go", nil) {
		t.Error("main.go should not be skipped")
	}
}

func TestShouldSkipFileWithMatcher(t *testing.T) {
	m := &IgnoreMatcher{Patterns: []string{"*.gen.go"}}
	if !ShouldSkipFile("internal/api/client.gen.go", m) {
		t.Error("client.gen.go should be ignored by the custom pattern")
	}
	if ShouldSkipFile("internal/api/client.go", m) {
		t.Error("client.go should not be ignored")
	}
}
