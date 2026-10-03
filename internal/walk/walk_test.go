// Tests for the shared filesystem walker.
//
// The walker is used by search, ast, stats, deps, dupes and graph, so a bug
// here would quietly break half the project. These tests build a small tree of
// files in a temp directory and check what actually gets visited.
package walk

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/Viswesh-G/scope/internal/ignore"
)

// makeTree creates a directory tree from a list of relative file paths.
// Directories are created for us, so a list entry like "internal/search/a.go"
// just works.
func makeTree(t *testing.T, files ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, name := range files {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("making dir for %q: %v", name, err)
		}
		if err := os.WriteFile(full, []byte("hello\n"), 0644); err != nil {
			t.Fatalf("writing %q: %v", name, err)
		}
	}
	return root
}

// collect runs the walker and returns the visited files as basenames, sorted
// so the order the filesystem happened to hand them over does not matter.
func collect(t *testing.T, root string, opts Options, ig *ignore.IgnoreMatcher) []string {
	t.Helper()
	var seen []string
	_, err := Files(context.Background(), root, opts, ig, func(path string) error {
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		seen = append(seen, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("Files returned an error: %v", err)
	}
	sort.Strings(seen)
	return seen
}

func same(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestFilesVisitsEverythingByDefault(t *testing.T) {
	root := makeTree(t, "a.txt", "b.md", "internal/c.go", "internal/deep/d.go")

	got := collect(t, root, Options{Recursive: true}, nil)
	want := []string{"a.txt", "b.md", "internal/c.go", "internal/deep/d.go"}

	if !same(got, want) {
		t.Errorf("visited %v, want %v", got, want)
	}
}

func TestFilesNonRecursiveStaysAtTopLevel(t *testing.T) {
	root := makeTree(t, "a.txt", "b.md", "internal/c.go")

	got := collect(t, root, Options{Recursive: false}, nil)
	want := []string{"a.txt", "b.md"}

	if !same(got, want) {
		t.Errorf("non-recursive walk visited %v, want only the top level %v", got, want)
	}
}

func TestFilesSkipsDefaultIgnoredDirectories(t *testing.T) {
	// .git and node_modules are on the built-in ignore list, so the walker
	// should never even look inside them.
	root := makeTree(t,
		"keep.txt",
		".git/config",
		"node_modules/left-pad/index.js",
		"dist/bundle.js",
	)

	got := collect(t, root, Options{Recursive: true}, nil)

	if !same(got, []string{"keep.txt"}) {
		t.Errorf("visited %v, want just keep.txt (ignored dirs must be skipped)", got)
	}
}

func TestFilesSkipsDefaultIgnoredExtensions(t *testing.T) {
	root := makeTree(t, "notes.txt", "photo.png", "archive.zip", "lib.dll")

	got := collect(t, root, Options{Recursive: true}, nil)

	if !same(got, []string{"notes.txt"}) {
		t.Errorf("visited %v, want just notes.txt (binary/media files must be skipped)", got)
	}
}

func TestFilesHonoursScopeIgnoreFile(t *testing.T) {
	root := makeTree(t, "main.go", "notes.md", "tmp/scratch.go", "tmp/scratch.md")

	ignoreFile := filepath.Join(root, ".scope-ignore")
	if err := os.WriteFile(ignoreFile, []byte("# scratch files\ntmp/\n"), 0644); err != nil {
		t.Fatalf("writing .scope-ignore: %v", err)
	}

	ig, err := ignore.LoadIgnoreFile(ignoreFile)
	if err != nil {
		t.Fatalf("loading ignore file: %v", err)
	}
	if ig == nil {
		t.Fatal("ignore file was not picked up, matcher is nil")
	}

	got := collect(t, root, Options{Recursive: true}, ig)

	// The .scope-ignore file itself is a normal text file, so it is still
	// searched. Ripgrep behaves the same way with .gitignore.
	want := []string{".scope-ignore", "main.go", "notes.md"}
	if !same(got, want) {
		t.Errorf("visited %v, want %v (everything under tmp/ is ignored)", got, want)
	}
}

func TestFilesPositiveGlobKeepsOnlyMatching(t *testing.T) {
	root := makeTree(t, "a.go", "b.go", "c.txt")

	got := collect(t, root, Options{Recursive: true, Globs: []string{"*.go"}}, nil)

	if !same(got, []string{"a.go", "b.go"}) {
		t.Errorf("visited %v, want only the .go files", got)
	}
}

func TestFilesNegativeGlobExcludesMatching(t *testing.T) {
	root := makeTree(t, "engine.go", "engine_test.go", "util.go", "util_test.go")

	// A leading ! means "leave this one out", even when other patterns
	// would otherwise include it.
	opts := Options{Recursive: true, Globs: []string{"*.go", "!*_test.go"}}
	got := collect(t, root, opts, nil)

	want := []string{"engine.go", "util.go"}
	if !same(got, want) {
		t.Errorf("visited %v, want %v (test files must be excluded)", got, want)
	}
}

func TestFilesNegativeGlobAloneStartsFromEverything(t *testing.T) {
	root := makeTree(t, "a.go", "a_test.go", "b.txt")

	// With no positive pattern the walker starts from "everything" and the
	// negative pattern carves things out of it.
	opts := Options{Recursive: true, Globs: []string{"!*_test.go"}}
	got := collect(t, root, opts, nil)

	want := []string{"a.go", "b.txt"}
	if !same(got, want) {
		t.Errorf("visited %v, want %v", got, want)
	}
}

func TestFilesCountsStats(t *testing.T) {
	root := makeTree(t, "a.go", "b.go", "sub/c.go", "skipme/d.go", "photo.png")

	// The visitor count and Stats.FilesScanned should agree, because the
	// metrics table prints that number as if it were a fact.
	visits := 0
	stats, err := Files(context.Background(), root, Options{Recursive: true}, nil, func(string) error {
		visits++
		return nil
	})
	if err != nil {
		t.Fatalf("Files returned an error: %v", err)
	}

	if stats.FilesScanned != int64(visits) {
		t.Errorf("Stats.FilesScanned = %d but the visitor ran %d times", stats.FilesScanned, visits)
	}
	if stats.FilesIgnored == 0 {
		t.Error("Stats.FilesIgnored is 0, but photo.png should have been ignored")
	}
	if stats.DirsScanned < 3 {
		t.Errorf("Stats.DirsScanned = %d, expected at least the root plus sub/ and skipme/", stats.DirsScanned)
	}
}

func TestFilesReturnsErrorForMissingPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope", "does-not-exist")

	_, err := Files(context.Background(), missing, Options{Recursive: true}, nil, func(string) error {
		t.Error("visitor should not run for a path that does not exist")
		return nil
	})
	if err == nil {
		t.Error("walking a missing directory should return an error, got nil")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("error should wrap os.ErrNotExist, got %v", err)
	}
}

func TestFilesReturnsErrorFromVisitor(t *testing.T) {
	root := makeTree(t, "a.go", "b.go")
	sentinel := errors.New("stop right there")

	calls := 0
	_, err := Files(context.Background(), root, Options{Recursive: true}, nil, func(string) error {
		calls++
		return sentinel
	})

	if !errors.Is(err, sentinel) {
		t.Errorf("walker should pass the visitor's error back up, got %v", err)
	}
	if calls != 1 {
		t.Errorf("walker kept going after the visitor failed: ran %d times, want 1", calls)
	}
}

func TestFilesStopsWhenContextIsCancelled(t *testing.T) {
	// A pile of files gives the walker something to chew on, so cancelling
	// partway through is a realistic thing to test.
	files := make([]string, 0, 200)
	for i := 0; i < 200; i++ {
		files = append(files, fmt.Sprintf("dir%d/f.go", i%20))
	}
	root := makeTree(t, files...)

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel as soon as the first file shows up, which is before the walk
	// has had any chance to finish.
	var visited int
	_, err := Files(ctx, root, Options{Recursive: true}, nil, func(string) error {
		visited++
		cancel()
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled walk should return context.Canceled, got %v", err)
	}
	if visited > 5 {
		t.Errorf("walker visited %d files after cancellation, expected it to stop almost immediately", visited)
	}
}

func TestFilesOnAlreadyCancelledContextVisitsNothing(t *testing.T) {
	root := makeTree(t, "a.go", "b.go", "c.go")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Files(ctx, root, Options{Recursive: true}, nil, func(string) error {
		t.Error("visitor should not run when the context is already cancelled")
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestMatchesGlobs(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		globs  []string
		expect bool
	}{
		{"no globs means everything", "a.go", nil, true},
		{"positive match", "a.go", []string{"*.go"}, true},
		{"positive miss", "a.txt", []string{"*.go"}, false},
		{"negative excludes", "a_test.go", []string{"*.go", "!*_test.go"}, false},
		{"positive survives a negative miss", "a.go", []string{"*.go", "!*_test.go"}, true},
		{"negative only keeps the rest", "a.txt", []string{"!*_test.go"}, true},
		{"empty globs are ignored", "a.go", []string{""}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesGlobs(tt.path, tt.globs); got != tt.expect {
				t.Errorf("matchesGlobs(%q, %v) = %v, want %v", tt.path, tt.globs, got, tt.expect)
			}
		})
	}
}
