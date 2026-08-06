// Package profiler is a simple wrapper around Go's built-in pprof tools.
// When --profile is passed, we start collecting CPU data at the beginning
// of the search and write both CPU + memory profiles when it finishes.
//
// The files land in .scope/ so they stay out of the way:
//   .scope/cpu.pprof
//   .scope/mem.pprof
//
// To inspect them after a run:
//   go tool pprof .scope/cpu.pprof
//   go tool pprof .scope/mem.pprof
package profiler

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
)

// Session holds the open CPU profile file so we can close it later.
// Created by Start(), cleaned up by Stop().
type Session struct {
	cpuFile *os.File
	dir     string
}

// Start begins CPU profiling and returns a Session.
// Call session.Stop() when the work you want to profile is done.
func Start(dir string) (*Session, error) {
	// make sure .scope/ exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("profiler: couldn't create output dir %q: %w", dir, err)
	}

	cpuPath := filepath.Join(dir, "cpu.pprof")
	f, err := os.Create(cpuPath)
	if err != nil {
		return nil, fmt.Errorf("profiler: couldn't create cpu.pprof: %w", err)
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		return nil, fmt.Errorf("profiler: couldn't start CPU profiling: %w", err)
	}

	return &Session{cpuFile: f, dir: dir}, nil
}

// Stop finishes CPU profiling and writes the memory profile.
// Safe to call with a nil session (does nothing).
func (s *Session) Stop() {
	if s == nil {
		return
	}

	// finish the CPU profile and close the file
	pprof.StopCPUProfile()
	s.cpuFile.Close()

	// write heap profile - GC first so we get a clean snapshot
	memPath := filepath.Join(s.dir, "mem.pprof")
	mf, err := os.Create(memPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "profiler: couldn't write mem.pprof: %v\n", err)
		return
	}
	defer mf.Close()

	runtime.GC()
	if err := pprof.WriteHeapProfile(mf); err != nil {
		fmt.Fprintf(os.Stderr, "profiler: couldn't write heap profile: %v\n", err)
	}
}

// PrintSummary prints where the profile files ended up and how to read them.
func PrintSummary(dir string) {
	fmt.Println()
	fmt.Printf("  CPU profile → %s\n", filepath.Join(dir, "cpu.pprof"))
	fmt.Printf("  Mem profile → %s\n", filepath.Join(dir, "mem.pprof"))
	fmt.Println()
	fmt.Printf("  Inspect with:  go tool pprof %s\n", filepath.Join(dir, "cpu.pprof"))
	fmt.Println()
}
