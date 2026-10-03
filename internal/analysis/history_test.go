package analysis

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Viswesh-G/scope/internal/output"
)

// some history commands print colored messages, so the global color
// variables must exist first. normally cmd/root.go handles this.
func TestMain(m *testing.M) {
	output.InitColors()
	os.Exit(m.Run())
}

// t.Chdir moves into a temp dir for the test, so .scope/history.json
// gets written there instead of polluting the real repo.
func TestSaveAndLoadRoundTrip(t *testing.T) {
	t.Chdir(t.TempDir())

	record := SearchRecord{
		Timestamp:  "2026-08-26T12:00:00Z",
		Pattern:    "TODO",
		Path:       "./internal",
		Workers:    4,
		Matches:    12,
		DurationMs: 3.5,
	}
	if err := Save(record); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	records, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	got := records[0]
	if got.Pattern != record.Pattern || got.Path != record.Path ||
		got.Workers != record.Workers || got.Matches != record.Matches ||
		got.DurationMs != record.DurationMs {
		t.Errorf("round trip changed the record:\ngot  %+v\nwant %+v", got, record)
	}
}

func TestLoadEmptyHistory(t *testing.T) {
	t.Chdir(t.TempDir())
	records, err := Load()
	if err != nil {
		t.Fatalf("Load on fresh dir should not error, got %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected empty history, got %d records", len(records))
	}
}

func TestSaveAppendsAndTrims(t *testing.T) {
	// We check the trimming rule on a plain slice instead of writing 1005
	// records to disk, because every save re-reads and re-writes the whole
	// file. Doing that 1005 times turns a fast test into a 20 second one.
	records := make([]SearchRecord, 0, MaxHistory+5)
	for i := 0; i < MaxHistory+5; i++ {
		records = append(records, SearchRecord{Pattern: "p", Matches: int64(i)})
	}

	trimmed := trimToCap(records)
	if len(trimmed) != MaxHistory {
		t.Fatalf("history should be capped at %d, got %d", MaxHistory, len(trimmed))
	}
	if trimmed[0].Matches != 5 {
		t.Errorf("oldest kept record = %d, want 5 (the newest %d of %d)", trimmed[0].Matches, MaxHistory, MaxHistory+5)
	}
	if trimmed[len(trimmed)-1].Matches != int64(MaxHistory+4) {
		t.Errorf("newest record = %d, want %d", trimmed[len(trimmed)-1].Matches, MaxHistory+4)
	}
}

func TestTrimToCapLeavesShortListsAlone(t *testing.T) {
	records := []SearchRecord{{Pattern: "a"}, {Pattern: "b"}}
	if got := trimToCap(records); len(got) != 2 {
		t.Errorf("a list under the cap should come back untouched, got %d records", len(got))
	}
}

func TestSaveAppendsInOrder(t *testing.T) {
	// The same thing as above but through the real file, so we know the
	// appending itself works and old entries are not reordered.
	t.Chdir(t.TempDir())

	for i := 0; i < 3; i++ {
		if err := Save(SearchRecord{Pattern: "p", Matches: int64(i)}); err != nil {
			t.Fatal(err)
		}
	}

	records, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 {
		t.Fatalf("got %d records, want 3", len(records))
	}
	for i, r := range records {
		if r.Matches != int64(i) {
			t.Errorf("record %d = %d, want %d", i, r.Matches, i)
		}
	}
}

func TestClearRemovesHistory(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := Save(SearchRecord{Pattern: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
	if _, err := os.Stat(historyFile()); !os.IsNotExist(err) {
		t.Error("history file should be gone after Clear")
	}
	// clearing again (nothing to delete) should still succeed
	if err := Clear(); err != nil {
		t.Errorf("Clear on empty history should not error, got %v", err)
	}
}

func TestPruneKeepsNewest(t *testing.T) {
	t.Chdir(t.TempDir())

	for i := 0; i < 10; i++ {
		if err := Save(SearchRecord{Pattern: "p", Matches: int64(i)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := Prune(3); err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	records, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 {
		t.Fatalf("after pruning got %d records, want 3", len(records))
	}
	// matches 7, 8, 9 are the newest three
	if records[0].Matches != 7 || records[2].Matches != 9 {
		t.Errorf("prune kept the wrong records: %+v", records)
	}
}

func TestPruneUnderLimitDoesNothing(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := Save(SearchRecord{Pattern: "p"}); err != nil {
		t.Fatal(err)
	}
	if err := Prune(100); err != nil {
		t.Fatalf("Prune failed: %v", err)
	}
	records, _ := Load()
	if len(records) != 1 {
		t.Errorf("pruning below the limit should change nothing, got %d records", len(records))
	}
}

func TestSaveFromManyGoroutinesKeepsEveryRecord(t *testing.T) {
	// This is the bug that used to bite us: the watcher and a normal search
	// can both write history at the same time, and without a lock one of the
	// records would silently disappear.
	t.Chdir(t.TempDir())

	const writers = 15
	const each = 5

	var wg sync.WaitGroup
	errs := make(chan error, writers*each)

	for w := 0; w < writers; w++ {
		for i := 0; i < each; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := Save(SearchRecord{Pattern: "concurrent"}); err != nil {
					errs <- err
				}
			}()
		}
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent Save failed: %v", err)
	}

	records, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	want := writers * each
	if len(records) != want {
		t.Errorf("got %d records, want %d - a concurrent write was lost", len(records), want)
	}
}

func TestLockHistoryRefusesWhileAnotherProcessHoldsIt(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(ScopeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Pretend another process is halfway through a write.
	lockPath := filepath.Join(ScopeDir, HistoryLock)
	if err := os.WriteFile(lockPath, nil, 0644); err != nil {
		t.Fatal(err)
	}

	// Do not sit around for the full production wait in a unit test.
	old := lockWait
	lockWait = 50 * time.Millisecond
	defer func() { lockWait = old }()

	release, err := lockHistory()
	if err == nil {
		release()
		t.Fatal("taking the lock should fail while another process holds it")
	}
	if !strings.Contains(err.Error(), "please run the command again") {
		t.Errorf("error should tell the user what to do, got: %v", err)
	}
}

func TestStaleLockIsCleanedUp(t *testing.T) {
	// A process killed mid-write leaves its lock file behind forever. The next
	// run has to notice and clear it, otherwise the tool is bricked.
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(ScopeDir, 0755); err != nil {
		t.Fatal(err)
	}

	lockPath := filepath.Join(ScopeDir, HistoryLock)
	if err := os.WriteFile(lockPath, nil, 0644); err != nil {
		t.Fatal(err)
	}

	old := lockWait
	lockWait = 50 * time.Millisecond
	defer func() { lockWait = old }()

	// The first attempt reports the stuck lock and removes it.
	if _, err := lockHistory(); err == nil {
		t.Fatal("expected the first attempt to report the stuck lock")
	}
	if _, statErr := os.Stat(lockPath); !os.IsNotExist(statErr) {
		t.Error("the stale lock file should have been removed")
	}

	// The next attempt now succeeds and cleans up after itself.
	release, err := lockHistory()
	if err != nil {
		t.Fatalf("second attempt should be able to take the lock, got %v", err)
	}
	release()
	if _, statErr := os.Stat(lockPath); !os.IsNotExist(statErr) {
		t.Error("releasing the lock should remove the lock file")
	}
}

func TestSaveLeavesNoStrayFilesBehind(t *testing.T) {
	// We write to a temp file and rename it over the real one, so a reader
	// never sees a half-written file. The temp file must not survive either.
	t.Chdir(t.TempDir())

	for i := 0; i < 3; i++ {
		if err := Save(SearchRecord{Pattern: "p", Matches: int64(i)}); err != nil {
			t.Fatal(err)
		}
	}

	entries, err := os.ReadDir(ScopeDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if name != HistoryFile && name != HistoryLock {
			t.Errorf("unexpected leftover file in %s: %s", ScopeDir, name)
		}
	}

	records, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 {
		t.Fatalf("got %d records, want 3", len(records))
	}
	for i, r := range records {
		if r.Matches != int64(i) {
			t.Errorf("record %d has Matches = %d, want %d - the file was not replaced cleanly", i, r.Matches, i)
		}
	}
}

func TestLoadReportsCorruptFile(t *testing.T) {
	// Saying the file is broken is better than quietly returning nothing,
	// because silently losing history is much harder to notice.
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(ScopeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(historyFile(), []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(); err == nil {
		t.Error("loading a corrupt history file should return an error, got nil")
	}
}

func TestExportWritesTheFile(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := Save(SearchRecord{Pattern: "keep", Matches: 7}); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(t.TempDir(), "exported.json")
	if err := Export(dst); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("reading exported file: %v", err)
	}
	if !strings.Contains(string(data), `"keep"`) {
		t.Errorf("exported file does not contain the saved record:\n%s", data)
	}
}

func TestReplayRejectsOutOfRangeIndex(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := Save(SearchRecord{Pattern: "only"}); err != nil {
		t.Fatal(err)
	}

	if err := Replay(0); err == nil {
		t.Error("--nth 0 should be rejected, nth is 1-based")
	}
	if err := Replay(2); err == nil {
		t.Error("--nth 2 should be rejected when history only has 1 entry")
	}
}
