# Scope CLI commands

This is the complete command reference for `scope`. Run `scope <command> --help`
for the flags and examples printed by the installed binary.

The installed binary is named `scope`. `scp` still works as an alias, so any
script or muscle memory built around the old name keeps working.

## Search

### `scope search`

Search file contents or filenames with a regular expression. Searches are
parallel by default and print a metrics report unless `--quiet` is used.

```bash
scope search -p "TODO"
scope search -p "func main" --path ./cmd --ignore-case
scope search -p "TODO" --context 2
scope search -p "TODO" --glob "*.go" --glob "!*_test.go"
scope search -p "error" --count --quiet
scope search -p "error" --hotspots
scope search -p "error" --max-results 20
scope search -p "error" --json --quiet
scope search -p "func" --html report.html
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
cat app.log | scope search -p "FATAL" --path -

# Use JSON in another tool.
scope search -p "FIXME" --json --quiet | jq '.[].file' | sort -u

# Save matches without the performance table.
scope search -p "TODO" --output matches.txt --quiet
```

`--profile` writes `.scope/cpu.pprof`, `.scope/mem.pprof`, and, when
Graphviz is available, SVG visualizations. Inspect a profile with:

```bash
go tool pprof -http=:8080 .scope/cpu.pprof
```

## Structural Go search

### `scope ast`

Search Go declarations using the Go parser. This does not match comments or
strings that merely contain a declaration-like name.

```bash
scope ast --type func
scope ast --type func --name "handle*"
scope ast --type struct --path ./internal
scope ast --type interface --name "*er"
```

Supported types are `func`, `struct`, `interface`, `var`, `const`, and `type`.

## Repository analysis

### `scope audit`

Run file statistics, Go dependency counts, and duplicate-file detection.

```bash
scope audit
scope audit --path ./internal
scope audit --skip-dupes
```

### `scope stats`

Count files by extension:

```bash
scope stats
scope stats --path ./src
```

### `scope deps`

Count imports in Go source files:

```bash
scope deps
scope deps --path ./internal
```

### `scope dupes`

Find files with identical contents using SHA-256 hashes.

```bash
scope dupes
scope dupes --path ./assets --workers 4
```

### `scope graph`

Print a filtered directory tree or Graphviz DOT output.

```bash
scope graph
scope graph --path ./internal
scope graph --dot > graph.dot
dot -Tsvg graph.dot > graph.svg
```

## History

Normal searches are stored in `.scope/history.json`, capped at 1000 entries.

```bash
scope history
scope history stats
scope history top
scope history fastest
scope history slowest
scope history recent --limit 25
scope history pattern "TODO"
scope history path "./internal"
scope history replay
scope history replay --nth 3
scope history export --output history-backup.json
scope history prune 50
scope history clear
```

## Interactive tools

### `scope tui`

Open the Bubble Tea terminal interface. It supports pattern, path, context,
advanced flags, case sensitivity, match/count/hotspot modes, scrolling, and
JSON input from a previous search.

```bash
scope tui
scope search -p "error" --json --quiet | scope tui
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

### `scope watch`

Run a search again after relevant file changes. Events are debounced so one
editor save does not trigger many searches.

```bash
scope watch -p "TODO"
scope watch -p "func main" --path ./cmd --ignore-case
```

### `scope serve`

Start the local live dashboard:

```bash
scope serve
scope serve --port 3000
```

Open the printed localhost URL. The dashboard shows history and accepts
interactive searches through the local HTTP API.

### `scope compare`

Benchmark Scope against ripgrep. Requires `rg` on `PATH`.

```bash
scope compare -p "github"
scope compare -p "TODO" --runs 50 --warmup 5
```

The report includes mean, trimmed mean, percentiles, standard deviation,
throughput, a sparkline, and a Mann-Whitney U comparison.

## Configuration

Configuration is stored in `~/.scope-config.yaml`.

```bash
scope config list
scope config set-color match red
scope config reset
```

The configurable output elements are `title`, `section`, `success`, `warning`,
`error`, `dim`, `file`, `match`, and `time`.

## Ignore rules

Scope skips common dependency, build, binary, and generated directories by
default. Add repository-specific patterns to `.scope-ignore`.

```bash
scope ignore init
scope ignore add "*.log"
scope ignore remove "*.log"
scope ignore list
```

Patterns are one per line. Blank lines and lines beginning with `#` are ignored.

## Other commands

```bash
scope version
scope completion bash
scope completion zsh
scope completion powershell
```

## Common workflows

```bash
# Audit a repository before a review.
scope audit --skip-dupes

# Find the most concentrated TODO files.
scope search -p "TODO" --hotspots

# Profile a performance-sensitive search.
scope search -p "func" --profile --parallel-profile

# Re-run the most recent search after editing.
scope history replay
```
