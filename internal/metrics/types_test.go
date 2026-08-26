package metrics

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestNewRegistryCreatesWorkers(t *testing.T) {
	r := NewRegistry(4)
	if len(r.Workers) != 4 {
		t.Fatalf("got %d workers, want 4", len(r.Workers))
	}
	for i, w := range r.Workers {
		if w.ID != i {
			t.Errorf("worker[%d].ID = %d, want %d", i, w.ID, i)
		}
	}
}

func TestNewRegistryZeroWorkers(t *testing.T) {
	r := NewRegistry(0)
	if len(r.Workers) != 0 {
		t.Errorf("expected 0 workers, got %d", len(r.Workers))
	}
}

func TestBuildReportSnapshotsCounters(t *testing.T) {
	r := NewRegistry(2)
	atomic.AddInt64(&r.DirsScanned, 3)
	atomic.AddInt64(&r.FilesScanned, 10)
	atomic.AddInt64(&r.FilesIgnored, 2)
	atomic.AddInt64(&r.MatchesFound, 7)

	w0 := r.Workers[0]
	atomic.AddInt64(&w0.FilesScanned, 6)
	atomic.AddInt64(&w0.MatchesFound, 5)
	atomic.AddInt64(&w0.BytesScanned, 1000)
	atomic.AddInt64(&w0.WorkTimeNs, 2_000_000) // 2ms

	w1 := r.Workers[1]
	atomic.AddInt64(&w1.FilesScanned, 4)
	atomic.AddInt64(&w1.MatchesFound, 2)
	atomic.AddInt64(&w1.BytesScanned, 500)
	atomic.AddInt64(&w1.WorkTimeNs, 1_000_000) // 1ms

	report := BuildReport(r)

	if report.DirsScanned != 3 || report.FilesScanned != 10 ||
		report.FilesIgnored != 2 || report.MatchesFound != 7 {
		t.Errorf("registry counters didn't come through: %+v", report)
	}

	if len(report.Workers) != 2 {
		t.Fatalf("got %d worker reports, want 2", len(report.Workers))
	}

	// worker 0 scanned 1000 bytes in 2ms = 500,000 bytes/sec
	if report.Workers[0].Throughput != 500000 {
		t.Errorf("worker 0 throughput = %f, want 500000", report.Workers[0].Throughput)
	}
}

func TestBuildReportLoadBalance(t *testing.T) {
	r := NewRegistry(2)
	w0 := r.Workers[0]
	w1 := r.Workers[1]
	atomic.StoreInt64(&w0.FilesScanned, 8)
	atomic.StoreInt64(&w1.FilesScanned, 2)
	atomic.StoreInt64(&w0.BytesScanned, 800)
	atomic.StoreInt64(&w1.BytesScanned, 200)

	report := BuildReport(r)
	bal := report.Balance

	if bal.MaxFiles != 8 || bal.MinFiles != 2 {
		t.Errorf("max/min files = %d/%d, want 8/2", bal.MaxFiles, bal.MinFiles)
	}
	if bal.AvgFiles != 5.0 {
		t.Errorf("avg files = %f, want 5", bal.AvgFiles)
	}
	// imbalance is max/min = 8/2 = 4x
	if bal.Imbalance != 4.0 {
		t.Errorf("imbalance = %f, want 4", bal.Imbalance)
	}
	// busiest worker (more bytes) should be ranked first
	if len(bal.Ranked) != 2 || bal.Ranked[0].ID != 0 {
		t.Errorf("ranked[0] should be worker 0 (most bytes), got %+v", bal.Ranked)
	}
}

func TestBuildReportEmptyBalance(t *testing.T) {
	r := NewRegistry(0)
	report := BuildReport(r)
	if report.Balance.Imbalance != 0 || report.Balance.Ranked != nil {
		t.Errorf("empty registry should give a zero balance, got %+v", report.Balance)
	}
}

// make sure concurrent atomic updates don't trip the race detector
func TestConcurrentCounterUpdates(t *testing.T) {
	r := NewRegistry(4)
	var wg sync.WaitGroup
	for _, w := range r.Workers {
		wg.Add(1)
		go func(w *WorkerStats) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				atomic.AddInt64(&w.FilesScanned, 1)
				atomic.AddInt64(&r.MatchesFound, 1)
				w.Mu.Lock()
				w.Files = append(w.Files, "file.txt")
				w.Mu.Unlock()
			}
		}(w)
	}
	wg.Wait()

	var total int64
	for _, w := range r.Workers {
		total += atomic.LoadInt64(&w.FilesScanned)
	}
	if total != 400 {
		t.Errorf("total files scanned = %d, want 400", total)
	}
	if r.MatchesFound != 400 {
		t.Errorf("matches found = %d, want 400", r.MatchesFound)
	}
}
