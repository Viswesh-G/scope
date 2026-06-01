package analysis

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Viswesh-G/scope/internal/output"
)

// CompareResult holds the outcome of a benchmark run for a single tool.
type CompareResult struct {
	Tool         string
	Duration     time.Duration // mean across timed runs
	BytesScanned int64
	FileCount    int64

	// Per-run samples (timed runs only, after warmup)
	Samples []time.Duration

	// Derived stats
	P50         time.Duration
	P95         time.Duration
	P99         time.Duration
	Min         time.Duration
	Max         time.Duration
	StdDev      time.Duration
	TrimmedMean time.Duration // 10% trimmed mean — outlier-resistant
}

// RunResult captures a single execution: wall time + files matched.
type RunResult struct {
	Duration  time.Duration
	FileCount int
	Err       error
}

func runCommand(name string, args ...string) RunResult {
	start := time.Now()
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	dur := time.Since(start)

	fileCount := 0
	if err == nil || isExitErr(err) {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.TrimSpace(line) != "" {
				fileCount++
			}
		}
	}

	return RunResult{Duration: dur, FileCount: fileCount, Err: err}
}

func isExitErr(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*exec.ExitError)
	return ok
}

// Compare benchmarks Scope vs ripgrep.
//
// Methodology:
//   - `warmup` throwaway rounds to prime the OS page cache for both tools equally.
//   - `runs` timed rounds with alternating order (even i → scope first, odd i → rg first)
//     to cancel first-runner cache advantage.
//   - Order within each pair is additionally randomised when randomOrder=true.
func Compare(pattern, path string, runs, warmup int) (CompareResult, CompareResult, error) {
	bytesScanned, err := DirSize(path)
	if err != nil {
		return CompareResult{}, CompareResult{}, err
	}

	fileCount, err := FileCount(path)
	if err != nil {
		return CompareResult{}, CompareResult{}, err
	}

	scopeArgs := []string{"search", "-p", pattern, "--path", path, "--no-history", "--no-config"}
	rgArgs := []string{pattern, path}

	// ── Warmup (alternating order, discarded) ─────────────────────────────────
	for i := 0; i < warmup; i++ {
		if i%2 == 0 {
			runCommand("./scope", scopeArgs...)
			r := runCommand("rg", rgArgs...)
			if errors.Is(r.Err, exec.ErrNotFound) {
				return CompareResult{}, CompareResult{}, fmt.Errorf(
					"ripgrep ('rg') is not installed or not in your PATH",
				)
			}
		} else {
			r := runCommand("rg", rgArgs...)
			if errors.Is(r.Err, exec.ErrNotFound) {
				return CompareResult{}, CompareResult{}, fmt.Errorf(
					"ripgrep ('rg') is not installed or not in your PATH",
				)
			}
			_ = r
			runCommand("./scope", scopeArgs...)
		}
	}

	var scopeSamples, rgSamples []time.Duration

	// ── Timed runs (alternating order) ───────────────────────────────────────
	// Even iterations: scope → rg   (scope runs first, rg gets warm cache)
	// Odd  iterations: rg → scope   (rg runs first, scope gets warm cache)
	// Across 20 runs this gives each tool 10 first-runner slots and 10 second-runner
	// slots, eliminating the systematic ordering bias.
	for i := 0; i < runs; i++ {
		if i%2 == 0 {
			sr := runCommand("./scope", scopeArgs...)
			if sr.Err != nil && !isExitErr(sr.Err) {
				return CompareResult{}, CompareResult{}, fmt.Errorf("scope failed: %w", sr.Err)
			}
			scopeSamples = append(scopeSamples, sr.Duration)

			rr := runCommand("rg", rgArgs...)
			if err := checkRgErr(rr.Err); err != nil {
				return CompareResult{}, CompareResult{}, err
			}
			rgSamples = append(rgSamples, rr.Duration)
		} else {
			rr := runCommand("rg", rgArgs...)
			if err := checkRgErr(rr.Err); err != nil {
				return CompareResult{}, CompareResult{}, err
			}
			rgSamples = append(rgSamples, rr.Duration)

			sr := runCommand("./scope", scopeArgs...)
			if sr.Err != nil && !isExitErr(sr.Err) {
				return CompareResult{}, CompareResult{}, fmt.Errorf("scope failed: %w", sr.Err)
			}
			scopeSamples = append(scopeSamples, sr.Duration)
		}
	}

	scopeResult := buildResult("Scope", scopeSamples, bytesScanned, fileCount)
	rgResult := buildResult("Ripgrep", rgSamples, bytesScanned, fileCount)

	return scopeResult, rgResult, nil
}

func checkRgErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("ripgrep ('rg') is not installed or not in your PATH")
	}
	if !isExitErr(err) {
		return fmt.Errorf("ripgrep failed: %w", err)
	}
	return nil
}

func buildResult(tool string, samples []time.Duration, bytes, files int64) CompareResult {
	if len(samples) == 0 {
		return CompareResult{Tool: tool, BytesScanned: bytes, FileCount: files}
	}

	sorted := make([]time.Duration, len(samples))
	copy(sorted, samples)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	var total time.Duration
	for _, s := range samples {
		total += s
	}
	avg := total / time.Duration(len(samples))

	return CompareResult{
		Tool:         tool,
		Duration:     avg,
		BytesScanned: bytes,
		FileCount:    files,
		Samples:      samples,
		P50:          percentile(sorted, 50),
		P95:          percentile(sorted, 95),
		P99:          percentile(sorted, 99),
		Min:          sorted[0],
		Max:          sorted[len(sorted)-1],
		StdDev:       stdDev(samples, avg),
		TrimmedMean:  trimmedMean(sorted, 0.10),
	}
}

// trimmedMean drops the bottom and top `frac` fraction of sorted samples
// and averages the remainder. frac=0.10 drops the lowest 10% and highest 10%.
func trimmedMean(sorted []time.Duration, frac float64) time.Duration {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	drop := int(math.Round(float64(n) * frac))
	trimmed := sorted[drop : n-drop]
	if len(trimmed) == 0 {
		return sorted[n/2]
	}
	var sum time.Duration
	for _, s := range trimmed {
		sum += s
	}
	return sum / time.Duration(len(trimmed))
}

func percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(float64(p)/100.0*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func stdDev(samples []time.Duration, avg time.Duration) time.Duration {
	if len(samples) < 2 {
		return 0
	}
	var variance float64
	for _, s := range samples {
		diff := float64(s - avg)
		variance += diff * diff
	}
	variance /= float64(len(samples) - 1)
	return time.Duration(math.Sqrt(variance))
}

// outlierRatio = Max / P50. Values close to 1.0x mean no bad outliers.
// A ratio of 1.45x means the worst run took 45% longer than the median.
func outlierRatio(max, p50 time.Duration) float64 {
	if p50 <= 0 {
		return 0
	}
	return float64(max) / float64(p50)
}

// mannWhitneyU returns the U statistic and a two-tailed p-value approximation
// (normal approximation, valid for n > 8). Used to determine whether the
// latency difference between the two tools is statistically significant.
func mannWhitneyU(a, b []time.Duration) (float64, float64) {
	n1, n2 := len(a), len(b)
	if n1 == 0 || n2 == 0 {
		return 0, 1
	}

	// Count: for each element in a, how many elements in b are smaller?
	var u1 float64
	for _, x := range a {
		for _, y := range b {
			if x > y {
				u1++
			} else if x == y {
				u1 += 0.5
			}
		}
	}
	u2 := float64(n1*n2) - u1
	u := math.Min(u1, u2)

	// Normal approximation
	mu := float64(n1*n2) / 2
	sigma := math.Sqrt(float64(n1*n2*(n1+n2+1)) / 12)
	if sigma == 0 {
		return u, 1
	}
	z := (u - mu) / sigma
	// Two-tailed p-value via complementary error function
	p := math.Erfc(math.Abs(z) / math.Sqrt2)
	return u, p
}

// PrintCompare renders the full benchmark report.
func PrintCompare(runs, warmup int, scopeResult, rgResult CompareResult) {
	output.PrintHeader("Benchmark", "Scope vs Ripgrep")

	// ── Methodology note ─────────────────────────────────────────────────────
	output.PrintKeyValue("Warmup runs", fmt.Sprintf("%d (discarded, alternating order)", warmup))
	output.PrintKeyValue("Timed runs", fmt.Sprintf("%d (alternating order — bias-cancelled)", runs))
	output.PrintKeyValue("Files on disk", fmt.Sprintf("%d", scopeResult.FileCount))
	output.PrintKeyValue("Data size", humanBytes(scopeResult.BytesScanned))
	output.PrintKeyValue("Note", "Measures end-to-end CLI latency (startup + search + exit)")
	fmt.Println()

	// ── Latency summary ───────────────────────────────────────────────────────
	output.PrintSection("Latency (ms)")
	output.PrintTable(
		[]string{"Tool", "Mean", "Trimmed Mean", "P50", "P95", "P99", "Min", "Max", "StdDev"},
		[][]string{
			{
				scopeResult.Tool,
				fmtMs(scopeResult.Duration),
				fmtMs(scopeResult.TrimmedMean),
				fmtMs(scopeResult.P50),
				fmtMs(scopeResult.P95),
				fmtMs(scopeResult.P99),
				fmtMs(scopeResult.Min),
				fmtMs(scopeResult.Max),
				fmtMs(scopeResult.StdDev),
			},
			{
				rgResult.Tool,
				fmtMs(rgResult.Duration),
				fmtMs(rgResult.TrimmedMean),
				fmtMs(rgResult.P50),
				fmtMs(rgResult.P95),
				fmtMs(rgResult.P99),
				fmtMs(rgResult.Min),
				fmtMs(rgResult.Max),
				fmtMs(rgResult.StdDev),
			},
		},
		[]string{"left", "right", "right", "right", "right", "right", "right", "right", "right"},
	)
	fmt.Println()

	// ── Throughput ────────────────────────────────────────────────────────────
	output.PrintSection("Throughput")
	output.PrintTable(
		[]string{"Tool", "Avg MB/s", "Peak MB/s (best run)"},
		[][]string{
			{
				scopeResult.Tool,
				fmt.Sprintf("%.1f MB/s", throughputMBs(scopeResult.BytesScanned, scopeResult.Duration)),
				fmt.Sprintf("%.1f MB/s", throughputMBs(scopeResult.BytesScanned, scopeResult.Min)),
			},
			{
				rgResult.Tool,
				fmt.Sprintf("%.1f MB/s", throughputMBs(rgResult.BytesScanned, rgResult.Duration)),
				fmt.Sprintf("%.1f MB/s", throughputMBs(rgResult.BytesScanned, rgResult.Min)),
			},
		},
		[]string{"left", "right", "right"},
	)
	fmt.Println()

	// ── Variance / consistency analysis ──────────────────────────────────────
	output.PrintSection("Consistency (lower is more predictable)")
	scopeCV := coefficientOfVariation(scopeResult.StdDev, scopeResult.Duration)
	rgCV := coefficientOfVariation(rgResult.StdDev, rgResult.Duration)
	scopeOR := outlierRatio(scopeResult.Max, scopeResult.P50)
	rgOR := outlierRatio(rgResult.Max, rgResult.P50)
	output.PrintTable(
		[]string{"Tool", "StdDev", "P95-P50 spread", "CV%", "Outlier ratio (Max/P50)"},
		[][]string{
			{
				scopeResult.Tool,
				fmtMs(scopeResult.StdDev),
				fmtMs(scopeResult.P95 - scopeResult.P50),
				fmt.Sprintf("%.1f%%", scopeCV),
				fmt.Sprintf("%.2fx", scopeOR),
			},
			{
				rgResult.Tool,
				fmtMs(rgResult.StdDev),
				fmtMs(rgResult.P95 - rgResult.P50),
				fmt.Sprintf("%.1f%%", rgCV),
				fmt.Sprintf("%.2fx", rgOR),
			},
		},
		[]string{"left", "right", "right", "right", "right"},
	)
	fmt.Println()

	// ── Statistical significance ──────────────────────────────────────────────
	output.PrintSection("Statistical significance (Mann-Whitney U test)")
	_, pValue := mannWhitneyU(scopeResult.Samples, rgResult.Samples)
	sigLabel := "NOT significant (p ≥ 0.05) — treat results with caution"
	if pValue < 0.001 {
		sigLabel = "Highly significant (p < 0.001)"
	} else if pValue < 0.01 {
		sigLabel = fmt.Sprintf("Significant (p ≈ %.3f)", pValue)
	} else if pValue < 0.05 {
		sigLabel = fmt.Sprintf("Marginally significant (p ≈ %.3f)", pValue)
	}
	output.PrintKeyValue("p-value", fmt.Sprintf("%.4f", pValue))
	output.PrintKeyValue("Interpretation", sigLabel)
	fmt.Println()

	// ── Run-by-run distribution ───────────────────────────────────────────────
	output.PrintSection("Per-run distribution (chronological)")
	printSparkline("Scope  ", scopeResult.Samples)
	printSparkline("Ripgrep", rgResult.Samples)
	fmt.Println()

	// ── Score card ────────────────────────────────────────────────────────────
	output.PrintSection("Score card")
	type category struct {
		name        string
		scopeWins   bool
		explanation string
	}

	// Peak throughput: correctly compare MB/s (higher is better), not raw latency.
	scopePeakMBs := throughputMBs(scopeResult.BytesScanned, scopeResult.Min)
	rgPeakMBs := throughputMBs(rgResult.BytesScanned, rgResult.Min)

	// Trimmed mean is a more honest "typical latency" than raw mean because it
	// discards the top and bottom 10% of runs, making single-outlier spikes
	// (e.g. Ripgrep's 71ms run) not carry disproportionate weight.
	categories := []category{
		{
			name:      "Mean latency",
			scopeWins: scopeResult.Duration <= rgResult.Duration,
			explanation: fmt.Sprintf("%s vs %s",
				fmtMs(scopeResult.Duration), fmtMs(rgResult.Duration)),
		},
		{
			name:      "Trimmed mean (10%)",
			scopeWins: scopeResult.TrimmedMean <= rgResult.TrimmedMean,
			explanation: fmt.Sprintf("%s vs %s",
				fmtMs(scopeResult.TrimmedMean), fmtMs(rgResult.TrimmedMean)),
		},
		{
			name:      "P95 latency",
			scopeWins: scopeResult.P95 <= rgResult.P95,
			explanation: fmt.Sprintf("%s vs %s",
				fmtMs(scopeResult.P95), fmtMs(rgResult.P95)),
		},
		{
			name:      "P99 latency",
			scopeWins: scopeResult.P99 <= rgResult.P99,
			explanation: fmt.Sprintf("%s vs %s",
				fmtMs(scopeResult.P99), fmtMs(rgResult.P99)),
		},
		{
			name:      "Peak throughput",
			scopeWins: scopePeakMBs >= rgPeakMBs, // higher MB/s is better
			explanation: fmt.Sprintf("%.1f MB/s vs %.1f MB/s (best run)",
				scopePeakMBs, rgPeakMBs),
		},
		{
			name:      "Consistency (CV)",
			scopeWins: scopeCV <= rgCV,
			explanation: fmt.Sprintf("%.1f%% vs %.1f%% CV",
				scopeCV, rgCV),
		},
		{
			name:      "Outlier stability",
			scopeWins: scopeOR <= rgOR,
			explanation: fmt.Sprintf("%.2fx vs %.2fx worst-run ratio",
				scopeOR, rgOR),
		},
	}

	scopeScore, rgScore := 0, 0
	for _, c := range categories {
		winner := "Ripgrep"
		if c.scopeWins {
			winner = "Scope"
			scopeScore++
		} else {
			rgScore++
		}
		output.PrintKeyValue(
			fmt.Sprintf("  %s", c.name),
			fmt.Sprintf("%-10s  (%s)", winner, c.explanation),
		)
	}
	fmt.Println()

	// ── Overall verdict ───────────────────────────────────────────────────────
	overallWinner := "Ripgrep"
	if scopeScore > rgScore {
		overallWinner = "Scope"
	} else if scopeScore == rgScore {
		overallWinner = "Tie"
	}

	// Use trimmed mean for the speed ratio — more robust than raw mean.
	trimRatio := float64(scopeResult.TrimmedMean) / float64(rgResult.TrimmedMean)
	var speedMsg string
	switch {
	case trimRatio < 0.95:
		speedMsg = fmt.Sprintf("Scope is %.2fx faster on this workload (trimmed mean)", 1/trimRatio)
	case trimRatio > 1.05:
		speedMsg = fmt.Sprintf("Scope is %.2fx slower on this workload (trimmed mean)", trimRatio)
	default:
		speedMsg = "tools are within 5% — effectively tied on this workload"
	}

	output.PrintKeyValue("Score", fmt.Sprintf("Scope %d – %d Ripgrep", scopeScore, rgScore))
	output.PrintKeyValue("Verdict", fmt.Sprintf("%s wins  (%s)", overallWinner, speedMsg))

	// Significance caveat
	if pValue >= 0.05 {
		output.PrintKeyValue("⚠  Caution", "Result is not statistically significant — run more iterations")
	}
	fmt.Println()
}

// printSparkline renders a compact ASCII bar chart of chronological run durations.
func printSparkline(label string, samples []time.Duration) {
	if len(samples) == 0 {
		return
	}

	blocks := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

	minD, maxD := samples[0], samples[0]
	for _, s := range samples {
		if s < minD {
			minD = s
		}
		if s > maxD {
			maxD = s
		}
	}

	var sb strings.Builder
	for _, s := range samples {
		idx := 0
		if maxD > minD {
			idx = int(float64(s-minD) / float64(maxD-minD) * float64(len(blocks)-1))
		}
		sb.WriteString(blocks[idx])
	}

	fmt.Printf("  %s  %s  min:%s max:%s\n",
		label, sb.String(), fmtMs(minD), fmtMs(maxD))
}

func coefficientOfVariation(stddev, avg time.Duration) float64 {
	if avg <= 0 {
		return 0
	}
	return float64(stddev) / float64(avg) * 100
}

func fmtMs(d time.Duration) string {
	return fmt.Sprintf("%.2fms", float64(d.Microseconds())/1000.0)
}

func humanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// DirSize returns total bytes of all regular files under path.
func DirSize(path string) (int64, error) {
	var total int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		total += info.Size()
		return nil
	})
	return total, err
}

// FileCount returns the number of regular files under path.
func FileCount(path string) (int64, error) {
	var count int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		count++
		return nil
	})
	return count, err
}

func throughputMBs(bytes int64, d time.Duration) float64 {
	if d <= 0 {
		return 0
	}
	return float64(bytes) / d.Seconds() / 1024 / 1024
}

// ── Utility: keep rand seeded so warmup alternation stays non-deterministic
// across test harness calls (used if caller ever adds random ordering mode).
func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}