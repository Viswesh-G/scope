package search

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// ---------------------------------------------------------------------------
// Hard-coded skip lists
// These directories and file extensions are always skipped — they are either
// generated files, binary blobs, or version-control internals that are never
// useful to text-search.
// ---------------------------------------------------------------------------

// ignoredDirs lists directory names that should never be walked into.
var ignoredDirs = map[string]struct{}{
	".git":         {}, // Git internal data
	"node_modules": {}, // npm / JavaScript dependencies
	"vendor":       {}, // Go / Ruby vendored deps
	"dist":         {}, // compiled frontend output
	"build":        {}, // compiled output (many ecosystems)
	"bin":          {}, // compiled binaries
	".idea":        {}, // JetBrains IDE config
	".vscode":      {}, // VS Code config
	".cache":       {}, // generic cache folders
}

// ignoredExtensions lists file suffixes that are binary or non-text.
var ignoredExtensions = map[string]struct{}{
	// compiled code
	".exe": {}, ".dll": {}, ".so": {}, ".dylib": {}, ".o": {}, ".a": {},
	// binary blobs
	".bin": {}, ".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {},
	".pdf": {}, ".zip": {}, ".tar": {}, ".gz": {}, ".7z": {},
	".mp4": {}, ".mp3": {},
}

// ShouldSkipDir returns true when the walker should skip an entire directory.
// It checks both the hard-coded list and the optional .scope-ignore rules.
func ShouldSkipDir(name string, ignore *IgnoreMatcher) bool {
	if _, ok := ignoredDirs[name]; ok {
		return true
	}
	if ignore != nil && ignore.ShouldIgnore(name) {
		return true
	}
	return false
}

// ShouldSkipFile returns true when a file should NOT be searched.
// It checks the extension denylist, the special "scope" filename, and any
// user-supplied .scope-ignore rules.
func ShouldSkipFile(path string, ignore *IgnoreMatcher) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if _, ok := ignoredExtensions[ext]; ok {
		return true
	}

	// "scope" (no extension) is a special sentinel file — always skip it.
	if filepath.Base(path) == "scope" {
		return true
	}

	if ignore != nil && ignore.ShouldIgnore(path) {
		return true
	}

	return false
}

// ---------------------------------------------------------------------------
// .scope-ignore support
// ---------------------------------------------------------------------------

// LoadIgnoreFile reads a .scope-ignore file and returns an IgnoreMatcher.
// The file format is simple:
//   - blank lines are ignored
//   - lines starting with # are comments
//   - every other line is a pattern (exact name, glob like *.log, or substring)
//
// If the file does not exist, LoadIgnoreFile returns (nil, nil) — callers
// should treat a nil IgnoreMatcher as "no extra rules".
func LoadIgnoreFile(path string) (*IgnoreMatcher, error) {
	file, err := os.Open(path)
	if err != nil {
		// File simply doesn't exist — that's fine, not an error for the caller.
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var patterns []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip blank lines and comments.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		patterns = append(patterns, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &IgnoreMatcher{Patterns: patterns}, nil
}

// ShouldIgnore checks whether path matches any pattern in the IgnoreMatcher.
//
// Three matching strategies are tried in order:
//  1. Exact match against the base name  (e.g. ".env")
//  2. Glob match against the base name   (e.g. "*.log")
//  3. Substring match inside the full path (e.g. "secret/")
func (m *IgnoreMatcher) ShouldIgnore(path string) bool {
	base := filepath.Base(path)

	for _, pattern := range m.Patterns {
		// 1. Exact name match.
		if base == pattern {
			return true
		}

		// 2. Glob match (e.g. "*.log" matches "app.log").
		if matched, err := filepath.Match(pattern, base); err == nil && matched {
			return true
		}

		// 3. The pattern appears anywhere in the full path.
		if strings.Contains(path, pattern) {
			return true
		}
	}

	return false
}
