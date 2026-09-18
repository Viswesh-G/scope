// This file implements `scope deps` — scanning Go source files and counting
// how many times each import package is referenced across the codebase.
//
// It uses a regex to find all import blocks and single-line imports,
// then extracts each quoted package path and tallies them.
package analysis

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/Viswesh-G/scope/internal/ignore"
	"github.com/Viswesh-G/scope/internal/output"
	"github.com/Viswesh-G/scope/internal/walk"
)

// importRe matches Go import statements in two forms:
//   - Single: import "pkg/name"
//   - Block:  import ( "pkg/a" \n "pkg/b" )
var importRe = regexp.MustCompile(`(?m)^\s*import\s+(?:(?:[a-zA-Z0-9_]+\s+)?("([^"]+)"|\([\s\S]*?\)))`)

// singleImportRe extracts quoted package paths from import text.
var singleImportRe = regexp.MustCompile(`"([^"]+)"`)

// RunDeps walks all .go files in path and counts how often each package
// is imported. Results are printed sorted by import count (most used first).
func RunDeps(path string) error {
	ig, err := ignore.LoadIgnoreFile(filepath.Join(path, ".scope-ignore"))
	if err != nil {
		return fmt.Errorf("loading .scope-ignore: %w", err)
	}

	// Map from import path (e.g. "fmt") to how many files import it.
	deps := make(map[string]int)

	_, err = walk.Files(context.Background(), path, walk.Options{Recursive: true}, ig, func(p string) error {
		// Only scan .go source files.
		if ignore.ShouldSkipFile(p, ig) || filepath.Ext(p) != ".go" {
			return nil
		}

		data, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("reading %q: %w", p, err)
		}

		// Find all import statements in this file.
		matches := importRe.FindAllStringSubmatch(string(data), -1)
		for _, m := range matches {
			// Extract all quoted package paths from each import statement.
			pkgs := singleImportRe.FindAllStringSubmatch(m[0], -1)
			for _, pkg := range pkgs {
				if len(pkg) > 1 {
					deps[pkg[1]]++
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walking %q: %w", path, err)
	}

	// Sort by count descending, then alphabetically for ties.
	type depCount struct {
		pkg   string
		count int
	}
	var list []depCount
	for pkg, count := range deps {
		list = append(list, depCount{pkg, count})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].count == list[j].count {
			return list[i].pkg < list[j].pkg
		}
		return list[i].count > list[j].count
	})

	output.PrintHeader("Go Dependencies", "")

	var rows [][]string
	for _, d := range list {
		rows = append(rows, []string{d.pkg, fmt.Sprintf("%d", d.count)})
	}
	output.PrintTable(
		[]string{"Package", "Import Count"},
		rows,
		[]string{"left", "right"},
	)
	return nil
}
