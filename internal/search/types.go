// Package search is the core of the whole project.
//
// The basic idea: one goroutine walks the filesystem and feeds file paths
// into a channel. N worker goroutines read from that channel and scan each
// file for the pattern. Matches go into another channel that gets drained
// and printed.
//
// The tricky part is doing all this concurrently without races, and tracking
// per-worker stats at the same time (that's the "self-profiling" bit).
package search

// SearchMode lets workers know whether to scan file contents or just filenames.
type SearchMode int

const (
	ContentSearch  SearchMode = iota // normal - search inside files
	FilenameSearch                   // --fname flag - match against the filename
)

// Match is one result: a file + line number + the matching text.
type Match struct {
	Mode    SearchMode
	File    string
	LineNum int // 0 for filename matches
	Line    string
}

// Config is everything the search engine needs to know.
// Built from CLI flags in cmd/search.go and passed to Run().
type Config struct {
	Pattern      string
	Path         string
	Recursive    bool
	Workers      int
	FilenameOnly bool
	Hotspots     bool  // rank files by match count instead of printing each match
	IgnoreCase   bool
	SkipHistory  bool  // set by the benchmark so test runs don't pollute history
}
