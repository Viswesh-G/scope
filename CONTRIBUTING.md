# Contributing to scope

Thank you for your interest in improving scope!

## Development Setup

1. Clone the repo
2. Ensure you have Go 1.25+ installed
3. Run `make build` to compile the `scope` binary (and the `scp` alias)

## Make Commands

We use a Makefile to simplify local development.

- `make build` - Compiles `scope` and the `scp` alias
- `make test` - Runs the unit tests
- `make race` - Runs tests with the Go race detector enabled
- `make bench` - Runs the performance microbenchmarks (`-benchmem`)
- `make cover` - Produces a coverage report
- `make check` - Runs `vet`, `fmt-check`, `vuln`, and `test` (run this before submitting a PR!)
- `make install` - Installs `scope` to your `$GOPATH/bin`

## Code style

The project is meant to be readable by someone learning Go, so:

- keep the `cmd/` layer thin, logic belongs in `internal/`;
- write comments that explain *why* something is happening, not what the line
  obviously does;
- prefer straightforward code over clever abstractions. If a feature can be
  added without a new abstraction layer, do that instead.

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

If you touch `internal/search`, run `make bench`. scope is optimized to skip the regex engine for plain-text searches using `strings.Contains`. If your changes cause a performance regression in the literal fast-path, the microbenchmarks will catch it.

The TUI runs the same `search` command as the CLI and keeps JSON output valid for
pipelines. When adding a TUI control, prefer composing existing CLI flags instead
of creating a second search implementation.
