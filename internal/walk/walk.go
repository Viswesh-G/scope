// Package walk contains the shared filesystem traversal used by analysis and search.
package walk

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Viswesh-G/scope/internal/ignore"
)

type Options struct {
	Recursive bool
	Globs     []string
}

type Stats struct {
	DirsScanned  int64
	FilesScanned int64
	FilesIgnored int64
}

// Files visits eligible files and stops promptly when ctx is cancelled.
func Files(ctx context.Context, root string, opts Options, ig *ignore.IgnoreMatcher, visit func(string) error) (Stats, error) {
	var stats Stats
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			stats.DirsScanned++
			if ignore.ShouldSkipDir(entry.Name(), ig) {
				return filepath.SkipDir
			}
			if !opts.Recursive && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if ignore.ShouldSkipFile(path, ig) || (len(opts.Globs) > 0 && !matchesGlobs(path, opts.Globs)) {
			stats.FilesIgnored++
			return nil
		}
		stats.FilesScanned++
		if err := visit(path); err != nil {
			return err
		}
		return nil
	})
	return stats, err
}

func matchesGlobs(path string, globs []string) bool {
	base := filepath.Base(path)
	hasPositive := false
	for _, pattern := range globs {
		if pattern != "" && pattern[0] != '!' {
			hasPositive = true
			break
		}
	}
	matched := !hasPositive
	for _, pattern := range globs {
		if pattern == "" {
			continue
		}
		negative := pattern[0] == '!'
		if negative {
			pattern = pattern[1:]
		}
		ok, err := filepath.Match(pattern, base)
		if err == nil && ok {
			if negative {
				return false
			}
			matched = true
		}
	}
	return matched
}
