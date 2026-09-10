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
	SearchTimeNs  int64 // sum of workers' total CPU time in nanoseconds (stored as int64 for atomic ops)
	TotalDuration time.Duration

	Workers []*WorkerStats
}

// WorkerEvent tracks when a worker started processing a file and how long it took.
// This helps us draw a visual timeline so students can see parallel processing in action!
type WorkerEvent struct {
	StartOffsetNs int64 // How many nanoseconds after the search started did this begin?
	DurationNs    int64 // How many nanoseconds did it take to process the file?
}

// WorkerStats tracks one goroutine's contribution.
// All int64 fields are updated with atomic ops to avoid races.
type WorkerStats struct {
	ID int

	FilesScanned int64
	MatchesFound int64
	BytesScanned int64
	WorkTimeNs   int64

	Mu     sync.Mutex // protects Files and Events - we lock it before adding to slices!
	Files  []string
	Events []WorkerEvent // list of events for the parallel profile visualization
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
