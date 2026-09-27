# scope

**A local-first codebase intelligence tool that shows you how it searches.**

[![Go Reference](https://pkg.go.dev/badge/github.com/Viswesh-G/scope.svg)](https://pkg.go.dev/github.com/Viswesh-G/scope)
[![CI](https://github.com/Viswesh-G/scope/actions/workflows/ci.yml/badge.svg)](https://github.com/Viswesh-G/scope/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Viswesh-G/scope.svg)](https://github.com/Viswesh-G/scope/releases/latest)
[![License](https://img.shields.io/github/license/Viswesh-G/scope.svg)](LICENSE)
[![Go Report](https://goreportcard.com/badge/github.com/Viswesh-G/scope)](https://goreportcard.com/report/github.com/Viswesh-G/scope)

Most search tools answer one question: *where is this string?*

`scope` answers that, and then keeps going. It tells you how long the search
took, how the work was split across CPU cores, which files are the hotspots,
and what your repository actually looks like structurally. Everything runs on
your machine. Nothing is uploaded, indexed, or sent anywhere.

```bash
go install github.com/Viswesh-G/scope@latest
scope search -p "TODO"
```

---

## Table of contents

- [Why this exists](#why-this-exists)
- [What you actually see](#what-you-actually-see)
- [Install](#install)
- [The four things it does](#the-four-things-it-does)
- [Command overview](#command-overview)
- [Being honest about ripgrep](#being-honest-about-ripgrep)
- [Architecture](#architecture)
- [Development](#development)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [License](#license)

---

## Why this exists

I wanted a search tool I could actually learn concurrency from. Every other
option was a black box: you type a pattern, results come out, and you have no
idea what happened in between.

So `scope` instruments itself. Every search prints a breakdown of the work:
how many directories and files were walked, how many matches turned up, how
much CPU time each worker burned, how evenly the files were distributed, and
whether the parallelism actually paid off. When something feels slow, you can
dump a real pprof profile and look at a flame graph instead of guessing.

It grew from there. Once the tool already understood your files and kept a
history of what you searched for, structural search, repository audits, and a
live dashboard were natural things to add.

---

## What you actually see

A normal search prints the matches, then the telemetry:

```text
--- Legend ---
  File Path : Line Number : Matched Text

Scope Metrics
Dirs Scanned  : 148
Files Scanned : 402
Files Ignored : 1193
Matches Found : 37
Walk Time     : 3.104ms
Search Time   : 12.482ms
Total Time    : 4.219ms
Parallelism   : 2.96x
Concurrency: Good

Worker Stats:
  Worker 0   ████████████████░░░░   78% files scanned
  work: 9.31ms   files: 313   matches: 31   size: 1.2 MB
  throughput: 135.1 KB/s
  Worker 1   ████░░░░░░░░░░░░░░░░   20% files scanned
  work: 3.02ms   files: 82   matches: 6   size: 410.2 KB
  throughput: 138.7 KB/s

Load Balance
────────────────────
Max Worker Load : 313 files  /  1.20 MB
Min Worker Load : 82 files   /  410.20 KB
Average Load    : 197.5 files  /  819.30 KB
Imbalance       : 1.12x
```

`Parallelism` is the interesting number. It is total worker CPU time divided
by wall-clock time, so `2.96x` means you got almost three cores' worth of
throughput out of a 4ms search. If it reads `0.4x`, your workers spent more
time waiting on the filesystem than scanning, and that is worth knowing.

There is also a terminal UI and a browser dashboard if you would rather not
read tables.

---

## Install

Requires Go 1.25 or newer.

```bash
# from source (recommended)
go install github.com/Viswesh-G/scope@latest
```

Or grab a prebuilt binary for Linux, macOS, or Windows from the
[releases page](https://github.com/Viswesh-G/scope/releases/latest).

<details>
<summary>Build from a checkout</summary>

```bash
make build   # produces ./scope and ./scp
make test
make check
```

</details>

### About the command name

The binary is called **`scope`**. An earlier version shipped as `scp`, which
turned out to be a bad idea: it shadows the OpenSSH `scp` command that already
exists on nearly every machine, and it made the project impossible to find by
searching for it.

`scp` still works as an alias, so nothing breaks if you have muscle memory for
it, but new installs get `scope`.

---

## The four things it does

### 1. Find

Literal patterns skip the regex engine entirely and use `strings.Contains`,
which is several times faster. Anything with metacharacters falls back to the
full `regexp` engine.

```bash
scope search -p "TODO"
scope search -p "func main" --path ./cmd -i
scope search -p "error" -C 2                 # context lines
scope search -p "FIXME" -g "*.go" -g "!*_test.go"
cat app.log | scope search -p "FATAL" --path -
scope search -p "error" --json -q | jq '.[].file'
```

Structural search understands Go code instead of matching text in it, so it
does not get confused by a comment that happens to mention a function name:

```bash
scope ast --type func --name "handle*"
scope ast --type struct --path ./internal
scope ast --type interface --name "*er"
```

Supported node types: `func`, `struct`, `interface`, `var`, `const`, `type`.

### 2. Understand

```bash
scope audit            # file stats + dependency counts + duplicate detection
scope stats            # files by extension
scope deps             # Go import frequency
scope dupes            # byte-identical files, by SHA-256
scope graph            # directory tree
scope graph --dot | dot -Tsvg > graph.svg
```

### 3. Measure

```bash
scope search -p "func" --profile              # .scope/cpu.pprof + mem.pprof
scope search -p "func" --parallel-profile    # ASCII worker timeline
scope compare -p "github" --runs 20 --warmup 3
```

`compare` runs both `scope` and `rg` against the same pattern, alternating who
goes first to cancel out cache warming, discarding warmup rounds, and reporting
mean, trimmed mean, percentiles, standard deviation, and a Mann-Whitney U test
so you get an answer instead of a single noisy timing.

### 4. Monitor

```bash
scope watch -p "TODO"      # re-runs the search on every file change
scope tui                  # interactive terminal UI
scope serve                # local web dashboard with live SSE updates
```

Searches are recorded to `.scope/history.json` (capped at 1000 entries):

```bash
scope history              # recent searches
scope history top          # your most-used patterns
scope history slowest      # what has been costing you time
scope history replay       # re-run the last one
scope history pattern "TODO"
```

---

## Command overview

| Command | What it does |
| --- | --- |
| `scope search` | The main event. Regex/literal search with metrics, JSON, HTML, and profile output. |
| `scope ast` | Structural search over Go declarations. |
| `scope audit` | One-shot repository health check. |
| `scope stats` | File counts by extension. |
| `scope deps` | Go import frequency. |
| `scope dupes` | Duplicate file detection via SHA-256. |
| `scope graph` | Directory tree or Graphviz DOT output. |
| `scope watch` | Debounced live re-search on file changes. |
| `scope tui` | Bubble Tea terminal interface. |
| `scope serve` | Local HTTP dashboard with Server-Sent Events. |
| `scope compare` | Statistical benchmark against ripgrep. |
| `scope history` | Search history: stats, top, slowest, replay, export. |
| `scope config` | Colors and output settings. |
| `scope ignore` | Manage `.scope-ignore` patterns. |

Full flag reference lives in [`COMMANDS.md`](COMMANDS.md).

---

## Being honest about ripgrep

`rg` is faster than `scope` on most plain literal searches, and it will stay
faster. It is a decade-old project written in Rust with hand-tuned SIMD
matching, and it is excellent at exactly one thing.

`scope` is a different bet. It is a codebase intelligence tool that happens to
contain a search engine, built so you can see and understand what the tool is
doing, and inspect Go structure rather than raw text. If you want the fastest
possible `grep`, use `rg`. If you want to understand a repository and watch how
the work gets done, this is the tool.

If `scope search` ever loses badly on a workload you care about, run
`scope compare` and the numbers will tell you exactly where.

---

## Architecture

```mermaid
flowchart LR
    Walker -->|file paths| Workers
    Workers -->|matches| Collector
    Workers -->|atomic counters| Registry
    Registry --> Report
    Report --> Terminal
    Report --> HTML
    Report --> History
    History --> Dashboard
```

| Package | Responsibility |
| --- | --- |
| `cmd/` | Cobra commands and flag parsing. Thin, no logic. |
| `internal/search/` | Concurrent search pipeline. |
| `internal/walk/` | Shared, cancellable filesystem traversal. |
| `internal/metrics/` | Atomic counters and worker reports. |
| `internal/analysis/` | History, repository stats, duplicates, graphs, benchmarks. |
| `internal/ast/` | Go structural search. |
| `internal/output/` | Terminal rendering, colors, HTML reports. |
| `internal/tui/` | Bubble Tea interface. |
| `internal/serve/` | Local HTTP API, SSE, embedded dashboard. |
| `internal/watch/` | Debounced filesystem watching. |
| `internal/ignore/` | Built-in and repository ignore rules. |
| `internal/profiler/` | CPU, memory, and flamegraph output. |

The pipeline is deliberately straightforward: one goroutine walks the tree and
sends paths down a channel, N workers scan files and push matches to another
channel, and a collector renders whatever it receives. No thread pool
abstraction, no plugin system. If you want to understand how the concurrency
works, you can read `internal/search/engine.go` top to bottom and actually get
it.

---

## Development

```bash
make build      # build ./scope and ./scp
make test       # unit tests
make race       # tests under the race detector
make bench      # search microbenchmarks with -benchmem
make cover      # coverage report
make vet        # go vet
make check      # everything CI runs
```

Optional tools, used when present: `rg` for `scope compare`, `graphviz` for
flamegraph SVGs, `govulncheck` for the security scan.

CI runs build, vet, formatting checks, golangci-lint, `govulncheck`, coverage,
a coverage threshold, and the race detector on both Linux and Windows.

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for the workflow and conventions.

---

## Roadmap

Things that are planned, roughly in order of how much they'd improve the tool:

- **Context cancellation.** Ctrl+C should stop a search cleanly instead of
  just killing the process.
- **Safer history storage.** The JSON history file is read-modify-written with
  no locking, so concurrent searches, watch mode, and dashboard-triggered
  searches can race. It needs an append-only log or a lock.
- **Symbol indexing.** Build a real index of Go symbols so `scope` can answer
  "where is this used" and "what breaks if I change this".
- **Cross-language structural search.** Put the parser behind an interface so
  Python, TypeScript, and Rust can be added without rewriting the walker.
- **JSON output everywhere.** `ast`, `graph`, `audit`, and `deps` should all be
  scriptable.
- **Performance regression benchmarks in CI.** A benchmark that runs on every
  PR and fails when throughput drops.
- **Harden the dashboard.** Localhost-only by default, explicit opt-in for
  remote binding, request path validation, optional auth token.

---

## Contributing

Issues, ideas, and pull requests are all welcome. If you are thinking about
touching `internal/search`, run `make bench` first and after, so a performance
regression does not slip in unnoticed.

The code is written to be readable by someone learning Go, and the comments
explain *why* rather than restating what the line does. Please keep it that
way. A tool whose selling point is that you can read it should stay that way.

---

## License

MIT. See [`LICENSE`](LICENSE).
