package analysis

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Viswesh-G/scope/internal/output"
)

type CompareOptions struct {
	Pattern string
	Path    string
	Runs    int
	Warmup  int
	Workers int
	Globs   []string
}

type CompareResult struct {
	Tool        string
	Version     string
	MatchCount  int
	MatchCounts []int
	Samples     []time.Duration
	Mean        time.Duration
	TrimmedMean time.Duration
	P50         time.Duration
	P95         time.Duration
	P99         time.Duration
	Min         time.Duration
	Max         time.Duration
	StdDev      time.Duration
}

type CompareReport struct {
	GeneratedAt       time.Time
	Pattern           string
	Path              string
	WorkingDirectory  string
	Runs              int
	Warmup            int
	Workers           int
	Globs             []string
	CacheCondition    string
	GoVersion         string
	OS                string
	Architecture      string
	LogicalCPUs       int
	CPUModel          string
	RepositoryCommit  string
	RepositoryDirty   string
	RepositoryBytes   int64
	RepositoryFiles   int64
	MatchCountsEqual  bool
	MatchCountsStable bool
	Scope             CompareResult
	Ripgrep           CompareResult
}

type compareArtifact struct {
	SchemaVersion     int                   `json:"schema_version"`
	GeneratedAt       time.Time             `json:"generated_at"`
	Pattern           string                `json:"pattern"`
	Path              string                `json:"path"`
	WorkingDirectory  string                `json:"working_directory"`
	Runs              int                   `json:"runs"`
	Warmup            int                   `json:"warmup_runs"`
	Workers           int                   `json:"workers"`
	Globs             []string              `json:"globs"`
	CacheCondition    string                `json:"cache_condition"`
	Environment       compareEnvironment    `json:"environment"`
	Repository        compareRepository     `json:"repository"`
	SearchRules       compareSearchRules    `json:"search_rules"`
	MatchCountsEqual  bool                  `json:"match_counts_equal"`
	MatchCountsStable bool                  `json:"match_counts_stable"`
	Scope             compareArtifactResult `json:"scope"`
	Ripgrep           compareArtifactResult `json:"ripgrep"`
}

type compareSearchRules struct {
	Scope       string `json:"scope"`
	Ripgrep     string `json:"ripgrep"`
	Limitations string `json:"limitations"`
}

type compareEnvironment struct {
	GoVersion        string `json:"go_version"`
	OS               string `json:"os"`
	Architecture     string `json:"architecture"`
	LogicalCPUs      int    `json:"logical_cpus"`
	CPUModel         string `json:"cpu_model"`
	RepositoryCommit string `json:"repository_commit"`
	RepositoryDirty  string `json:"repository_dirty"`
}

type compareRepository struct {
	SizeBytes        int64  `json:"size_bytes"`
	FileCount        int64  `json:"file_count"`
	MeasurementScope string `json:"measurement_scope"`
}

type compareArtifactResult struct {
	Version       string    `json:"version"`
	MeanMS        float64   `json:"mean_ms"`
	TrimmedMeanMS float64   `json:"trimmed_mean_10_percent_ms"`
	P50MS         float64   `json:"p50_ms"`
	P95MS         float64   `json:"p95_ms"`
	P99MS         float64   `json:"p99_ms"`
	MinMS         float64   `json:"min_ms"`
	MaxMS         float64   `json:"max_ms"`
	StdDevMS      float64   `json:"stddev_ms"`
	MatchCount    int       `json:"match_count"`
	MatchCounts   []int     `json:"match_counts_by_run"`
	SamplesMS     []float64 `json:"samples_ms"`
}

type rgEvent struct {
	Type string `json:"type"`
}

type commandResult struct {
	Duration   time.Duration
	MatchCount int
}

func selfBinary() string {
	exe, err := os.Executable()
	if err != nil {
		return "scope"
	}
	return exe
}

func Compare(options CompareOptions) (CompareReport, error) {
	if options.Pattern == "" {
		return CompareReport{}, fmt.Errorf("pattern must not be empty")
	}
	if options.Path == "" {
		return CompareReport{}, fmt.Errorf("path must not be empty")
	}
	if options.Runs < 1 {
		return CompareReport{}, fmt.Errorf("runs must be at least 1")
	}
	if options.Warmup < 0 {
		return CompareReport{}, fmt.Errorf("warmup must not be negative")
	}
	if options.Workers < 1 {
		return CompareReport{}, fmt.Errorf("workers must be at least 1")
	}

	repoBytes, repoFiles, err := repositorySize(options.Path)
	if err != nil {
		return CompareReport{}, err
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return CompareReport{}, fmt.Errorf("getting working directory: %w", err)
	}

	scope := selfBinary()
	scopeVersion, err := commandVersion(scope, "version")
	if err != nil {
		return CompareReport{}, fmt.Errorf("getting scope version: %w", err)
	}
	rgVersion, err := commandVersion("rg", "--version")
	if err != nil {
		return CompareReport{}, fmt.Errorf("getting ripgrep version: %w", err)
	}

	scopeArgs := []string{
		"search", "-p", options.Pattern, "--path", options.Path,
		"--workers", strconv.Itoa(options.Workers),
		"--no-history", "--no-config", "--quiet", "--json",
	}
	rgArgs := []string{
		"--json", "--threads", strconv.Itoa(options.Workers),
	}
	for _, glob := range options.Globs {
		scopeArgs = append(scopeArgs, "--glob", glob)
		rgArgs = append(rgArgs, "--glob", glob)
	}
	rgArgs = append(rgArgs, options.Pattern, options.Path)

	for i := 0; i < options.Warmup; i++ {
		first, second := scope, "rg"
		firstArgs, secondArgs := scopeArgs, rgArgs
		if i%2 == 1 {
			first, second = second, first
			firstArgs, secondArgs = secondArgs, firstArgs
		}
		if _, err := runSearch(first, firstArgs...); err != nil {
			return CompareReport{}, fmt.Errorf("%s warmup failed: %w", first, err)
		}
		if _, err := runSearch(second, secondArgs...); err != nil {
			return CompareReport{}, fmt.Errorf("%s warmup failed: %w", second, err)
		}
	}

	scopeResult := CompareResult{
		Tool:        "Scope",
		Version:     scopeVersion,
		Samples:     make([]time.Duration, 0, options.Runs),
		MatchCounts: make([]int, 0, options.Runs),
	}
	rgResult := CompareResult{
		Tool:        "Ripgrep",
		Version:     rgVersion,
		Samples:     make([]time.Duration, 0, options.Runs),
		MatchCounts: make([]int, 0, options.Runs),
	}

	for i := 0; i < options.Runs; i++ {
		first, second := scope, "rg"
		firstArgs, secondArgs := scopeArgs, rgArgs
		if i%2 == 1 {
			first, second = second, first
			firstArgs, secondArgs = secondArgs, firstArgs
		}
		firstResult, err := runSearch(first, firstArgs...)
		if err != nil {
			return CompareReport{}, fmt.Errorf("%s benchmark run failed: %w", first, err)
		}
		secondResult, err := runSearch(second, secondArgs...)
		if err != nil {
			return CompareReport{}, fmt.Errorf("%s benchmark run failed: %w", second, err)
		}

		if first == scope {
			addSample(&scopeResult, firstResult)
			addSample(&rgResult, secondResult)
		} else {
			addSample(&rgResult, firstResult)
			addSample(&scopeResult, secondResult)
		}
	}

	finishResult(&scopeResult)
	finishResult(&rgResult)

	revision, modified := buildRevision()
	cacheCondition := "Uncontrolled cache state: warmup disabled."
	if options.Warmup > 0 {
		cacheCondition = "Best-effort warm OS file cache: both tools run during alternating warmups; this is not a cold-cache measurement."
	}

	return CompareReport{
		GeneratedAt:       time.Now().UTC(),
		Pattern:           options.Pattern,
		Path:              options.Path,
		WorkingDirectory:  workingDirectory,
		Runs:              options.Runs,
		Warmup:            options.Warmup,
		Workers:           options.Workers,
		Globs:             append([]string(nil), options.Globs...),
		CacheCondition:    cacheCondition,
		GoVersion:         runtime.Version(),
		OS:                runtime.GOOS,
		Architecture:      runtime.GOARCH,
		LogicalCPUs:       runtime.NumCPU(),
		CPUModel:          "not collected by the Go runtime",
		RepositoryCommit:  revision,
		RepositoryDirty:   modified,
		RepositoryBytes:   repoBytes,
		RepositoryFiles:   repoFiles,
		MatchCountsEqual:  countsEqual(scopeResult.MatchCounts, rgResult.MatchCounts),
		MatchCountsStable: countsStable(scopeResult.MatchCounts) && countsStable(rgResult.MatchCounts),
		Scope:             scopeResult,
		Ripgrep:           rgResult,
	}, nil
}

func addSample(result *CompareResult, sample commandResult) {
	result.Samples = append(result.Samples, sample.Duration)
	result.MatchCounts = append(result.MatchCounts, sample.MatchCount)
}

func finishResult(result *CompareResult) {
	if len(result.MatchCounts) > 0 {
		result.MatchCount = result.MatchCounts[len(result.MatchCounts)-1]
	}
	result.Mean = mean(result.Samples)
	sorted := append([]time.Duration(nil), result.Samples...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	if len(sorted) > 0 {
		result.TrimmedMean = trimmedMean(sorted, 0.10)
		result.P50 = percentile(sorted, 50)
		result.P95 = percentile(sorted, 95)
		result.P99 = percentile(sorted, 99)
		result.Min = sorted[0]
		result.Max = sorted[len(sorted)-1]
	}
	result.StdDev = standardDeviation(result.Samples, result.Mean)
}

func runSearch(name string, args ...string) (commandResult, error) {
	start := time.Now()
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	duration := time.Since(start)

	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return commandResult{}, err
		}
		if name != "rg" || exitErr.ExitCode() != 1 {
			return commandResult{}, fmt.Errorf("exited with status %d: %s", exitErr.ExitCode(), strings.TrimSpace(string(exitErr.Stderr)))
		}
	}

	var count int
	if name == "rg" {
		var err error
		count, err = countRipgrepMatches(out)
		if err != nil {
			return commandResult{}, err
		}
	} else {
		var err error
		count, err = countScopeMatches(out)
		if err != nil {
			return commandResult{}, err
		}
	}
	return commandResult{Duration: duration, MatchCount: count}, nil
}

func countScopeMatches(data []byte) (int, error) {
	var matches []json.RawMessage
	if err := json.Unmarshal(data, &matches); err != nil {
		return 0, fmt.Errorf("parsing scope JSON results: %w", err)
	}
	return len(matches), nil
}

func countRipgrepMatches(data []byte) (int, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	count := 0
	for {
		var event rgEvent
		err := decoder.Decode(&event)
		if errors.Is(err, io.EOF) {
			return count, nil
		}
		if err != nil {
			return 0, fmt.Errorf("parsing ripgrep JSON results: %w", err)
		}
		if event.Type == "match" {
			count++
		}
	}
}

func commandVersion(name, arg string) (string, error) {
	out, err := exec.Command(name, arg).Output()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if line == "" {
		return "", fmt.Errorf("%s returned an empty version", name)
	}
	return line, nil
}

func repositorySize(root string) (int64, int64, error) {
	var size, files int64
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			size += info.Size()
			files++
		}
		return nil
	})
	if err != nil {
		return 0, 0, fmt.Errorf("measuring repository %q: %w", root, err)
	}
	return size, files, nil
}

func buildRevision() (string, string) {
	revision := os.Getenv("GITHUB_SHA")
	modified := "unknown"
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				revision = setting.Value
			case "vcs.modified":
				modified = setting.Value
			}
		}
	}
	if revision == "" {
		revision = "unknown"
	}
	return revision, modified
}

func countsEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func countsStable(counts []int) bool {
	if len(counts) < 2 {
		return true
	}
	for _, count := range counts[1:] {
		if count != counts[0] {
			return false
		}
	}
	return true
}

func mean(samples []time.Duration) time.Duration {
	if len(samples) == 0 {
		return 0
	}
	var total time.Duration
	for _, sample := range samples {
		total += sample
	}
	return total / time.Duration(len(samples))
}

func percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	index := int(math.Ceil(float64(p)/100*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func standardDeviation(samples []time.Duration, average time.Duration) time.Duration {
	if len(samples) < 2 {
		return 0
	}
	var variance float64
	for _, sample := range samples {
		difference := float64(sample - average)
		variance += difference * difference
	}
	variance /= float64(len(samples) - 1)
	return time.Duration(math.Sqrt(variance))
}

func trimmedMean(sorted []time.Duration, fraction float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	drop := int(math.Round(float64(len(sorted)) * fraction))
	if drop*2 >= len(sorted) {
		return percentile(sorted, 50)
	}
	return mean(sorted[drop : len(sorted)-drop])
}

func mannWhitneyPValue(a, b []time.Duration) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 1
	}
	var u float64
	for _, left := range a {
		for _, right := range b {
			switch {
			case left > right:
				u++
			case left == right:
				u += 0.5
			}
		}
	}
	product := float64(len(a) * len(b))
	mu := product / 2
	sigma := math.Sqrt(product * float64(len(a)+len(b)+1) / 12)
	if sigma == 0 {
		return 1
	}
	u = math.Min(u, product-u)
	z := (u - mu) / sigma
	return math.Erfc(math.Abs(z) / math.Sqrt2)
}

func SaveCompareReport(path string, report CompareReport) error {
	if path == "" {
		return fmt.Errorf("benchmark report path must not be empty")
	}
	artifact := compareArtifact{
		SchemaVersion:    1,
		GeneratedAt:      report.GeneratedAt,
		Pattern:          report.Pattern,
		Path:             report.Path,
		WorkingDirectory: report.WorkingDirectory,
		Runs:             report.Runs,
		Warmup:           report.Warmup,
		Workers:          report.Workers,
		Globs:            report.Globs,
		CacheCondition:   report.CacheCondition,
		Environment: compareEnvironment{
			GoVersion:        report.GoVersion,
			OS:               report.OS,
			Architecture:     report.Architecture,
			LogicalCPUs:      report.LogicalCPUs,
			CPUModel:         report.CPUModel,
			RepositoryCommit: report.RepositoryCommit,
			RepositoryDirty:  report.RepositoryDirty,
		},
		Repository: compareRepository{
			SizeBytes:        report.RepositoryBytes,
			FileCount:        report.RepositoryFiles,
			MeasurementScope: "all regular files below the path, including files ignored by either search tool",
		},
		SearchRules: compareSearchRules{
			Scope:       "Scope built-in ignored directories/extensions and .scope-ignore rules",
			Ripgrep:     "ripgrep default ignore files, gitignore rules, hidden-file behavior, and binary handling",
			Limitations: "Glob and regular-expression behavior can differ between tools, and their default ignore rules are not equivalent.",
		},
		MatchCountsEqual:  report.MatchCountsEqual,
		MatchCountsStable: report.MatchCountsStable,
		Scope:             artifactResult(report.Scope),
		Ripgrep:           artifactResult(report.Ripgrep),
	}
	data, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding benchmark report: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating report directory: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("writing benchmark report: %w", err)
	}
	return nil
}

func artifactResult(result CompareResult) compareArtifactResult {
	samples := make([]float64, len(result.Samples))
	for i, sample := range result.Samples {
		samples[i] = milliseconds(sample)
	}
	return compareArtifactResult{
		Version:       result.Version,
		MeanMS:        milliseconds(result.Mean),
		TrimmedMeanMS: milliseconds(result.TrimmedMean),
		P50MS:         milliseconds(result.P50),
		P95MS:         milliseconds(result.P95),
		P99MS:         milliseconds(result.P99),
		MinMS:         milliseconds(result.Min),
		MaxMS:         milliseconds(result.Max),
		StdDevMS:      milliseconds(result.StdDev),
		MatchCount:    result.MatchCount,
		MatchCounts:   result.MatchCounts,
		SamplesMS:     samples,
	}
}

func milliseconds(duration time.Duration) float64 {
	return float64(duration) / float64(time.Millisecond)
}

func PrintCompare(report CompareReport) {
	output.PrintHeader("Benchmark", "Scope vs Ripgrep")
	output.PrintKeyValue("Pattern", report.Pattern)
	output.PrintKeyValue("Path", report.Path)
	output.PrintKeyValue("Working directory", report.WorkingDirectory)
	output.PrintKeyValue("Workers", strconv.Itoa(report.Workers))
	output.PrintKeyValue("Timed runs", strconv.Itoa(report.Runs))
	output.PrintKeyValue("Warmup runs", strconv.Itoa(report.Warmup))
	output.PrintKeyValue("Cache", report.CacheCondition)
	output.PrintKeyValue("Environment", fmt.Sprintf("%s/%s, %d logical CPUs", report.OS, report.Architecture, report.LogicalCPUs))
	output.PrintKeyValue("Go version", report.GoVersion)
	output.PrintKeyValue("Repository revision", report.RepositoryCommit)
	output.PrintKeyValue("Repository state", report.RepositoryDirty)
	output.PrintKeyValue("Repository size", fmt.Sprintf("%s across %d files (including ignored files)", humanBytes(report.RepositoryBytes), report.RepositoryFiles))
	output.PrintKeyValue("Scope", report.Scope.Version)
	output.PrintKeyValue("Ripgrep", report.Ripgrep.Version)
	if len(report.Globs) > 0 {
		output.PrintKeyValue("Globs", strings.Join(report.Globs, ", "))
	}
	fmt.Println()

	output.PrintSection("Search policy and result counts")
	output.PrintKeyValue("Scope ignores", "built-in directories/extensions and .scope-ignore")
	output.PrintKeyValue("Ripgrep ignores", "ripgrep defaults, including gitignore rules and hidden files")
	output.PrintKeyValue("Pattern behavior", "regular-expression and glob syntax can differ between tools")
	output.PrintKeyValue("Ignore behavior", "default ignore rules are not equivalent")
	output.PrintKeyValue("Match count parity", fmt.Sprintf("scope %d, ripgrep %d", report.Scope.MatchCount, report.Ripgrep.MatchCount))
	if !report.MatchCountsEqual || !report.MatchCountsStable {
		output.WarningColor.Println("Match counts differed or changed between runs; timing results are not a like-for-like comparison.")
	}
	fmt.Println()

	output.PrintSection("End-to-end CLI latency (ms)")
	output.PrintTable(
		[]string{"Tool", "Mean", "Trimmed", "P50", "P95", "P99", "Min", "Max", "StdDev"},
		[][]string{
			{report.Scope.Tool, fmtMs(report.Scope.Mean), fmtMs(report.Scope.TrimmedMean), fmtMs(report.Scope.P50), fmtMs(report.Scope.P95), fmtMs(report.Scope.P99), fmtMs(report.Scope.Min), fmtMs(report.Scope.Max), fmtMs(report.Scope.StdDev)},
			{report.Ripgrep.Tool, fmtMs(report.Ripgrep.Mean), fmtMs(report.Ripgrep.TrimmedMean), fmtMs(report.Ripgrep.P50), fmtMs(report.Ripgrep.P95), fmtMs(report.Ripgrep.P99), fmtMs(report.Ripgrep.Min), fmtMs(report.Ripgrep.Max), fmtMs(report.Ripgrep.StdDev)},
		},
		[]string{"left", "right", "right", "right", "right", "right", "right", "right", "right"},
	)
	fmt.Println()
	output.PrintKeyValue("Approx. Mann-Whitney p-value", fmt.Sprintf("%.4f (exploratory; not a CI gate)", mannWhitneyPValue(report.Scope.Samples, report.Ripgrep.Samples)))
	fmt.Println()
	output.WarningColor.Println("Exploratory measurements only. Cache state, background load, startup overhead, and tool-specific search rules affect results; no CI performance gate is applied.")
}

func fmtMs(duration time.Duration) string {
	return fmt.Sprintf("%.3f", milliseconds(duration))
}

func humanBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
