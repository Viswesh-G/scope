package analysis

import (
	"os"
	"testing"

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
	t.Chdir(t.TempDir())

	// save MaxHistory + 5 records and check the oldest ones get trimmed
	for i := 0; i < MaxHistory+5; i++ {
		err := Save(SearchRecord{Pattern: "p", Matches: int64(i)})
		if err != nil {
			t.Fatalf("Save %d failed: %v", i, err)
		}
	}

	records, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != MaxHistory {
		t.Fatalf("history should be capped at %d, got %d", MaxHistory, len(records))
	}
	first := records[0].Matches
	want := int64(5) // records 0..4 should have been dropped
	if first != want {
		t.Errorf("oldest kept record = %d, want %d (newest 1000 of 1005)", first, want)
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
