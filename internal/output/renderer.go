// Package output handles everything printed to the terminal.
//
// Four files:
//
//	renderer.go   - the Renderer interface
//	base_color.go - global color vars, InitColors(), MapColor()
//	ui.go         - reusable helpers (headers, tables, errors, etc.)
//	console.go    - ConsoleRenderer: prints the post-search metrics report
package output

import "github.com/Viswesh-G/scope/internal/metrics"

// Renderer is an interface so we can swap out how reports are displayed.
// Right now there's only ConsoleRenderer, but this makes it easy to add
// an HTMLRenderer or JSONRenderer later without touching the search engine.
type Renderer interface {
	Render(metrics.Report)
}
