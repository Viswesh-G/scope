// Package ignore figures out which files and directories to skip.
//
// Two sources of rules:
//   1. defaults.yaml - baked into the binary at compile time via go:embed.
//      Covers .git, node_modules, binary extensions, etc.
//   2. .scope-ignore  - optional per-repo file, same format as .gitignore.
//
// Both are checked on every file/dir the walker visits.
package ignore

import (
	"bufio"
	_ "embed"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// embed the defaults file directly into the binary at compile time.
// this means we don't need to ship the yaml file alongside the binary.
//
//go:embed defaults.yaml
var defaultsYAML []byte

// using maps for O(1) lookup instead of scanning a list each time
var ignoredDirs map[string]struct{}
var ignoredExtensions map[string]struct{}

func init() {
	ignoredDirs = make(map[string]struct{})
	ignoredExtensions = make(map[string]struct{})

	var defaults struct {
		Dirs       []string `yaml:"dirs"`
		Extensions []string `yaml:"extensions"`
	}
	if err := yaml.Unmarshal(defaultsYAML, &defaults); err != nil {
		panic("failed to load ignore defaults: " + err.Error())
	}

	for _, d := range defaults.Dirs {
		ignoredDirs[d] = struct{}{}
	}
	for _, e := range defaults.Extensions {
		ignoredExtensions[e] = struct{}{}
	}
}

// IgnoreMatcher holds patterns loaded from a .scope-ignore file.
// nil is safe to pass to ShouldSkipDir/ShouldSkipFile (handled gracefully).
type IgnoreMatcher struct {
	Patterns []string
}

func ShouldSkipDir(name string, ig *IgnoreMatcher) bool {
	if _, ok := ignoredDirs[name]; ok {
		return true
	}
	if ig != nil && ig.ShouldIgnore(name) {
		return true
	}
	return false
}

func ShouldSkipFile(path string, ig *IgnoreMatcher) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if _, ok := ignoredExtensions[ext]; ok {
		return true
	}
	// skip the scope binary itself so it doesn't try to search inside its own exe
	if filepath.Base(path) == "scope" {
		return true
	}
	if ig != nil && ig.ShouldIgnore(path) {
		return true
	}
	return false
}

// LoadIgnoreFile reads a .scope-ignore file.
// Returns (nil, nil) if the file doesn't exist - that's not an error.
func LoadIgnoreFile(path string) (*IgnoreMatcher, error) {
	file, err := os.Open(path)
	if err != nil {
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

// ShouldIgnore checks if a path matches any stored pattern.
// Tries three matching strategies in order:
//   1. exact basename match  ("Makefile" matches any file named Makefile)
//   2. glob match            ("*.log" matches "app.log")
//   3. substring match       ("vendor/" matches "vendor/pkg/file.go")
func (m *IgnoreMatcher) ShouldIgnore(path string) bool {
	base := filepath.Base(path)
	for _, pattern := range m.Patterns {
		if base == pattern {
			return true
		}
		if matched, err := filepath.Match(pattern, base); err == nil && matched {
			return true
		}
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}
