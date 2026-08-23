// Package profiler is a thin wrapper around Go's built-in pprof tools.
// When --profile is passed to a search, we start collecting CPU samples.
// When the search finishes, we write both .pprof files and try to generate SVGs.
//
// Output files land in .scope/ so they stay out of the way:
//   .scope/cpu.pprof  ← raw CPU profile
//   .scope/mem.pprof  ← raw memory profile
//   .scope/cpu.svg    ← flamegraph SVG (only if Graphviz is installed)
//   .scope/mem.svg    ← memory graph SVG (only if Graphviz is installed)
//   .scope/flamegraphs.md ← markdown viewer (embeds SVGs, with fallback text)
package profiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/pprof"
)

// Session holds the open CPU profile file so we can close it when done.
// Created by Start(), cleaned up by Stop().
type Session struct {
	cpuFile *os.File // the open file we're writing CPU samples into
	dir     string   // the output directory (e.g. ".scope")
}

// Start begins CPU profiling. Returns a Session you must call Stop() on later.
func Start(dir string) (*Session, error) {
	// Make sure the output folder exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("profiler: couldn't create output dir %q: %w", dir, err)
	}

	f, err := os.Create(filepath.Join(dir, "cpu.pprof"))
	if err != nil {
		return nil, fmt.Errorf("profiler: couldn't create cpu.pprof: %w", err)
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		return nil, fmt.Errorf("profiler: couldn't start CPU profiling: %w", err)
	}

	return &Session{cpuFile: f, dir: dir}, nil
}

// Stop finishes profiling, writes the memory profile, and tries to generate SVGs.
// Safe to call on a nil session (does nothing), so you can always defer Stop().
func (s *Session) Stop() {
	if s == nil {
		return
	}

	// 1. Finish CPU profile and close the file
	pprof.StopCPUProfile()
	s.cpuFile.Close()

	// 2. Write the memory (heap) profile
	memPath := filepath.Join(s.dir, "mem.pprof")
	mf, err := os.Create(memPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "profiler: couldn't create mem.pprof: %v\n", err)
		return
	}
	defer mf.Close()

	runtime.GC() // run GC first so we get a clean snapshot of live objects
	if err := pprof.WriteHeapProfile(mf); err != nil {
		fmt.Fprintf(os.Stderr, "profiler: couldn't write heap profile: %v\n", err)
	}

	// 3. Try to generate SVG flamegraphs using the Go toolchain
	cpuSvg := filepath.Join(s.dir, "cpu.svg")
	memSvg := filepath.Join(s.dir, "mem.svg")
	goExe := findGoExecutable()

	cpuOk := generateSVG(goExe, cpuSvg, s.cpuFile.Name())
	memOk := generateSVG(goExe, memSvg, memPath)

	// 4. Write the markdown file (works whether or not SVGs were generated)
	writeFlamegraphsMD(s.dir, cpuOk, memOk)
}

// generateSVG calls "go tool pprof -svg" to convert a .pprof file into an SVG.
// Returns true if it worked, false if something went wrong (e.g. Graphviz not installed).
func generateSVG(goExe, outputSvg, inputPprof string) bool {
	if goExe == "" {
		return false // no Go toolchain found, skip silently
	}

	cmd := exec.Command(goExe, "tool", "pprof", "-svg", "-output", outputSvg, inputPprof)
	
	// Winget often installs Graphviz but forgets to add it to the Windows PATH.
	// To make this "just work" for the user, we'll manually inject common Graphviz locations!
	if runtime.GOOS == "windows" {
		graphvizPaths := `;C:\Program Files\Graphviz\bin;C:\Program Files (x86)\Graphviz\bin`
		cmd.Env = append(os.Environ(), "PATH="+os.Getenv("PATH")+graphvizPaths)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		// This usually means Graphviz 'dot' is not installed — it's optional, not a bug
		fmt.Fprintf(os.Stderr, "profiler: SVG skipped (%v)\n  (install Graphviz to enable flamegraphs: https://graphviz.org/download/)\n  Output: %s\n", err, string(out))
		return false
	}
	return true
}

// writeFlamegraphsMD writes .scope/flamegraphs.md.
// If SVGs exist, it embeds them as images. If not, it shows instructions instead.
func writeFlamegraphsMD(dir string, cpuOk, memOk bool) {
	cpuSection := svgSection("CPU Profile", "cpu.svg", cpuOk,
		"go tool pprof -http=:8080 .scope/cpu.pprof")

	memSection := svgSection("Memory Profile", "mem.svg", memOk,
		"go tool pprof -http=:8080 .scope/mem.pprof")

	content := "# Scope Flamegraphs\n\n" +
		"These show exactly where time and memory went during your search.\n\n" +
		cpuSection + "\n\n" + memSection + "\n"

	mdPath := filepath.Join(dir, "flamegraphs.md")
	if err := os.WriteFile(mdPath, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "profiler: couldn't write flamegraphs.md: %v\n", err)
	}
}

// svgSection builds one section of the markdown file.
// If svgOk is true, we embed the SVG image. If not, we show a fallback command.
func svgSection(title, svgFile string, svgOk bool, fallbackCmd string) string {
	header := "## " + title + "\n\n"
	if svgOk {
		// Standard markdown image syntax — works in VS Code, GitHub, etc.
		return header + "![" + title + "](" + svgFile + ")"
	}
	// Fallback: tell the user how to view the profile manually
	return header +
		"*SVG not generated — Graphviz is not installed.*\n\n" +
		"To view this profile as an interactive flamegraph, run:\n\n" +
		"```bash\n" + fallbackCmd + "\n```"
}

// findGoExecutable tries to find the 'go' binary so we can call "go tool pprof".
// We look in GOROOT first (most reliable), then fall back to PATH.
func findGoExecutable() string {
	// GOROOT/bin/go is the most reliable way to find the right Go version
	goroot := runtime.GOROOT()
	if goroot != "" {
		candidate := filepath.Join(goroot, "bin", "go")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		// On Windows, try with .exe extension
		candidate += ".exe"
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// Fallback: search the system PATH
	if path, err := exec.LookPath("go"); err == nil {
		return path
	}

	return "" // couldn't find it
}

// PrintSummary prints where the profile files landed and how to view them.
func PrintSummary(dir string) {
	fmt.Println()
	fmt.Printf("  CPU profile  → %s\n", filepath.Join(dir, "cpu.pprof"))
	fmt.Printf("  Mem profile  → %s\n", filepath.Join(dir, "mem.pprof"))
	fmt.Printf("  Flamegraphs  → %s\n", filepath.Join(dir, "flamegraphs.md"))
	fmt.Println()
	fmt.Printf("  View online: go tool pprof -http=:8080 %s\n", filepath.Join(dir, "cpu.pprof"))
	fmt.Println()
}
