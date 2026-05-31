package metrics

import (
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
}

type WorkerReport struct {
	ID int

	FilesScanned int64
	MatchesFound int64
	WorkDuration time.Duration

	Files []string
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

		report.Workers = append(report.Workers, WorkerReport{
			ID:           w.ID,
			FilesScanned: atomic.LoadInt64(&w.FilesScanned),
			MatchesFound: atomic.LoadInt64(&w.MatchesFound),
			WorkDuration: time.Duration(atomic.LoadInt64(&w.WorkTimeNs)),
			Files:        files,
		})
	}

	return report
}
