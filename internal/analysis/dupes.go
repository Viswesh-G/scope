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

func RunDupes(path string, workers int) error {
	registry := metrics.NewRegistry(workers)
	totalStart := time.Now()

	ig, err := ignore.LoadIgnoreFile(filepath.Join(path, ".scope-ignore"))
	if err != nil {
		return fmt.Errorf("loading .scope-ignore: %w", err)
	}

	fileCh := make(chan string, workers*4)
	hashCh := make(chan struct {
		Path string
		Hash string
	}, 256)

	// Walker
	go func() {
		start := time.Now()
		walkFiles(path, true, ig, fileCh, registry)
		registry.WalkDuration = time.Since(start)
	}()

	// Workers
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range fileCh {
				hash, err := hashFile(p)
				if err == nil {
					hashCh <- struct {
						Path string
						Hash string
					}{p, hash}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(hashCh)
	}()

	// Collector
	hashes := make(map[string][]string)
	for res := range hashCh {
		hashes[res.Hash] = append(hashes[res.Hash], res.Path)
	}

	fmt.Println()
	fmt.Println(output.TitleColor.Sprint("Duplicate Files"))
	fmt.Println(output.DimColor.Sprint("────────────────────"))
	fmt.Println()

	found := false
	for _, paths := range hashes {
		if len(paths) > 1 {
			found = true
			for _, p := range paths {
				fmt.Println(output.FileColor.Sprint(p))
			}
			fmt.Println()
		}
	}

	if !found {
		fmt.Println("No duplicates found.")
	}

	registry.TotalDuration = time.Since(totalStart)
	return nil
}

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
