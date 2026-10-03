// Package analysis also handles saving and loading search history.
//
// This file (history.go) manages the .scope/history.json file. Every time a user
// runs a search, we save a record of it here. This allows us to build features like:
//   - "What are my most common searches?" (RunTop)
//   - "Which searches took the longest?" (RunSlowest)
//   - "Re-run my last search" (Replay)
package analysis

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Viswesh-G/scope/internal/output"
)

const MaxHistory = 1000

const (
	ScopeDir    = ".scope"
	HistoryFile = "history.json"

	// HistoryLock is the name of the little lock file we drop next to the
	// history so two processes cannot rewrite it at the same time.
	HistoryLock = "history.lock"
)

// searchMu guards the read-modify-write steps inside this process. Two
// goroutines can call Save() at once (for example the watcher and a search
// running together), and a plain read-then-write would let one of them
// silently drop the other's record.
var searchMu sync.Mutex

// lockWait is how long we are willing to wait for another process to finish
// writing before we decide its lock file was left behind by a crash.
// It is a variable rather than a constant only so tests can shorten the wait.
var lockWait = 2 * time.Second

// SearchRecord is one entry in the history file.
type SearchRecord struct {
	Timestamp  string  `json:"timestamp"`
	Pattern    string  `json:"pattern"`
	Path       string  `json:"path"`
	Workers    int     `json:"workers"`
	Matches    int64   `json:"matches"`
	DurationMs float64 `json:"duration_ms"`
}

func historyFile() string {
	return filepath.Join(ScopeDir, HistoryFile)
}

// Save appends a record to history and trims if we're over the limit.
func Save(record SearchRecord) error {
	return changeHistory(func(records []SearchRecord) ([]SearchRecord, error) {
		records = append(records, record)
		return trimToCap(records), nil
	})
}

// trimToCap throws away the oldest records once we go over the limit.
// We keep the newest ones because recent searches are the useful ones.
func trimToCap(records []SearchRecord) []SearchRecord {
	if len(records) <= MaxHistory {
		return records
	}
	return records[len(records)-MaxHistory:]
}

// changeHistory runs a read-modify-write on the history file with both locks
// held, then saves whatever the callback returns.
//
// We need two locks because the history file is shared by more than one
// process: `scope watch` and the dashboard both spawn real searches as
// separate processes, so the in-memory mutex is not enough on its own.
func changeHistory(change func([]SearchRecord) ([]SearchRecord, error)) error {
	searchMu.Lock()
	defer searchMu.Unlock()

	if err := os.MkdirAll(ScopeDir, 0755); err != nil {
		return err
	}

	unlock, err := lockHistory()
	if err != nil {
		return err
	}
	defer unlock()

	records, err := Load()
	if err != nil {
		return err
	}

	records, err = change(records)
	if err != nil {
		return err
	}

	return writeHistory(records)
}

// lockHistory creates an empty file that only one process can create at a
// time. os.OpenFile with O_EXCL fails if the file is already there, which is
// exactly the "somebody else got here first" signal we want.
//
// It returns a function that removes the lock, which the caller must always
// run (normally with defer).
func lockHistory() (func(), error) {
	path := filepath.Join(ScopeDir, HistoryLock)
	deadline := time.Now().Add(lockWait)

	for {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err == nil {
			file.Close()
			return func() { os.Remove(path) }, nil
		}
		if !os.IsExist(err) {
			return nil, err
		}

		// Somebody else has the lock. Wait a moment and try again.
		if time.Now().After(deadline) {
			// We waited long enough that the holder is probably a process that
			// was killed mid-write. Delete the stale lock so later runs are not
			// blocked forever, and tell the caller to try once more.
			os.Remove(path)
			return nil, fmt.Errorf("another process left %s behind, removed it, please run the command again", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// writeHistory saves the records to disk.
//
// We write to a temporary file first and then rename it over the real one.
// A rename is atomic on every platform we build for, so a reader (the
// dashboard, or another scope process) either sees the old file or the new
// one, never a half-written mess.
func writeHistory(records []SearchRecord) error {
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}

	final := historyFile()
	temp, err := os.CreateTemp(ScopeDir, "history-*.json")
	if err != nil {
		return err
	}
	// If anything goes wrong from here on we would leave a stray temp file
	// behind, so always clean it up. Removing an already-renamed file is a
	// harmless no-op.
	defer os.Remove(temp.Name())

	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}

	return os.Rename(temp.Name(), final)
}

// Load reads the history file. Returns an empty slice if it doesn't exist yet.
func Load() ([]SearchRecord, error) {
	data, err := os.ReadFile(historyFile())
	if err != nil {
		if os.IsNotExist(err) {
			return []SearchRecord{}, nil
		}
		return nil, err
	}

	var records []SearchRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func RunHistory() error {
	records, err := Load()
	if err != nil {
		return err
	}
	if len(records) == 0 {
		output.PrintSuccess("No history yet.")
		return nil
	}

	output.PrintHeader("Recent Searches", "")

	var rows [][]string
	for i := len(records) - 1; i >= 0; i-- {
		r := records[i]
		rows = append(rows, []string{
			r.Timestamp,
			r.Pattern,
			fmt.Sprintf("%d", r.Matches),
			fmt.Sprintf("%d", r.Workers),
			fmt.Sprintf("%.3f ms", r.DurationMs),
		})
	}

	output.PrintTable(
		[]string{"Timestamp", "Pattern", "Matches", "Workers", "Duration"},
		rows,
		[]string{"left", "left", "right", "right", "right"},
	)
	return nil
}

func RunStats() error {
	records, err := Load()
	if err != nil {
		return err
	}
	if len(records) == 0 {
		output.PrintSuccess("No history yet.")
		return nil
	}

	patterns := make(map[string]struct{})
	var totalMatches int64
	var totalDuration float64
	fastest := records[0].DurationMs
	slowest := records[0].DurationMs

	for _, r := range records {
		patterns[r.Pattern] = struct{}{}
		totalMatches += r.Matches
		totalDuration += r.DurationMs
		if r.DurationMs < fastest {
			fastest = r.DurationMs
		}
		if r.DurationMs > slowest {
			slowest = r.DurationMs
		}
	}

	output.PrintHeader("History Stats", "")
	output.PrintKeyValue("Total Searches", fmt.Sprintf("%d", len(records)))
	output.PrintKeyValue("Unique Patterns", fmt.Sprintf("%d", len(patterns)))
	output.PrintKeyValue("Total Matches", fmt.Sprintf("%d", totalMatches))
	output.PrintKeyValue("Average Duration", fmt.Sprintf("%.3f ms", totalDuration/float64(len(records))))
	output.PrintKeyValue("Fastest Search", fmt.Sprintf("%.3f ms", fastest))
	output.PrintKeyValue("Slowest Search", fmt.Sprintf("%.3f ms", slowest))
	fmt.Println()
	return nil
}

func RunTop() error {
	records, err := Load()
	if err != nil {
		return err
	}
	if len(records) == 0 {
		output.PrintSuccess("No history yet.")
		return nil
	}

	counts := make(map[string]int)
	for _, r := range records {
		counts[r.Pattern]++
	}

	type item struct {
		Pattern string
		Count   int
	}
	var items []item
	for p, c := range counts {
		items = append(items, item{p, c})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Count > items[j].Count })

	output.PrintHeader("Top Patterns", "")
	var rows [][]string
	for _, it := range items {
		rows = append(rows, []string{it.Pattern, fmt.Sprintf("%d", it.Count)})
	}
	output.PrintTable([]string{"Pattern", "Count"}, rows, []string{"left", "right"})
	return nil
}

func RunSlowest() error {
	records, err := Load()
	if err != nil {
		return err
	}
	if len(records) == 0 {
		output.PrintSuccess("No history yet.")
		return nil
	}

	sort.Slice(records, func(i, j int) bool { return records[i].DurationMs > records[j].DurationMs })

	output.PrintHeader("Slowest Searches", "")
	limit := min(10, len(records))
	var rows [][]string
	for i := 0; i < limit; i++ {
		rows = append(rows, []string{records[i].Pattern, fmt.Sprintf("%.3f ms", records[i].DurationMs)})
	}
	output.PrintTable([]string{"Pattern", "Duration"}, rows, []string{"left", "right"})
	return nil
}

func RunFastest() error {
	records, err := Load()
	if err != nil {
		return err
	}
	if len(records) == 0 {
		output.PrintSuccess("No history yet.")
		return nil
	}

	sort.Slice(records, func(i, j int) bool { return records[i].DurationMs < records[j].DurationMs })

	output.PrintHeader("Fastest Searches", "")
	limit := min(10, len(records))
	var rows [][]string
	for i := 0; i < limit; i++ {
		rows = append(rows, []string{records[i].Pattern, fmt.Sprintf("%.3f ms", records[i].DurationMs)})
	}
	output.PrintTable([]string{"Pattern", "Duration"}, rows, []string{"left", "right"})
	return nil
}

func RunRecent(limit int) error {
	records, err := Load()
	if err != nil {
		return err
	}
	if len(records) == 0 {
		output.PrintSuccess("No history yet.")
		return nil
	}

	if limit > len(records) {
		limit = len(records)
	}

	output.PrintHeader("Recent Searches", "")
	var rows [][]string
	for i := len(records) - 1; i >= len(records)-limit; i-- {
		rows = append(rows, []string{records[i].Timestamp, records[i].Pattern})
	}
	output.PrintTable([]string{"Timestamp", "Pattern"}, rows, []string{"left", "left"})
	return nil
}

func RunPattern(pattern string) error {
	records, err := Load()
	if err != nil {
		return err
	}

	var rows [][]string
	for _, r := range records {
		if strings.Contains(r.Pattern, pattern) {
			rows = append(rows, []string{r.Timestamp, r.Pattern, fmt.Sprintf("%d", r.Matches)})
		}
	}

	if len(rows) == 0 {
		output.PrintSuccess(fmt.Sprintf("No history matching %q.", pattern))
		return nil
	}

	output.PrintHeader(fmt.Sprintf("History for pattern: %s", pattern), "")
	output.PrintTable([]string{"Timestamp", "Pattern", "Matches"}, rows, []string{"left", "left", "right"})
	return nil
}

func RunPath(path string) error {
	records, err := Load()
	if err != nil {
		return err
	}

	var rows [][]string
	for _, r := range records {
		if strings.Contains(r.Path, path) {
			rows = append(rows, []string{r.Timestamp, r.Pattern, r.Path})
		}
	}

	if len(rows) == 0 {
		output.PrintSuccess(fmt.Sprintf("No history for path %q.", path))
		return nil
	}

	output.PrintHeader(fmt.Sprintf("History for path: %s", path), "")
	output.PrintTable([]string{"Timestamp", "Pattern", "Path"}, rows, []string{"left", "left", "left"})
	return nil
}

func Export(dst string) error {
	records, err := Load()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func Prune(limit int) error {
	return changeHistory(func(records []SearchRecord) ([]SearchRecord, error) {
		if len(records) <= limit {
			// Nothing to throw away, so hand the same list back untouched.
			return records, nil
		}
		return records[len(records)-limit:], nil
	})
}

func Clear() error {
	searchMu.Lock()
	defer searchMu.Unlock()

	if err := os.MkdirAll(ScopeDir, 0755); err != nil {
		return err
	}

	unlock, err := lockHistory()
	if err != nil {
		return err
	}
	defer unlock()

	if err := os.Remove(historyFile()); err != nil && !os.IsNotExist(err) {
		return err
	}
	output.PrintSuccess("History cleared.")
	return nil
}

// Replay re-runs a past search by picking the Nth most recent record and
// re-executing scope search with the same flags. nth=1 means most recent.
// We re-exec the current binary so the output looks exactly like a normal search.
func Replay(nth int) error {
	records, err := Load()
	if err != nil {
		return err
	}
	if len(records) == 0 {
		output.PrintWarning("No history to replay.")
		return nil
	}

	// nth is 1-indexed from the end (1 = newest)
	if nth < 1 || nth > len(records) {
		return fmt.Errorf("--nth %d out of range (history has %d entries)", nth, len(records))
	}
	r := records[len(records)-nth]

	output.PrintHeader("Replaying search", "")
	output.PrintKeyValue("Pattern", r.Pattern)
	output.PrintKeyValue("Path", r.Path)
	output.PrintKeyValue("Workers", fmt.Sprintf("%d", r.Workers))
	fmt.Println()

	// re-exec the scope binary with the same args
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("couldn't find scope binary: %w", err)
	}

	args := []string{
		"search",
		"-p", r.Pattern,
		"--path", r.Path,
		"-w", fmt.Sprintf("%d", r.Workers),
	}

	cmd := exec.Command(self, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
