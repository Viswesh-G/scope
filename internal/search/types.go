package search

type Match struct {
	File    string
	LineNum int
	Line    string
}

type Config struct {
	Pattern    string
	Path       string
	Recursive  bool
	IgnoreCase bool
	Workers    int
}

type IgnoreMatcher struct {
	Patterns []string
}
