// This file implements the `scope dupes` feature — finding files with
// identical contents using SHA256 hashing.
//
// Algorithm:
//  1. Walk the filesystem and collect all file paths (via walkFiles).
//  2. N worker goroutines compute SHA256 hashes in parallel.
//  3. Files with the same hash are grouped together as duplicates.
//
// SHA256 is a cryptographic hash: if two files have the same hash, they
// are virtually guaranteed to have identical contents.
package analysis

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Viswesh-G/scope/internal/ignore"
	"github.com/Viswesh-G/scope/internal/metrics"
	"github.com/Viswesh-G/scope/internal/output"
)

// RunDupes scans path for files with identical SHA256 hashes and prints
// any duplicate groups it finds.
func RunDupes(path string, workers int) error {
	registry := metrics.NewRegistry(workers)
	totalStart := time.Now()

	ig, err := ignore.LoadIgnoreFile(filepath.Join(path, ".scope-ignore"))
	if err != nil {
		return fmt.Errorf("loading .scope-ignore: %w", err)
	}

	// fileCh: walker sends file paths here; workers read from it.
	// hashCh: workers send (path, hash) pairs here; collector reads from it.
	fileCh := make(chan string, workers*4)
	type hashResult struct {
		Path string
		Hash string
	}
	hashCh := make(chan hashResult, 256)

	// Stage 1: Walk the filesystem in a goroutine.
	go func() {
		start := time.Now()
		walkFiles(path, true, ig, fileCh, registry)
		registry.WalkDuration = time.Since(start)
	}()

	// Stage 2: Hash files in parallel using N worker goroutines.
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range fileCh {
				hash, err := hashFile(p)
				if err == nil {
					hashCh <- hashResult{Path: p, Hash: hash}
				}
			}
		}()
	}

	// Close hashCh once all workers are done, so the collector stops.
	go func() {
		wg.Wait()
		close(hashCh)
	}()

	// Stage 3: Group files by hash. Files sharing the same hash are duplicates.
	hashes := make(map[string][]string) // hash → list of file paths
	for res := range hashCh {
		hashes[res.Hash] = append(hashes[res.Hash], res.Path)
	}

	// Print any groups with more than one file.
	output.PrintHeader("Duplicate Files", "")
	found := false
	for _, paths := range hashes {
		if len(paths) > 1 {
			found = true
			for _, p := range paths {
				fmt.Println(output.FileColor.Sprint(p))
			}
			fmt.Println() // blank line between groups
		}
	}

	if !found {
		output.PrintSuccess("No duplicate files found.")
	}

	registry.TotalDuration = time.Since(totalStart)
	return nil
}

// hashFile computes the SHA256 hash of a file's contents and returns it
// as a lowercase hex string (e.g. "a3f2...").
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
