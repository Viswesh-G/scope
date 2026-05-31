package ignore

import (
	"bufio"
	_ "embed"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed defaults.yaml
var defaultsYAML []byte

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

// IgnoreMatcher holds a list of compiled patterns.
type IgnoreMatcher struct {
	Patterns []string
}

// ShouldSkipDir returns true when the walker should skip an entire directory.
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
func ShouldSkipFile(path string, ignore *IgnoreMatcher) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if _, ok := ignoredExtensions[ext]; ok {
		return true
	}

	if filepath.Base(path) == "scope" {
		return true
	}

	if ignore != nil && ignore.ShouldIgnore(path) {
		return true
	}

	return false
}

// LoadIgnoreFile reads a .scope-ignore file and returns an IgnoreMatcher.
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

// ShouldIgnore checks whether path matches any pattern in the IgnoreMatcher.
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
