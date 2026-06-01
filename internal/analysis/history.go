package analysis

import (
	"encoding/json"
	"fmt"
	"os"
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

type SearchRecord struct {
	Timestamp  string  `json:"timestamp"`
	Pattern    string  `json:"pattern"`
	Path       string  `json:"path"`
	Workers    int     `json:"workers"`
	Matches    int64   `json:"matches"`
	DurationMs float64 `json:"duration_ms"`
}

func historyFile() string {
	return filepath.Join(
		ScopeDir,
		HistoryFile,
	)
}

func Save(record SearchRecord) error {

	path := historyFile()

	if err := os.MkdirAll(
		filepath.Dir(path),
		0755,
	); err != nil {
		return err
	}

	records, _ := Load()

	records = append(records, record)

	if len(records) > MaxHistory {
		records = records[len(records)-MaxHistory:]
	}

	data, err := json.MarshalIndent(
		records,
		"",
		"  ",
	)
	if err != nil {
		return err
	}

	return os.WriteFile(
		path,
		data,
		0644,
	)
}

func Load() ([]SearchRecord, error) {

	path := historyFile()

	data, err := os.ReadFile(path)

	if err != nil {

		if os.IsNotExist(err) {
			return []SearchRecord{}, nil
		}

		return nil, err
	}

	var records []SearchRecord

	if err := json.Unmarshal(
		data,
		&records,
	); err != nil {
		return nil, err
	}

	return records, nil
}

// =====================================================
// history
// =====================================================

func RunHistory() error {
	records, err := Load()
	if err != nil {
		return err
	}

	if len(records) == 0 {
		output.PrintSuccess("No history.")
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

// =====================================================
// history stats
// =====================================================

func RunStats() error {
	records, err := Load()
	if err != nil {
		return err
	}

	if len(records) == 0 {
		output.PrintSuccess("No history.")
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
	output.PrintKeyValue("Average Runtime", fmt.Sprintf("%.3f ms", totalDuration/float64(len(records))))
	output.PrintKeyValue("Fastest Search", fmt.Sprintf("%.3f ms", fastest))
	output.PrintKeyValue("Slowest Search", fmt.Sprintf("%.3f ms", slowest))
	fmt.Println()

	return nil
}

// =====================================================
// history top
// =====================================================

func RunTop() error {
	records, err := Load()
	if err != nil {
		return err
	}

	if len(records) == 0 {
		output.PrintSuccess("No history.")
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
		items = append(items, item{Pattern: p, Count: c})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Count > items[j].Count
	})

	output.PrintHeader("Top Patterns", "")

	var rows [][]string
	for _, item := range items {
		rows = append(rows, []string{item.Pattern, fmt.Sprintf("%d", item.Count)})
	}

	output.PrintTable(
		[]string{"Pattern", "Count"},
		rows,
		[]string{"left", "right"},
	)

	return nil
}

// =====================================================
// history slowest
// =====================================================

func RunSlowest() error {
	records, err := Load()
	if err != nil {
		return err
	}

	if len(records) == 0 {
		output.PrintSuccess("No history.")
		return nil
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].DurationMs > records[j].DurationMs
	})

	output.PrintHeader("Slowest Searches", "")
	limit := min(10, len(records))

	var rows [][]string
	for i := 0; i < limit; i++ {
		r := records[i]
		rows = append(rows, []string{r.Pattern, fmt.Sprintf("%.3f ms", r.DurationMs)})
	}

	output.PrintTable(
		[]string{"Pattern", "Duration"},
		rows,
		[]string{"left", "right"},
	)

	return nil
}

// =====================================================
// history fastest
// =====================================================

func RunFastest() error {
	records, err := Load()
	if err != nil {
		return err
	}

	if len(records) == 0 {
		output.PrintSuccess("No history.")
		return nil
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].DurationMs < records[j].DurationMs
	})

	output.PrintHeader("Fastest Searches", "")
	limit := min(10, len(records))

	var rows [][]string
	for i := 0; i < limit; i++ {
		r := records[i]
		rows = append(rows, []string{r.Pattern, fmt.Sprintf("%.3f ms", r.DurationMs)})
	}

	output.PrintTable(
		[]string{"Pattern", "Duration"},
		rows,
		[]string{"left", "right"},
	)

	return nil
}

// =====================================================
// history recent
// =====================================================

func RunRecent(limit int) error {
	records, err := Load()
	if err != nil {
		return err
	}

	if len(records) == 0 {
		output.PrintSuccess("No history.")
		return nil
	}

	if limit > len(records) {
		limit = len(records)
	}

	output.PrintHeader("Recent Searches", "")
	var rows [][]string
	for i := len(records) - 1; i >= len(records)-limit; i-- {
		r := records[i]
		rows = append(rows, []string{r.Timestamp, r.Pattern})
	}

	output.PrintTable(
		[]string{"Timestamp", "Pattern"},
		rows,
		[]string{"left", "left"},
	)

	return nil
}

// =====================================================
// history pattern
// =====================================================

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
		output.PrintSuccess(fmt.Sprintf("No history for pattern '%s'.", pattern))
		return nil
	}

	output.PrintHeader(fmt.Sprintf("History for pattern: %s", pattern), "")
	output.PrintTable(
		[]string{"Timestamp", "Pattern", "Matches"},
		rows,
		[]string{"left", "left", "right"},
	)

	return nil
}

// =====================================================
// history path
// =====================================================

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
		output.PrintSuccess(fmt.Sprintf("No history for path '%s'.", path))
		return nil
	}

	output.PrintHeader(fmt.Sprintf("History for path: %s", path), "")
	output.PrintTable(
		[]string{"Timestamp", "Pattern", "Path"},
		rows,
		[]string{"left", "left", "left"},
	)

	return nil
}

// =====================================================
// history export
// =====================================================

func Export(dst string) error {

	records, err := Load()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(
		records,
		"",
		"  ",
	)
	if err != nil {
		return err
	}

	return os.WriteFile(
		dst,
		data,
		0644,
	)
}

// =====================================================
// history prune
// =====================================================

func Prune(limit int) error {

	records, err := Load()
	if err != nil {
		return err
	}

	if len(records) <= limit {
		return nil
	}

	records = records[len(records)-limit:]

	data, err := json.MarshalIndent(
		records,
		"",
		"  ",
	)
	if err != nil {
		return err
	}

	return os.WriteFile(
		historyFile(),
		data,
		0644,
	)
}

// =====================================================
// history clear
// =====================================================

func Clear() error {
	err := os.Remove(historyFile())
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	output.PrintSuccess("History cleared.")
	return nil
}
