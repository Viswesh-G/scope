# Contributing to SCP (Scope)

Thank you for your interest in improving SCP!

## Development Setup

1. Clone the repo
2. Ensure you have Go 1.25+ installed
3. Run `make build` to compile the `scp` binary

## Make Commands

We use a Makefile to simplify local development.

- `make build` - Compiles `scp`
- `make test` - Runs the unit tests
- `make race` - Runs tests with the Go race detector enabled
- `make bench` - Runs the performance microbenchmarks (`-benchmem`)
- `make check` - Runs `vet`, `fmt-check`, and `test` (run this before submitting a PR!)
- `make install` - Installs `scp` to your `$GOPATH/bin`

## Architecture

Scope is divided into two main areas:
- `cmd/`: The CLI layer built with Cobra. It parses arguments and passes them to internal packages.
- `internal/`: The core logic.
  - `search/`: The high-performance concurrent file crawler and regex matcher.
  - `walk/`: Shared, cancellable filesystem traversal for search and analysis.
  - `metrics/`: Atomic performance counters to track worker efficiency.
  - `ast/`: Go structural search using the standard library parser.
  - `serve/`: The HTTP server, SSE broadcaster, and Live Dashboard UI.
  - `tui/`: The Bubbletea interactive terminal user interface.

## Adding a Command

1. Create `cmd/mycmd.go`
2. Define a `cobra.Command`
3. Add it to the root command in `init()`: `rootCmd.AddCommand(myCmd)`
4. Put the actual logic inside `internal/mycmd/` to keep the CLI layer thin.

## Benchmarks

If you touch `internal/search`, run `make bench`. SCP is optimized to skip the regex engine for plain-text searches using `strings.Contains`. If your changes cause a performance regression in the literal fast-path, the microbenchmarks will catch it.

The TUI runs the same `search` command as the CLI and keeps JSON output valid for
pipelines. When adding a TUI control, prefer composing existing CLI flags instead
of creating a second search implementation.
