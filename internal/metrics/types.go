// Package metrics tracks performance data during a search.
//
// Two-layer design:
//
//	Registry  - live counters, updated atomically during the search
//	Report    - clean snapshot built after the search, used for display
//
// Using atomics (not mutexes) because each counter is just a single int64
// that multiple goroutines increment independently. No coordination needed.
package metrics

import (
	"sync"
	"time"
)

// Registry holds the raw counters for one search run.
// Created before the search, written to by the walker + workers.
type Registry struct {
	DirsScanned  int64
	FilesScanned int64
	FilesIgnored int64
	MatchesFound int64

	WalkDuration  time.Duration
	SearchTimeNs  int64 // sum of workers' scan time in nanoseconds (stored as int64 for atomic ops)
	TotalDuration time.Duration

	Workers []*WorkerStats
}

// WorkerStats tracks one goroutine's contribution.
// All int64 fields are updated with atomic ops to avoid races.
type WorkerStats struct {
	ID int

	FilesScanned int64
	MatchesFound int64
	BytesScanned int64
	WorkTimeNs   int64

	Mu    sync.Mutex // protects Files - can't append to a slice atomically
	Files []string
}

func NewRegistry(workerCount int) *Registry {
	r := &Registry{
		Workers: make([]*WorkerStats, workerCount),
	}
	for i := range workerCount {
		r.Workers[i] = &WorkerStats{ID: i}
	}
	return r
}
