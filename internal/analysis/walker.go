// This small adapter connects the shared walker to analysis metrics and channels.
// The traversal rules themselves live in internal/walk so search and analysis
// do not slowly drift apart.
package analysis

import (
	"context"

	"github.com/Viswesh-G/scope/internal/ignore"
	"github.com/Viswesh-G/scope/internal/metrics"
	"github.com/Viswesh-G/scope/internal/walk"
)

func walkFiles(ctx context.Context, searchPath string, recursive bool, ig *ignore.IgnoreMatcher, fileCh chan<- string, registry *metrics.Registry) error {
	defer close(fileCh)
	stats, err := walk.Files(ctx, searchPath, walk.Options{Recursive: recursive}, ig, func(path string) error {
		select {
		case fileCh <- path:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	registry.DirsScanned += stats.DirsScanned
	registry.FilesScanned += stats.FilesScanned
	registry.FilesIgnored += stats.FilesIgnored
	return err
}
