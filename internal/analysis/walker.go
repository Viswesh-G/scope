// Shared filesystem walker used by analysis commands (dupes, etc.)
//
// Note: search/engine.go has its own similar walker embedded inside Run().
// I kept them separate because the search walker needs access to the search
// Config (e.g. cfg.Recursive), and merging them would complicate both.
// The logic is identical - just different call sites.
package analysis

import (
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/Viswesh-G/scope/internal/ignore"
	"github.com/Viswesh-G/scope/internal/metrics"
)

func walkFiles(searchPath string, recursive bool, ig *ignore.IgnoreMatcher, fileCh chan<- string, registry *metrics.Registry) {
	defer close(fileCh)

	_ = filepath.WalkDir(searchPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			atomic.AddInt64(&registry.DirsScanned, 1)
			if ignore.ShouldSkipDir(d.Name(), ig) {
				return filepath.SkipDir
			}
			if !recursive && path != searchPath {
				return filepath.SkipDir
			}
			return nil
		}

		if ignore.ShouldSkipFile(path, ig) {
			atomic.AddInt64(&registry.FilesIgnored, 1)
			return nil
		}

		atomic.AddInt64(&registry.FilesScanned, 1)
		fileCh <- path
		return nil
	})
}
