package search
type SearchMode int

const (
	ContentSearch SearchMode = iota
	FilenameSearch
)


type Match struct {
	Mode SearchMode

	File    string
	LineNum int
	Line    string
}


type Config struct {
	Pattern    string
	FilenameOnly   bool
	Hotspots   bool
	Path       string
	Recursive  bool
	IgnoreCase bool
	Workers    int
	SkipHistory bool
}


