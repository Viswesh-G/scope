// Package analysis is everything that isn't the search engine itself:
// history, benchmarking, file stats, dep counting, duplicate detection, and the directory graph.
package analysis

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Viswesh-G/scope/internal/output"
)

const MaxHistory = 1000

const (
	ScopeDir    = ".scope"
	HistoryFile = "history.json"
)

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
	path := historyFile()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	records, _ := Load()
	records = append(records, record)

	if len(records) > MaxHistory {
		records = records[len(records)-MaxHistory:]
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
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
	records, err := Load()
	if err != nil {
		return err
	}
	if len(records) <= limit {
		return nil
	}

	records = records[len(records)-limit:]
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(historyFile(), data, 0644)
}

func Clear() error {
	err := os.Remove(historyFile())
	if err != nil && !os.IsNotExist(err) {
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
