// internal/metrics/types.go
package metrics

import (
	"sync"
	"time"
)

type Registry struct {
	DirsScanned  int64
	FilesScanned int64
	FilesIgnored int64
	MatchesFound int64

	WalkDuration  time.Duration
	SearchTimeNs  int64
	TotalDuration time.Duration

	Workers []*WorkerStats
}

type WorkerStats struct {
	ID int

	FilesScanned int64
	MatchesFound int64
	WorkTimeNs   int64

	Mu    sync.Mutex
	Files []string // paths processed by this worker
}

func NewRegistry(workerCount int) *Registry {
	r := &Registry{
		Workers: make([]*WorkerStats, workerCount),
	}

	for i := range workerCount {
		r.Workers[i] = &WorkerStats{
			ID: i,
		}
	}

	return r
}
