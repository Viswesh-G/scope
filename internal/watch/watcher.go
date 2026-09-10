// Package watch implements the "scp watch" command.
//
// The idea is simple: use the operating system's file-change notification API
// (via the fsnotify library) to watch a directory tree, then re-run the scp
// search automatically every time a file changes.
//
// One tricky part is "debouncing": when you save a file, your editor might
// write multiple events in quick succession. Without debouncing, the search
// would re-run many times per second. We solve this by waiting a short time
// after the last event before re-running the search.
package watch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Viswesh-G/scope/internal/output"
	"github.com/Viswesh-G/scope/internal/search"
	"github.com/fsnotify/fsnotify"
)

// Config holds the settings for a watch session.
// It mirrors the relevant fields from search.Config so the two commands feel consistent.
type Config struct {
	Pattern    string
	Path       string
	Workers    int
	IgnoreCase bool
}

// debounceDelay is how long we wait after the last file event before re-running.
// 300ms is a comfortable value: fast enough to feel instant, slow enough that
// burst events from a single editor save don't trigger multiple searches.
const debounceDelay = 300 * time.Millisecond

// Run starts the watcher loop.
// It immediately runs the first search, then waits for file changes.
func Run(cfg Config) error {
	// Create the fsnotify watcher. This asks the OS to send us events when
	// files change — much more efficient than polling with time.Sleep.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating file watcher: %w", err)
	}
	defer watcher.Close()

	// Add the root directory and all its subdirectories recursively.
	// fsnotify only watches individual directories, not trees, so we need
	// to walk the tree ourselves and add each directory.
	if err := addDirRecursive(watcher, cfg.Path); err != nil {
		return fmt.Errorf("watching path %q: %w", cfg.Path, err)
	}

	fmt.Println()
	output.TitleColor.Printf("Watching %s for %q\n", cfg.Path, cfg.Pattern)
	output.DimColor.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Run the search immediately on startup so the user sees results right away.
	runSearch(cfg)

	// The timer is used for debouncing. Every time we get a file event, we
	// reset this timer. The search only fires when the timer actually expires.
	var debounce *time.Timer

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil // channel was closed, watcher is done
			}

			// Ignore irrelevant events like Chmod
			if !isRelevantEvent(event) {
				continue
			}

			// Reset the debounce timer. Stop() returns false if the timer already
			// expired, in which case we try to drain the channel before resetting.
			if debounce != nil {
				if !debounce.Stop() {
					select {
					case <-debounce.C:
					default:
					}
				}
			}

			// Wait a bit before actually running the search
			debounce = time.AfterFunc(debounceDelay, func() {
				// Print a separator so each search run is visually distinct
				fmt.Println()
				output.DimColor.Printf("── changed: %s ──\n", filepath.Base(event.Name))
				runSearch(cfg)
			})

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			output.PrintWarning(fmt.Sprintf("watcher error: %v", err))
		}
	}
}

// addDirRecursive walks the directory tree starting at root and adds every
// directory to the watcher. fsnotify only supports watching individual dirs,
// not recursive trees, so we have to add them all manually.
func addDirRecursive(watcher *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable paths gracefully
		}
		if !d.IsDir() {
			return nil
		}
		// Skip hidden dirs and common junk directories so we don't
		// fill the watcher with thousands of useless entries
		name := d.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
			return filepath.SkipDir
		}
		return watcher.Add(path)
	})
}

// isRelevantEvent returns true if the event is something that should trigger
// a re-search. We care about file writes and renames, but not permission changes.
func isRelevantEvent(event fsnotify.Event) bool {
	return event.Has(fsnotify.Write) ||
		event.Has(fsnotify.Create) ||
		event.Has(fsnotify.Remove) ||
		event.Has(fsnotify.Rename)
}

// runSearch builds a search.Config from our watch config and runs it.
// We use --quiet mode so the metrics table doesn't clutter the watch output.
func runSearch(cfg Config) {
	scfg := search.Config{
		OriginalPattern: cfg.Pattern,
		Pattern:         cfg.Pattern,
		Path:            cfg.Path,
		Recursive:       true,
		Workers:         cfg.Workers,
		IgnoreCase:      cfg.IgnoreCase,
		Quiet:           true,    // suppress the metrics table
		SkipHistory:     true,    // watch runs don't pollute history
	}
	if err := search.Run(scfg); err != nil {
		output.PrintError(err)
	}
}
