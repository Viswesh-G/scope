package analysis

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/Viswesh-G/scope/internal/ignore"
	"github.com/Viswesh-G/scope/internal/output"
)

var importRe = regexp.MustCompile(`(?m)^\s*import\s+(?:(?:[a-zA-Z0-9_]+\s+)?("[^"]+")|\(([\s\S]*?)\))`)
var singleImportRe = regexp.MustCompile(`"([^"]+)"`)

func RunDeps(path string) error {
	ig, err := ignore.LoadIgnoreFile(filepath.Join(path, ".scope-ignore"))
	if err != nil {
		return fmt.Errorf("loading .scope-ignore: %w", err)
	}

	deps := make(map[string]int)

	_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if ignore.ShouldSkipDir(d.Name(), ig) {
				return filepath.SkipDir
			}
			return nil
		}
		if ignore.ShouldSkipFile(p, ig) || filepath.Ext(p) != ".go" {
			return nil
		}

		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}

		matches := importRe.FindAllStringSubmatch(string(data), -1)
		for _, m := range matches {
			if m[1] != "" {
				// Single import
				pkg := singleImportRe.FindStringSubmatch(m[1])
				if len(pkg) > 1 {
					deps[pkg[1]]++
				}
			} else if m[2] != "" {
				// Block import
				blockMatches := singleImportRe.FindAllStringSubmatch(m[2], -1)
				for _, bm := range blockMatches {
					if len(bm) > 1 {
						deps[bm[1]]++
					}
				}
			}
		}

		return nil
	})

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
		[]string{"Package", "Imports"},
		rows,
		[]string{"left", "right"},
	)

	return nil
}
