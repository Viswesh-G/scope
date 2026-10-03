package analysis

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCountScopeMatches(t *testing.T) {
	got, err := countScopeMatches([]byte(`[{"file":"a.go","line":1,"content":"hit"},{"file":"b.go","line":2,"content":"hit"}]`))
	if err != nil {
		t.Fatalf("countScopeMatches returned an error: %v", err)
	}
	if got != 2 {
		t.Errorf("got %d matches, want 2", got)
	}
}

func TestCountRipgrepMatches(t *testing.T) {
	data := []byte("{\"type\":\"begin\"}\n{\"type\":\"match\"}\n{\"type\":\"context\"}\n{\"type\":\"match\"}\n{\"type\":\"end\"}\n")
	got, err := countRipgrepMatches(data)
	if err != nil {
		t.Fatalf("countRipgrepMatches returned an error: %v", err)
	}
	if got != 2 {
		t.Errorf("got %d matches, want 2", got)
	}
}

func TestCountsEqual(t *testing.T) {
	if !countsEqual([]int{0, 2, 2}, []int{0, 2, 2}) {
		t.Error("identical per-run counts should match")
	}
	if countsEqual([]int{1, 2}, []int{1, 3}) {
		t.Error("different per-run counts should not match")
	}
	if countsEqual([]int{1}, []int{1, 1}) {
		t.Error("different run counts should not match")
	}
	if !countsStable([]int{2, 2, 2}) {
		t.Error("identical results across runs should be stable")
	}
	if countsStable([]int{2, 2, 3}) {
		t.Error("a changed count across runs should be unstable")
	}
}

func TestTrimmedMeanAndMannWhitneyPValue(t *testing.T) {
	samples := []time.Duration{
		time.Millisecond,
		2 * time.Millisecond,
		3 * time.Millisecond,
		4 * time.Millisecond,
		100 * time.Millisecond,
	}
	if got := trimmedMean(samples, 0.2); got != 3*time.Millisecond {
		t.Errorf("trimmedMean = %s, want 3ms", got)
	}
	if got := mannWhitneyPValue(samples, samples); got < 0.99 {
		t.Errorf("identical sample sets should not show a difference, p-value = %.4f", got)
	}
}

func TestSaveCompareReportIncludesReproducibilityData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "compare.json")
	report := CompareReport{
		GeneratedAt:      time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC),
		Pattern:          "TODO",
		Path:             ".",
		Runs:             2,
		Warmup:           1,
		Workers:          4,
		Globs:            []string{"*.go", "!*_test.go"},
		CacheCondition:   "best-effort warm cache",
		GoVersion:        "go1.25.6",
		OS:               "windows",
		Architecture:     "amd64",
		LogicalCPUs:      8,
		CPUModel:         "not collected",
		RepositoryCommit: "abc123",
		RepositoryDirty:  "true",
		RepositoryBytes:  1000,
		RepositoryFiles:  12,
		MatchCountsEqual: true,
		Scope: CompareResult{
			Tool: "Scope", Version: "scope v0.5.0",
			MatchCount:  3,
			MatchCounts: []int{3, 3},
			Samples:     []time.Duration{time.Millisecond, 2 * time.Millisecond},
			Mean:        1500 * time.Microsecond,
			TrimmedMean: 1500 * time.Microsecond,
			P99:         2 * time.Millisecond,
		},
		Ripgrep: CompareResult{
			Tool: "Ripgrep", Version: "ripgrep 15.0.0",
			MatchCount:  3,
			MatchCounts: []int{3, 3},
			Samples:     []time.Duration{500 * time.Microsecond, time.Millisecond},
			Mean:        750 * time.Microsecond,
		},
	}

	if err := SaveCompareReport(path, report); err != nil {
		t.Fatalf("SaveCompareReport returned an error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading saved report: %v", err)
	}
	var artifact map[string]json.RawMessage
	if err := json.Unmarshal(data, &artifact); err != nil {
		t.Fatalf("saved report is not valid JSON: %v", err)
	}
	for _, key := range []string{"environment", "repository", "search_rules", "scope", "ripgrep", "cache_condition"} {
		if _, ok := artifact[key]; !ok {
			t.Errorf("saved report is missing %q", key)
		}
	}

	var scope compareArtifactResult
	if err := json.Unmarshal(artifact["scope"], &scope); err != nil {
		t.Fatalf("reading scope result from report: %v", err)
	}
	if scope.MatchCount != 3 || len(scope.SamplesMS) != 2 || scope.MeanMS != 1.5 {
		t.Errorf("saved scope results do not preserve counts and millisecond samples: %+v", scope)
	}
	if scope.TrimmedMeanMS != 1.5 || scope.P99MS == 0 {
		t.Errorf("saved scope results do not preserve summary statistics: %+v", scope)
	}
}

func TestRepositorySize(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "one.txt"), []byte("123"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "two.txt"), []byte("4567"), 0644); err != nil {
		t.Fatal(err)
	}

	size, files, err := repositorySize(root)
	if err != nil {
		t.Fatalf("repositorySize returned an error: %v", err)
	}
	if size != 7 || files != 2 {
		t.Errorf("repositorySize = (%d bytes, %d files), want (7 bytes, 2 files)", size, files)
	}
}
