# Scope

A fast, concurrent, self-profiling search engine — built from scratch in Go!

Scope works like `ripgrep` or `grep`: it searches files for regex patterns. But it also **instruments itself** — after every search it shows you exactly where time went: which workers scanned which files, how much data was processed, how evenly the work was distributed, and what the parallelism ratio was.

It can also profile itself with Go's built-in `pprof` (generating easy-to-read SVG flamegraphs), export beautiful HTML reports, run a live dashboard, and benchmark itself head-to-head against ripgrep.

---

## Quick Start & Installation

You can install Scope directly using Go:

```bash
go install github.com/Viswesh-G/scope@latest
```

Try it out immediately:

```bash
# Search the current directory for "TODO"
scope search -p "TODO"

# Start the Live Dashboard to track your search history!
scope serve --port 8080

# Generate a beautiful HTML report
scope search -p "func" --html report.html

# Profile the search and generate SVG flamegraphs automatically!
scope search -p "func" --profile

# Benchmark scope against ripgrep (requires rg on PATH)
scope compare -p "github" --runs 20
```

---

## Commands Overview

| Command | What it does |
|---|---|
| `scope search` | Search files for a regex pattern. Supports `--html` and `--profile` |
| `scope serve` | Starts a Live Dashboard on `localhost:8080` to view search history |
| `scope compare` | Benchmark scope vs ripgrep head-to-head |
| `scope audit` | File stats + Go imports + duplicate files, all in one shot |
| `scope history` | View recent searches and metrics in the terminal |
| `scope stats` / `deps` / `dupes` / `graph` | Deep-dive codebase analysis tools |
| `scope config` / `ignore` | Customise colors and ignore patterns |

*(Run `scope <command> --help` for full flags and examples).*

---

## Live Dashboard & HTML Reports

Scope makes it incredibly easy to visualize what your search engine is doing. 

### The Live Dashboard (`scope serve`)
Run `scope serve` to start a local server at `http://localhost:8080`. 
This gives you a beautifully designed, dark-themed UI that automatically updates every 2 seconds to show you:
- Your total searches and total matches found.
- The fastest and average search durations.
- A live-updating history table of all your searches!

### Standalone HTML Reports
You can export any search to a standalone HTML file:
```bash
scope search -p "TODO" --html report.html
```
Open `report.html` in your browser. It contains all matched lines, along with visual CSS charts showing exactly how much data each worker goroutine processed. It works 100% offline!

---

## Profiling & Flamegraphs

Pass `--profile` to any search to generate Go `pprof` profiles:

```bash
scope search -p "func" --profile
```

Scope will automatically:
1. Generate `.scope/cpu.pprof` and `.scope/mem.pprof`.
2. Run `go tool pprof -svg` in the background.
3. Generate `.scope/flamegraphs.md` which cleanly embeds the SVGs so you can view them directly in your IDE (like VSCode or GitHub)!

---

## Architecture

```text
scope/
├── main.go                   # Entry point
├── cmd/                      # CLI layer (Cobra commands)
│   ├── serve.go              # Live dashboard command
│   ├── search.go             # Core search command
│   └── ...                   # Other commands
└── internal/
    ├── search/               # Core engine (walker + workers + collector)
    ├── serve/                # HTTP server and HTML Dashboard UI
    ├── metrics/              # Atomic performance counters
    ├── output/               # Terminal & HTML rendering 
    ├── profiler/             # pprof wrapper & SVG generation
    └── analysis/             # History, benchmarking, file stats
```

**How search works:**
1. **Walker goroutine** crawls the filesystem and sends paths into a channel.
2. **N worker goroutines** read paths in parallel, scan each file, and push matches.
3. **Collector** drains matches and prints them to the terminal or an HTML file.
4. **Metrics** are captured atomically during the whole process for the final report.

---

## CI/CD Automation

Scope includes a GitHub Actions workflow (`.github/workflows/ci.yml`). Every push or pull request to the `main` branch automatically runs `go build` and `go test` on Ubuntu, ensuring the CLI is always fast and stable.

---