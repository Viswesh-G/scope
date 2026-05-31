package metrics

import (
	"sort"
	"sync/atomic"
	"time"
)

type Report struct {
	DirsScanned  int64
	FilesScanned int64
	FilesIgnored int64
	MatchesFound int64

	WalkDuration   time.Duration
	SearchDuration time.Duration
	TotalDuration  time.Duration

	Parallelism float64

	Workers []WorkerReport

	Balance LoadBalance
}

type WorkerReport struct {
	ID int

	FilesScanned int64
	MatchesFound int64
	BytesScanned int64
	WorkDuration time.Duration

	// Throughput in bytes per second (0 if WorkDuration == 0)
	Throughput float64

	Files []string
}

// LoadBalance summarises how evenly work was distributed across workers.
type LoadBalance struct {
	MaxFiles  int64
	MinFiles  int64
	AvgFiles  float64
	MaxBytes  int64
	MinBytes  int64
	AvgBytes  float64
	Imbalance float64 // maxFiles / minFiles (or 1 when minFiles == 0)

	// Workers ranked by bytes scanned descending.
	Ranked []WorkerReport
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
		if w.FilesScanned > maxFiles {
			maxFiles = w.FilesScanned
		}
		if w.FilesScanned < minFiles {
			minFiles = w.FilesScanned
		}
		sumFiles += w.FilesScanned

		if w.BytesScanned > maxBytes {
			maxBytes = w.BytesScanned
		}
		if w.BytesScanned < minBytes {
			minBytes = w.BytesScanned
		}
		sumBytes += w.BytesScanned
	}

	n := float64(len(workers))
	imbalance := 1.0
	if minFiles > 0 {
		imbalance = float64(maxFiles) / float64(minFiles)
	} else if maxFiles > 0 {
		imbalance = float64(maxFiles) // any/0 → show max as imbalance
	}

	ranked := make([]WorkerReport, len(workers))
	copy(ranked, workers)
	sort.Slice(ranked, func(i, j int) bool {
		// Primary: bytes scanned desc; secondary: matches desc
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
