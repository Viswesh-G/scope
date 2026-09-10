# Scope CLI — Commands Reference

> **Legend:** flags marked `(req)` are required. All others are optional with the shown default.

---

## Search

### `scope search`
Search files for a regex pattern. The core command.

**Flags:**
| Flag | Short | Default | Description |
|---|---|---|---|
| `--pattern` | `-p` | *(req)* | Regex pattern to search for |
| `--path` | | `.` | Directory to search in (use `-` for stdin) |
| `--recursive` | `-r` | `true` | Search subdirectories |
| `--ignore-case` | `-i` | `false` | Case-insensitive search |
| `--workers` | `-w` | #CPUs | Number of parallel goroutines |
| `--fname` | `-f` | `false` | Match filenames instead of file contents |
| `--hotspots` | | `false` | Rank files by match count (no per-line output) |
| `--count` | | `false` | Print only the total match count |
| `--quiet` | `-q` | `false` | Suppress the metrics table (just show matches) |
| `--max-results` | `-m` | `0` (unlimited) | Stop showing matches after N results |
| `--output` | `-o` | *(stdout)* | Write matches to a file instead of stdout |
| `--html` | | `""` | Write a beautiful standalone HTML report |
| `--json` | | `false` | Emit matches as a JSON array (file, line, content) |
| `--after-context` | `-A` | `0` | Print N lines of context after each match |
| `--before-context` | `-B` | `0` | Print N lines of context before each match |
| `--context` | `-C` | `0` | Print N lines of context on both sides (sets -A and -B) |
| `--glob` | `-g` | *(none)* | File glob pattern to include/exclude (repeatable, e.g. `*.go`, `!*_test.go`) |
| `--profile` | | `false` | Write CPU + memory profiles to `.scope/` |

**Examples:**
```bash
# Basic search
scope search -p "TODO"

# Case-insensitive in a specific directory
scope search -p "func main" --path ./cmd -i

# Find which files have the most matches
scope search -p "github" --hotspots

# Just print the count, nothing else
scope search -p "TODO" --count

# Stop after the first 10 matches
scope search -p "error" -m 10

# Save matches to a file, skip the metrics table
scope search -p "TODO" -o matches.txt -q

# Combine: search + profile + save output
scope search -p "func" --profile -o func_matches.txt -q

# Generate a beautiful HTML report with visual charts
scope search -p "github" --html report.html

# Filename search (find files named like the pattern)
scope search -p "engine" -f

# Show 2 lines of context around every match (like grep -C 2)
scope search -p "TODO" -C 2

# Asymmetric context: 3 lines before, 1 line after
scope search -p "func" -B 3 -A 1

# Only search Go files, skip test files
scope search -p "TODO" -g "*.go" -g "!*_test.go"

# Emit results as a JSON array (pipe-friendly)
scope search -p "error" --json

# Search stdin (pipe input directly)
cat server.log | scope search -p "ERROR" --path -
```

---

## Profiling

### `scope search --profile`
Wraps a search run with Go's built-in `runtime/pprof`. Generates two files in `.scope/`:

| File | Contains |
|---|---|
| `.scope/cpu.pprof` | Where CPU time was spent (stack samples every 10ms) |
| `.scope/mem.pprof` | Heap allocations at the end of the search |
| `.scope/cpu.svg` | Scalable Vector Graphic flamegraph for CPU |
| `.scope/mem.svg` | Scalable Vector Graphic flamegraph for Memory |
| `.scope/flamegraphs.md` | Embedded SVG flamegraphs for easy viewing |

```bash
scope search -p "func" --profile

# Then inspect:
go tool pprof .scope/cpu.pprof             # interactive CLI
go tool pprof -http=:8080 .scope/cpu.pprof # browser flamegraph
go tool pprof .scope/mem.pprof             # heap allocations
```

> **Tip:** the browser flamegraph (`-http=:8080`) is the most useful — it shows you exactly which functions ate the most CPU time and lets you click around.

---

## Benchmarking

### `scope compare`
Benchmark scope vs ripgrep head-to-head. Alternates which tool goes first each round to cancel out OS cache effects.

**Flags:**
| Flag | Default | Description |
|---|---|---|
| `--pattern` / `-p` | *(req)* | Pattern to search for |
| `--path` | `.` | Directory to search |
| `--runs` | `20` | Number of timed rounds |
| `--warmup` | `3` | Throwaway warmup rounds (not counted) |

```bash
scope compare -p "github" --runs 20
scope compare -p "TODO" --path ./internal --runs 50 --warmup 5
```

**Output includes:** mean, trimmed mean (10%), P50/P95/P99, standard deviation, throughput (MB/s), a sparkline run-by-run chart, and a Mann-Whitney U significance test.

> Requires `rg` (ripgrep) on your PATH.

---

## Codebase Audit

### `scope audit` *(chains stats + deps + dupes)*
Run a full codebase health check in one command. Saves running three commands separately.

**Flags:**
| Flag | Default | Description |
|---|---|---|
| `--path` | `.` | Directory to audit |
| `--skip-dupes` | `false` | Skip the (slower) duplicate file scan |

```bash
scope audit
scope audit --path ./internal
scope audit --skip-dupes     # faster, skips SHA256 dedup scan
```

**Output includes:**
- File extension breakdown (how many `.go`, `.md`, etc.)
- Go import counts (which packages you depend on most)
- Duplicate files by SHA256 content hash

---

## Codebase Analysis (individual commands)

### `scope stats`
File extension breakdown — how many files of each type exist.

```bash
scope stats
scope stats --path ./src
```

### `scope deps`
Go import counts — which packages appear across the codebase most often.

```bash
scope deps
scope deps --path ./internal
```

### `scope dupes`
Find files with identical contents (SHA256 hash comparison).

**Flags:** `--path` (`.`), `--workers`/`-w` (#CPUs)

```bash
scope dupes
scope dupes --path ./assets -w 4
```

### `scope graph`
Print the directory tree. Optionally export as Graphviz DOT.

**Flags:** `--path` (`.`), `--dot` (false)

```bash
scope graph
scope graph --path ./internal
scope graph --dot > graph.dot
dot -Tsvg graph.dot > graph.svg   # render with graphviz
```

---

## Search History

Every `scope search` is recorded to `.scope/history.json` (capped at 1000 entries).

### `scope history`
Show all recorded searches in a table (timestamp, pattern, matches, workers, duration).

```bash
scope history
```

### `scope history stats`
Aggregated metrics: total searches, unique patterns, total matches, avg/fastest/slowest duration.

```bash
scope history stats
```

### `scope history top`
Most frequently searched patterns, ranked by count.

```bash
scope history top
```

### `scope history slowest`
Top 10 slowest searches you've run.

```bash
scope history slowest
```

### `scope history fastest`
Top 10 fastest searches you've run.

```bash
scope history fastest
```

### `scope history recent`
Most recent N searches (default: 10).

**Flags:** `--limit`/`-n` (default `10`)

```bash
scope history recent
scope history recent --limit 25
```

### `scope history pattern <text>`
Filter history to entries whose pattern contains `<text>`.

```bash
scope history pattern "TODO"
scope history pattern "func"
```

### `scope history path <path>`
Filter history to entries that searched in a specific path.

```bash
scope history path "./cmd"
scope history path "./internal"
```

### `scope history replay`
Re-run the most recent (or Nth most recent) search from history.
Great for quickly repeating a past search without retyping.

**Flags:** `--nth` (default `1` = most recent)

```bash
scope history replay          # repeat the last search
scope history replay --nth 3  # repeat the 3rd most recent
```

> The replayed search is saved to history as a new entry.

### `scope history export`
Export all history to a JSON file.

**Flags:** `--output`/`-o` (default `history_export.json`)

```bash
scope history export
scope history export -o my_backup.json
```

### `scope history prune <N>`
Keep only the newest N records (trim the rest).

```bash
scope history prune 50
```

### `scope history clear`
Delete all search history.

```bash
scope history clear
```

---

## Configuration

Settings saved to `~/.scope-config.yaml` via Viper. Persist across all sessions.

### `scope config set-color <element> <color>`
Change the color of any output element.

**Elements:** `title`, `section`, `success`, `warning`, `error`, `dim`, `file`, `match`, `time`
**Colors:** `black`, `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`, `faint`, `bold`, `hired`, `higreen`, `hiyellow`, `hiblue`, `himagenta`, `hicyan`, `hiwhite`

```bash
scope config set-color match red
scope config set-color title hicyan
scope config set-color file hiblue
```

### `scope config list`
Show all elements, their current color, and a live color preview.

```bash
scope config list
```

### `scope config reset`
Reset all colors back to defaults.

```bash
scope config reset
```

---

## Ignore Rules

`.scope-ignore` works like `.gitignore` — one pattern per line, `#` for comments, wildcards supported.

### `scope ignore init`
Create a `.scope-ignore` file with default patterns (`.git`, `node_modules`, `vendor`, `build`, `dist`).

```bash
scope ignore init
```

### `scope ignore add <pattern>`
Add a pattern to `.scope-ignore`. Supports globs.

```bash
scope ignore add "*.log"
scope ignore add "tmp/"
scope ignore add "**/*_test.go"
```

### `scope ignore remove <pattern>`
Remove a pattern from `.scope-ignore`.

```bash
scope ignore remove "*.log"
```

### `scope ignore list`
Show all patterns in the local `.scope-ignore`.

```bash
scope ignore list
```

---

## Chaining Examples

Scope's flags are designed so common workflows collapse into single commands:

```bash
# Profile a search and save matches to file (no terminal clutter)
scope search -p "func" --profile -o funcs.txt -q

# Quick audit before committing
scope audit --skip-dupes

# Replay your last search to check results haven't changed
scope history replay

# Find where TODO comments are most concentrated
scope search -p "TODO" --hotspots

# Export history after a long dev session
scope history export -o session_backup.json

# Count matches in a subdirectory only
scope search -p "error" --path ./internal --count

# Benchmark after a performance change
scope compare -p "func" --runs 30 --warmup 5

# Open a flamegraph after profiling (requires Go toolchain)
scope search -p "github" --profile
# You can now easily view .scope/flamegraphs.md directly in your IDE!
# Or continue using pprof:
go tool pprof -http=:8080 .scope/cpu.pprof

# Show surrounding context around every error (like grep -C 3)
scope search -p "error" -C 3

# Grep only Go source files (skip tests and vendor)
scope search -p "TODO" -g "*.go" -g "!*_test.go" -g "!vendor/*"

# Pipe a log file into scp and filter live
cat app.log | scope search -p "FATAL" --path -

# Machine-readable output for scripts / CI
scope search -p "FIXME" --json -q | jq '.[].file' | sort -u
```
