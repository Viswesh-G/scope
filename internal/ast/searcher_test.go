package ast

// Tests for the AST searcher.
// Each test creates a small Go source file in a temp directory and checks that
// the searcher finds the right declarations.
import (
	"go/parser"
	"go/token"
	"testing"
)

// testSource is a small Go file we can parse in tests without writing to disk.
const testSource = `
package example

import "fmt"

type Animal struct {
	Name string
	Age  int
}

type Speaker interface {
	Speak() string
}

func greetAnimal(a Animal) string {
	return fmt.Sprintf("Hello, %s!", a.Name)
}

func (a Animal) Speak() string {
	return "I am " + a.Name
}

var DefaultAnimal = Animal{Name: "Cat"}

const MaxAnimals = 100
`

func parseTestSource(t *testing.T) (*token.FileSet, interface{}) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", testSource, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse test source: %v", err)
	}
	return fset, file
}

func TestMatchFunc_NoFilter(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", testSource, 0)
	if err != nil {
		t.Fatal(err)
	}

	cfg := Config{NodeType: "func", Name: "", Path: "."}
	results := searchFile(fset, "test.go", file, cfg)

	// testSource has two functions: greetAnimal and Animal.Speak
	if len(results) != 2 {
		t.Errorf("expected 2 func results, got %d", len(results))
	}
}

func TestMatchFunc_WithGlob(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", testSource, 0)
	if err != nil {
		t.Fatal(err)
	}

	cfg := Config{NodeType: "func", Name: "greet*", Path: "."}
	results := searchFile(fset, "test.go", file, cfg)

	if len(results) != 1 {
		t.Errorf("expected 1 func result matching 'greet*', got %d", len(results))
	}
	if results[0].Name != "greetAnimal" {
		t.Errorf("expected func name 'greetAnimal', got %q", results[0].Name)
	}
}

func TestMatchStruct(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", testSource, 0)
	if err != nil {
		t.Fatal(err)
	}

	cfg := Config{NodeType: "struct", Name: "", Path: "."}
	results := searchFile(fset, "test.go", file, cfg)

	if len(results) != 1 {
		t.Errorf("expected 1 struct result, got %d", len(results))
	}
	if results[0].Name != "Animal" {
		t.Errorf("expected struct 'Animal', got %q", results[0].Name)
	}
}

func TestMatchInterface(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", testSource, 0)
	if err != nil {
		t.Fatal(err)
	}

	cfg := Config{NodeType: "interface", Name: "*er", Path: "."}
	results := searchFile(fset, "test.go", file, cfg)

	if len(results) != 1 {
		t.Errorf("expected 1 interface result matching '*er', got %d", len(results))
	}
	if results[0].Name != "Speaker" {
		t.Errorf("expected interface 'Speaker', got %q", results[0].Name)
	}
}

func TestMatchName_EmptyPatternMatchesAll(t *testing.T) {
	if !matchName("AnythingAtAll", "") {
		t.Error("empty pattern should match all names")
	}
}

func TestMatchName_GlobStar(t *testing.T) {
	if !matchName("handleCreate", "handle*") {
		t.Error("'handle*' should match 'handleCreate'")
	}
	if matchName("createHandle", "handle*") {
		t.Error("'handle*' should NOT match 'createHandle'")
	}
}

func TestMatchName_CaseInsensitive(t *testing.T) {
	if !matchName("HandleCreate", "handle*") {
		t.Error("matching should be case-insensitive")
	}
}

func TestValidateNodeType(t *testing.T) {
	valid := []string{"func", "struct", "interface", "var", "const", "type"}
	for _, v := range valid {
		if err := validateNodeType(v); err != nil {
			t.Errorf("expected %q to be valid, got: %v", v, err)
		}
	}

	if err := validateNodeType("banana"); err == nil {
		t.Error("expected error for unknown node type 'banana'")
	}
}
