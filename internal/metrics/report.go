// BuildReport takes a live Registry and produces a clean Report for display.
// Called once after the search is done - snapshots all the atomic counters.
package metrics

import (
	"sort"
	"sync/atomic"
	"time"
)

// Report is the cleaned-up version of Registry. Passed to the renderer.
type Report struct {
	DirsScanned  int64
	FilesScanned int64
	FilesIgnored int64
	MatchesFound int64

	WalkDuration   time.Duration
	SearchDuration time.Duration
	TotalDuration  time.Duration

	// Parallelism = SearchDuration / TotalDuration
	// If 4 workers each spent 100ms scanning, SearchDuration = 400ms.
	// If the whole search took 120ms wall time, Parallelism = 400/120 ≈ 3.3x
	// meaning we got 3.3x the throughput we'd get with a single worker.
	Parallelism float64

	Workers []WorkerReport
	Balance LoadBalance
}

type WorkerReport struct {
	ID           int
	FilesScanned int64
	MatchesFound int64
	BytesScanned int64
	WorkDuration time.Duration
	Throughput   float64  // bytes/second, 0 if WorkDuration == 0
	Files        []string
}

// LoadBalance summarises how evenly files were split across workers.
// Imbalance = MaxFiles / MinFiles. 1.0 is perfect, higher = uneven.
type LoadBalance struct {
	MaxFiles  int64
	MinFiles  int64
	AvgFiles  float64
	MaxBytes  int64
	MinBytes  int64
	AvgBytes  float64
	Imbalance float64

	Ranked []WorkerReport // workers sorted by bytes scanned (busiest first)
}

func BuildReport(r *Registry) Report {
	searchDuration := time.Duration(atomic.LoadInt64(&r.SearchTimeNs))

	report := Report{
		DirsScanned:    atomic.LoadInt64(&r.DirsScanned),
		FilesScanned:   atomic.LoadInt64(&r.FilesScanned),
		FilesIgnored:   atomic.LoadInt64(&r.FilesIgnored),
		MatchesFound:   atomic.LoadInt64(&r.MatchesFound),
		WalkDuration:   r.WalkDuration,
		SearchDuration: searchDuration,
		TotalDuration:  r.TotalDuration,
	}

	if r.TotalDuration > 0 {
		report.Parallelism = float64(searchDuration) / float64(r.TotalDuration)
	}

	for _, w := range r.Workers {
		w.Mu.Lock()
		files := make([]string, len(w.Files))
		copy(files, w.Files)
		w.Mu.Unlock()

		dur := time.Duration(atomic.LoadInt64(&w.WorkTimeNs))
		bytes := atomic.LoadInt64(&w.BytesScanned)

		wr := WorkerReport{
			ID:           w.ID,
			FilesScanned: atomic.LoadInt64(&w.FilesScanned),
			MatchesFound: atomic.LoadInt64(&w.MatchesFound),
			BytesScanned: bytes,
			WorkDuration: dur,
			Files:        files,
		}
		if dur > 0 {
			wr.Throughput = float64(bytes) / dur.Seconds()
		}
		report.Workers = append(report.Workers, wr)
	}

	report.Balance = computeBalance(report.Workers)
	return report
}

func computeBalance(workers []WorkerReport) LoadBalance {
	if len(workers) == 0 {
		return LoadBalance{}
	}

	var (
		maxFiles int64 = workers[0].FilesScanned
		minFiles int64 = workers[0].FilesScanned
		sumFiles int64
		maxBytes int64 = workers[0].BytesScanned
		minBytes int64 = workers[0].BytesScanned
		sumBytes int64
	)

	for _, w := range workers {
		if w.FilesScanned > maxFiles { maxFiles = w.FilesScanned }
		if w.FilesScanned < minFiles { minFiles = w.FilesScanned }
		sumFiles += w.FilesScanned
		if w.BytesScanned > maxBytes { maxBytes = w.BytesScanned }
		if w.BytesScanned < minBytes { minBytes = w.BytesScanned }
		sumBytes += w.BytesScanned
	}

	n := float64(len(workers))
	imbalance := 1.0
	if minFiles > 0 {
		imbalance = float64(maxFiles) / float64(minFiles)
	} else if maxFiles > 0 {
		imbalance = float64(maxFiles) // one worker got everything, others got nothing
	}

	ranked := make([]WorkerReport, len(workers))
	copy(ranked, workers)
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].BytesScanned != ranked[j].BytesScanned {
			return ranked[i].BytesScanned > ranked[j].BytesScanned
		}
		return ranked[i].MatchesFound > ranked[j].MatchesFound
	})

	return LoadBalance{
		MaxFiles:  maxFiles,
		MinFiles:  minFiles,
		AvgFiles:  float64(sumFiles) / n,
		MaxBytes:  maxBytes,
		MinBytes:  minBytes,
		AvgBytes:  float64(sumBytes) / n,
		Imbalance: imbalance,
		Ranked:    ranked,
	}
}
