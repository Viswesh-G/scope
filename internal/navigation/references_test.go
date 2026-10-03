package navigation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindReferencesAcrossPackages(t *testing.T) {
	root := t.TempDir()
	writeSource(t, root, "go.mod", "module example.com/refdemo\n\ngo 1.25.6\n")
	writeSource(t, root, "target.go", "package refdemo\n\nfunc Target() {}\n")
	writeSource(t, root, "caller/caller.go", "package caller\n\nimport \"example.com/refdemo\"\n\nfunc Call() { refdemo.Target() }\n")
	writeSource(t, root, "target_test.go", "package refdemo\n\nimport \"testing\"\n\nfunc TestTargetUse(t *testing.T) { Target() }\n")

	results, err := FindReferences(Config{Path: root, Name: "Target"})
	if err != nil {
		t.Fatalf("FindReferences returned an error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d symbol results, want 1", len(results))
	}
	got := results[0]
	if got.Kind != "func" {
		t.Errorf("symbol kind = %q, want func", got.Kind)
	}
	if got.Definition.File != "target.go" || got.Definition.Line != 3 {
		t.Errorf("definition = %+v, want target.go:3", got.Definition)
	}

	foundCaller := false
	foundTest := false
	seen := make(map[string]bool)
	for _, reference := range got.References {
		key := fmt.Sprintf("%s:%d:%d", reference.File, reference.Line, reference.Column)
		if seen[key] {
			t.Errorf("reference location appears more than once: %+v", reference)
		}
		seen[key] = true
		if reference.File == "caller/caller.go" {
			foundCaller = true
		}
		if reference.File == "target_test.go" {
			foundTest = true
		}
	}
	if !foundCaller {
		t.Errorf("references did not include the external package call: %+v", got.References)
	}
	if !foundTest {
		t.Errorf("references did not include the test call: %+v", got.References)
	}
}

func TestFindReferencesRequiresNarrowingForDuplicateNames(t *testing.T) {
	root := t.TempDir()
	writeSource(t, root, "go.mod", "module example.com/duplicates\n\ngo 1.25.6\n")
	writeSource(t, root, "first/target.go", "package first\n\nfunc Target() {}\n")
	writeSource(t, root, "second/target.go", "package second\n\nfunc Target() {}\n")

	_, err := FindReferences(Config{Path: root, Name: "Target"})
	if err == nil || !strings.Contains(err.Error(), "--file and --line") {
		t.Fatalf("expected an ambiguity error that asks for --file and --line, got %v", err)
	}

	results, err := FindReferences(Config{
		Path: root,
		Name: "Target",
		File: filepath.Join("first", "target.go"),
		Line: 3,
	})
	if err != nil {
		t.Fatalf("narrowed FindReferences returned an error: %v", err)
	}
	if len(results) != 1 || results[0].Definition.File != "first/target.go" {
		t.Fatalf("narrowed search selected the wrong definition: %+v", results)
	}
}

func TestFindReferencesReportsTypeErrors(t *testing.T) {
	root := t.TempDir()
	writeSource(t, root, "go.mod", "module example.com/broken\n\ngo 1.25.6\n")
	writeSource(t, root, "broken.go", "package broken\n\nfunc broken() { missing() }\n")

	_, err := FindReferences(Config{Path: root, Name: "broken"})
	if err == nil || !strings.Contains(err.Error(), "could not type-check packages") {
		t.Fatalf("expected a package type-check error, got %v", err)
	}
}

func TestFindReferencesValidatesSelection(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   string
	}{
		{"empty path", Config{Name: "Target"}, "package path"},
		{"empty name", Config{Path: "."}, "symbol name"},
		{"negative line", Config{Path: ".", Name: "Target", Line: -1}, "line"},
		{"line without file", Config{Path: ".", Name: "Target", Line: 2}, "--line requires --file"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := FindReferences(test.config)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}

func writeSource(t *testing.T, root, name, source string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("creating directory for %s: %v", name, err)
	}
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}
