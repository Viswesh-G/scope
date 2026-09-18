# Scope CLI commands

This is the complete command reference for `scp`. Run `scp <command> --help`
for the flags and examples printed by the installed binary.

## Search

### `scp search`

Search file contents or filenames with a regular expression. Searches are
parallel by default and print a metrics report unless `--quiet` is used.

```bash
scp search -p "TODO"
scp search -p "func main" --path ./cmd --ignore-case
scp search -p "TODO" --context 2
scp search -p "TODO" --glob "*.go" --glob "!*_test.go"
scp search -p "error" --count --quiet
scp search -p "error" --hotspots
scp search -p "error" --max-results 20
scp search -p "error" --json --quiet
scp search -p "func" --html report.html
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `-p, --pattern` | required | Regular-expression pattern |
| `--path` | `.` | Directory, or `-` for standard input |
| `-r, --recursive` | `true` | Include subdirectories |
| `-i, --ignore-case` | `false` | Ignore letter case |
| `-w, --workers` | CPU count | Number of search workers |
| `-f, --fname` | `false` | Match filenames instead of contents |
| `--hotspots` | `false` | Rank files by match count |
| `--count` | `false` | Print only the total match count |
| `-m, --max-results` | `0` | Stop after this many displayed matches |
| `-q, --quiet` | `false` | Suppress the metrics report |
| `-o, --output` | stdout | Write normal matches to a file |
| `--html` | empty | Write a standalone HTML report |
| `--json` | `false` | Emit a JSON array |
| `-A, --after-context` | `0` | Context lines after a match |
| `-B, --before-context` | `0` | Context lines before a match |
| `-C, --context` | `0` | Context lines on both sides |
| `-g, --glob` | none | Include/exclude files; repeatable |
| `--profile` | `false` | Write CPU and memory profiles |
| `--parallel-profile` | `false` | Show worker activity as a timeline |

Examples:

```bash
# Read a log through standard input.
cat app.log | scp search -p "FATAL" --path -

# Use JSON in another tool.
scp search -p "FIXME" --json --quiet | jq '.[].file' | sort -u

# Save matches without the performance table.
scp search -p "TODO" --output matches.txt --quiet
```

`--profile` writes `.scope/cpu.pprof`, `.scope/mem.pprof`, and, when
Graphviz is available, SVG visualizations. Inspect a profile with:

```bash
go tool pprof -http=:8080 .scope/cpu.pprof
```

## Structural Go search

### `scp ast`

Search Go declarations using the Go parser. This does not match comments or
strings that merely contain a declaration-like name.

```bash
scp ast --type func
scp ast --type func --name "handle*"
scp ast --type struct --path ./internal
scp ast --type interface --name "*er"
```

Supported types are `func`, `struct`, `interface`, `var`, `const`, and `type`.

## Repository analysis

### `scp audit`

Run file statistics, Go dependency counts, and duplicate-file detection.

```bash
scp audit
scp audit --path ./internal
scp audit --skip-dupes
```

### `scp stats`

Count files by extension:

```bash
scp stats
scp stats --path ./src
```

### `scp deps`

Count imports in Go source files:

```bash
scp deps
scp deps --path ./internal
```

### `scp dupes`

Find files with identical contents using SHA-256 hashes.

```bash
scp dupes
scp dupes --path ./assets --workers 4
```

### `scp graph`

Print a filtered directory tree or Graphviz DOT output.

```bash
scp graph
scp graph --path ./internal
scp graph --dot > graph.dot
dot -Tsvg graph.dot > graph.svg
```

## History

Normal searches are stored in `.scope/history.json`, capped at 1000 entries.

```bash
scp history
scp history stats
scp history top
scp history fastest
scp history slowest
scp history recent --limit 25
scp history pattern "TODO"
scp history path "./internal"
scp history replay
scp history replay --nth 3
scp history export --output history-backup.json
scp history prune 50
scp history clear
```

## Interactive tools

### `scp tui`

Open the Bubble Tea terminal interface. It supports pattern, path, context,
advanced flags, case sensitivity, match/count/hotspot modes, scrolling, and
JSON input from a previous search.

```bash
scp tui
scp search -p "error" --json --quiet | scp tui
```

Keys:

| Key | Action |
| --- | --- |
| `Tab` | Switch tabs |
| `Up` / `Down` | Move between fields or scroll results |
| `Enter` | Run a search |
| `Ctrl+I` | Toggle case sensitivity |
| `Ctrl+R` | Cycle match, count, and hotspot modes |
| `Esc` / `Ctrl+C` | Exit |

### `scp watch`

Run a search again after relevant file changes. Events are debounced so one
editor save does not trigger many searches.

```bash
scp watch -p "TODO"
scp watch -p "func main" --path ./cmd --ignore-case
```

### `scp serve`

Start the local live dashboard:

```bash
scp serve
scp serve --port 3000
```

Open the printed localhost URL. The dashboard shows history and accepts
interactive searches through the local HTTP API.

### `scp compare`

Benchmark Scope against ripgrep. Requires `rg` on `PATH`.

```bash
scp compare -p "github"
scp compare -p "TODO" --runs 50 --warmup 5
```

The report includes mean, trimmed mean, percentiles, standard deviation,
throughput, a sparkline, and a Mann-Whitney U comparison.

## Configuration

Configuration is stored in `~/.scope-config.yaml`.

```bash
scp config list
scp config set-color match red
scp config reset
```

The configurable output elements are `title`, `section`, `success`, `warning`,
`error`, `dim`, `file`, `match`, and `time`.

## Ignore rules

Scope skips common dependency, build, binary, and generated directories by
default. Add repository-specific patterns to `.scope-ignore`.

```bash
scp ignore init
scp ignore add "*.log"
scp ignore remove "*.log"
scp ignore list
```

Patterns are one per line. Blank lines and lines beginning with `#` are ignored.

## Other commands

```bash
scp version
scp completion bash
scp completion zsh
scp completion powershell
```

## Common workflows

```bash
# Audit a repository before a review.
scp audit --skip-dupes

# Find the most concentrated TODO files.
scp search -p "TODO" --hotspots

# Profile a performance-sensitive search.
scp search -p "func" --profile --parallel-profile

# Re-run the most recent search after editing.
scp history replay
```
