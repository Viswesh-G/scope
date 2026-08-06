# Scope

A fast, concurrent, self-profiling search engine — built from scratch in Go.

Scope works like `ripgrep` or `grep`: it searches files for regex patterns. But it also **instruments itself** — after every search it shows you exactly where time went: which workers scanned which files, how much data was processed, how evenly the work was distributed, and what the parallelism ratio was.

It can also profile itself with Go's built-in `pprof` and benchmark itself head-to-head against ripgrep with real statistics (mean, P50, P95, P99, Mann-Whitney U test, and a sparkline chart).

---

## Quick Start

```bash
# Search the current directory for "TODO"
scope search -p "TODO"

# Case-insensitive search in a specific directory
scope search -p "func main" --path ./cmd -i

# Find which files have the most matches
scope search -p "github" --hotspots

# Profile the search and open the CPU profile in a browser flamegraph
scope search -p "func" --profile
go tool pprof -http=:8080 .scope/cpu.pprof

# Benchmark scope against ripgrep (requires rg on PATH)
scope compare -p "github" --runs 20

# Full codebase health check: file stats + Go imports + duplicate files
scope audit

# View recent searches
scope history
```

---

## Commands

| Command | What it does |
|---|---|
| `scope search` | Search files for a regex pattern |
| `scope compare` | Benchmark scope vs ripgrep |
| `scope audit` | File stats + Go imports + duplicate files, all in one shot |
| `scope history` | View recent searches |
| `scope history stats` | Aggregate metrics over all history |
| `scope history top` | Most frequently searched patterns |
| `scope history slowest` | 10 slowest searches |
| `scope history fastest` | 10 fastest searches |
| `scope history recent` | N most recent searches |
| `scope history pattern <text>` | Filter history by pattern text |
| `scope history path <path>` | Filter history by search path |
| `scope history replay` | Re-run a past search from history |
| `scope history export` | Export history to a JSON file |
| `scope history prune <N>` | Keep only the newest N records |
| `scope history clear` | Delete all history |
| `scope stats` | File extension breakdown |
| `scope deps` | Go import counts |
| `scope dupes` | Find duplicate files (SHA256) |
| `scope graph` | Print the directory tree (or export as Graphviz DOT) |
| `scope config set-color` | Customise output colors |
| `scope config list` | Show current colors |
| `scope config reset` | Reset colors to defaults |
| `scope ignore init` | Create a `.scope-ignore` file |
| `scope ignore add` | Add a pattern to `.scope-ignore` |
| `scope ignore remove` | Remove a pattern from `.scope-ignore` |
| `scope ignore list` | Show all patterns in `.scope-ignore` |

Run `scope <command> --help` for flags and examples.

---

## Architecture

```
scope/
├── main.go                   # Entry point (just calls cmd.Execute)
├── cmd/                      # CLI layer — thin Cobra commands, no real logic here
│   ├── root.go               # Root command + Execute() + PersistentPreRunE (loads config)
│   ├── search.go             # scope search
│   ├── compare.go            # scope compare
│   ├── audit.go              # scope audit (chains stats + deps + dupes)
│   ├── history.go            # scope history (+ 10 subcommands)
│   ├── config.go             # scope config
│   ├── ignore.go             # scope ignore
│   ├── stats.go              # scope stats
│   ├── deps.go               # scope deps
│   ├── dupes.go              # scope dupes
│   └── graph.go              # scope graph
└── internal/
    ├── search/               # Core search engine (walker + workers + collector)
    ├── metrics/              # Atomic performance counters + report builder
    ├── output/               # Terminal rendering (colors, tables, reports)
    ├── config/               # User config (~/.scope-config.yaml via Viper)
    ├── ignore/               # .scope-ignore parsing + built-in defaults
    ├── profiler/             # pprof wrapper — writes .scope/cpu.pprof + mem.pprof
    └── analysis/             # History, benchmarking, stats, deps, dupes, graph
```

### How search works

1. **Walker goroutine** crawls the filesystem and sends file paths into a buffered channel.
2. **N worker goroutines** read from that channel in parallel, scan each file line-by-line, and push matches into a second channel.
3. **Collector** drains the match channel and prints matches, or ranks them by count for `--hotspots`.
4. After the search, all atomic counters (bytes scanned, files ignored, work time per worker, etc.) are snapshotted into a `metrics.Report` and rendered in a table.

**Literal fast-path:** When the search pattern contains no regex metacharacters (e.g. `"TODO"`, not `"TO\w+"`), scope uses `strings.Contains` instead of the regex engine — 5–10× faster for the most common case.

**Parallelism ratio:** The metrics table shows `SearchDuration / TotalDuration`. If 8 workers each spent 200ms scanning and the whole search took 250ms wall time, that's a ~6.4× parallelism ratio — meaning you got 6.4× more throughput than a single goroutine would give you.

---

## Profiling (already working!)

Pass `--profile` to any search to generate Go pprof profiles:

```bash
scope search -p "func" --profile

# Then inspect:
go tool pprof .scope/cpu.pprof          # interactive CLI
go tool pprof -http=:8080 .scope/cpu.pprof  # browser flamegraph
go tool pprof .scope/mem.pprof          # heap allocations
```

Two files are written to `.scope/`:

| File | What's in it |
|---|---|
| `.scope/cpu.pprof` | CPU samples taken every 10ms during the search |
| `.scope/mem.pprof` | Heap allocations snapshot at the end |

---

## Benchmarking

```bash
scope compare -p "github" --runs 20
scope compare -p "TODO" --path ./internal --runs 50 --warmup 5
```

The benchmark alternates which tool runs first each round to cancel out OS file-cache warming effects. Output includes: mean, trimmed mean (10%), P50/P95/P99, standard deviation, throughput (MB/s), a sparkline run-by-run chart, and a Mann-Whitney U significance test.

> Requires `rg` (ripgrep) on your PATH.

---

## Ignore Rules

Scope automatically skips:
- `.git`, `node_modules`, `vendor`, `build`, `dist`, and other common noise (built-in)
- Binary file extensions (`.exe`, `.so`, `.png`, etc.)
- Anything listed in `.scope-ignore` in the directory you're searching

To create a `.scope-ignore`:
```bash
scope ignore init       # creates with defaults
scope ignore add "*.log"
scope ignore add "tmp/"
```

---

## Customisation

Change any output color:
```bash
scope config set-color match red
scope config set-color title hicyan
scope config list        # preview all options
scope config reset       # restore defaults
```

Colors are saved to `~/.scope-config.yaml`. They persist across all sessions.

---

## Roadmap

Done:
- [x] `scope search --profile` → `.scope/cpu.pprof` + `.scope/mem.pprof`
- [x] `scope compare` → head-to-head benchmark with full statistics

Still to do:
- [ ] `scope search --html report.html` → HTML metrics report (open in browser)
- [ ] `scope serve` → live dashboard at `localhost:8080`
- [ ] `scope flame latest` → open flamegraph in browser directly from scope
