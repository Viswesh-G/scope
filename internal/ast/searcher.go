// Package ast implements "scp ast": structural search for Go source code.
//
// The big idea: instead of searching raw text with a regex, we parse the Go source
// code into an Abstract Syntax Tree (AST) and search the tree's nodes directly.
//
// This means you can find "all functions named handle*" without accidentally
// matching a comment that says "// handle this case" or a variable called handleFn.
// The search is precise because it knows what is and isn't a function declaration.
//
// We use Go's own standard library for this: "go/parser" to parse the file,
// and "go/ast" to walk the resulting tree. No external tools, no CGO.
package ast

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/Viswesh-G/scope/internal/output"
)

// Config holds the search settings for an AST search run.
type Config struct {
	NodeType string // "func", "struct", "interface", "var", "const", "type"
	Name     string // glob pattern for the declaration name, e.g. "handle*" or "*er"
	Path     string // directory to search
}

// Result is one declaration that matched the search.
type Result struct {
	File     string // path to the .go file
	Line     int    // line number of the declaration
	NodeType string // what kind of node this is (e.g. "func")
	Name     string // the identifier name of the declaration
	Sig      string // a short signature or description (e.g. "(x int) string")
}

// Run walks the directory at cfg.Path and searches every .go file for
// declarations that match the type and name filters.
func Run(cfg Config) error {
	if err := validateNodeType(cfg.NodeType); err != nil {
		return err
	}

	// File set tracks position info (file name + line number) for all parsed files.
	// It's shared across all files so we can ask "what line is this node on?"
	fset := token.NewFileSet()

	var results []Result
	var parseErrors []string

	// Walk the directory looking for .go files to parse
	err := filepath.WalkDir(cfg.Path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			// Skip common non-Go directories to keep things fast
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		// Parse this Go file into an AST.
		// parser.ParseComments includes comment nodes so the tree is complete,
		// but we don't actually use comments for matching — they're just there
		// in case a future feature needs them.
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			// Don't abort the whole search if one file has a syntax error.
			// Real codebases sometimes have WIP files that don't compile.
			parseErrors = append(parseErrors, fmt.Sprintf("%s: %v", path, err))
			return nil
		}

		// Walk the file's AST and collect matching declarations
		fileResults := searchFile(fset, path, file, cfg)
		results = append(results, fileResults...)
		return nil
	})

	if err != nil {
		return fmt.Errorf("walking directory %q: %w", cfg.Path, err)
	}

	// Print parse errors as warnings, but don't fail
	for _, e := range parseErrors {
		output.PrintWarning(fmt.Sprintf("parse error: %s", e))
	}

	if len(results) == 0 {
		output.PrintSuccess(fmt.Sprintf("No %s declarations found matching %q.", cfg.NodeType, cfg.Name))
		return nil
	}

	printResults(results, cfg)
	return nil
}

// searchFile walks a single parsed Go file and returns all matching declarations.
func searchFile(fset *token.FileSet, path string, file *ast.File, cfg Config) []Result {
	var results []Result

	// ast.Inspect is a depth-first tree walker. The function we pass gets called
	// for every node in the tree. Returning true means "keep going into children".
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return false
		}

		var result *Result

		switch cfg.NodeType {
		case "func":
			result = matchFunc(fset, path, n, cfg.Name)
		case "struct":
			result = matchStruct(fset, path, n, cfg.Name)
		case "interface":
			result = matchInterface(fset, path, n, cfg.Name)
		case "var":
			result = matchVar(fset, path, n, cfg.Name)
		case "const":
			result = matchConst(fset, path, n, cfg.Name)
		case "type":
			result = matchType(fset, path, n, cfg.Name)
		}

		if result != nil {
			results = append(results, *result)
		}

		return true // always visit children
	})

	return results
}

// -- Individual node matchers --

// matchFunc checks if a node is a function declaration whose name matches the pattern.
// Function declarations look like: func myFunc(...) { ... }
// Method declarations look like: func (r *Receiver) myMethod(...) { ... }
func matchFunc(fset *token.FileSet, path string, n ast.Node, namePattern string) *Result {
	fn, ok := n.(*ast.FuncDecl)
	if !ok || fn.Name == nil {
		return nil
	}

	if !matchName(fn.Name.Name, namePattern) {
		return nil
	}

	// Build a short signature string to display
	sig := buildFuncSig(fn)
	pos := fset.Position(fn.Pos())

	return &Result{
		File:     path,
		Line:     pos.Line,
		NodeType: "func",
		Name:     fn.Name.Name,
		Sig:      sig,
	}
}

// matchStruct checks if a node is a type declaration wrapping a struct type.
// Struct declarations look like: type MyStruct struct { ... }
func matchStruct(fset *token.FileSet, path string, n ast.Node, namePattern string) *Result {
	ts, ok := n.(*ast.TypeSpec)
	if !ok || ts.Name == nil {
		return nil
	}
	if _, isStruct := ts.Type.(*ast.StructType); !isStruct {
		return nil
	}
	if !matchName(ts.Name.Name, namePattern) {
		return nil
	}

	pos := fset.Position(ts.Pos())
	return &Result{
		File:     path,
		Line:     pos.Line,
		NodeType: "struct",
		Name:     ts.Name.Name,
	}
}

// matchInterface checks if a node is a type declaration wrapping an interface type.
func matchInterface(fset *token.FileSet, path string, n ast.Node, namePattern string) *Result {
	ts, ok := n.(*ast.TypeSpec)
	if !ok || ts.Name == nil {
		return nil
	}
	if _, isIface := ts.Type.(*ast.InterfaceType); !isIface {
		return nil
	}
	if !matchName(ts.Name.Name, namePattern) {
		return nil
	}

	pos := fset.Position(ts.Pos())
	return &Result{
		File:     path,
		Line:     pos.Line,
		NodeType: "interface",
		Name:     ts.Name.Name,
	}
}

// matchVar checks for top-level variable declarations: var x = ...
func matchVar(fset *token.FileSet, path string, n ast.Node, namePattern string) *Result {
	vs, ok := n.(*ast.ValueSpec)
	if !ok {
		return nil
	}
	// Check if this ValueSpec is inside a var block (not const)
	// We do this by checking the parent GenDecl, but since ast.Inspect doesn't
	// give us the parent, we check all names and report the first match.
	for _, ident := range vs.Names {
		if ident == nil {
			continue
		}
		if !matchName(ident.Name, namePattern) {
			continue
		}
		pos := fset.Position(ident.Pos())
		return &Result{
			File:     path,
			Line:     pos.Line,
			NodeType: "var/const",
			Name:     ident.Name,
		}
	}
	return nil
}

// matchConst is the same as matchVar since they share the same AST node type.
// We distinguish them at the GenDecl level, but that's not easily accessible from Inspect.
func matchConst(fset *token.FileSet, path string, n ast.Node, namePattern string) *Result {
	return matchVar(fset, path, n, namePattern)
}

// matchType finds any type declaration (including ones that aren't struct or interface).
func matchType(fset *token.FileSet, path string, n ast.Node, namePattern string) *Result {
	ts, ok := n.(*ast.TypeSpec)
	if !ok || ts.Name == nil {
		return nil
	}
	if !matchName(ts.Name.Name, namePattern) {
		return nil
	}
	pos := fset.Position(ts.Pos())
	return &Result{
		File:     path,
		Line:     pos.Line,
		NodeType: "type",
		Name:     ts.Name.Name,
	}
}

// -- Helpers --

// matchName returns true if name matches the glob pattern.
// An empty pattern matches everything (show all declarations of that type).
// We use filepath.Match for glob matching: * matches any sequence of non-separator chars.
func matchName(name, pattern string) bool {
	if pattern == "" {
		return true // no filter = match everything
	}
	matched, err := filepath.Match(strings.ToLower(pattern), strings.ToLower(name))
	if err != nil {
		return strings.EqualFold(name, pattern) // if pattern is invalid, fall back to exact match
	}
	return matched
}

// buildFuncSig creates a short, human-readable signature for a function declaration.
// For a method: "(recv *T) funcName(params) results"
// For a function: "funcName(params) results"
func buildFuncSig(fn *ast.FuncDecl) string {
	var parts []string

	// receiver (for methods)
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		recv := fn.Recv.List[0]
		recvType := formatExpr(recv.Type)
		if len(recv.Names) > 0 {
			parts = append(parts, fmt.Sprintf("(%s %s)", recv.Names[0].Name, recvType))
		} else {
			parts = append(parts, fmt.Sprintf("(%s)", recvType))
		}
	}

	parts = append(parts, fn.Name.Name)

	// parameter list
	if fn.Type.Params != nil {
		parts = append(parts, "("+formatFieldList(fn.Type.Params)+")")
	} else {
		parts = append(parts, "()")
	}

	// return types
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		results := formatFieldList(fn.Type.Results)
		if len(fn.Type.Results.List) > 1 {
			parts = append(parts, "("+results+")")
		} else {
			parts = append(parts, results)
		}
	}

	return strings.Join(parts, " ")
}

// formatFieldList turns an ast.FieldList (like function parameters) into a
// readable string like "x int, y string".
func formatFieldList(fl *ast.FieldList) string {
	if fl == nil {
		return ""
	}
	var parts []string
	for _, field := range fl.List {
		typeName := formatExpr(field.Type)
		if len(field.Names) == 0 {
			parts = append(parts, typeName)
		} else {
			var names []string
			for _, n := range field.Names {
				names = append(names, n.Name)
			}
			parts = append(parts, strings.Join(names, ", ")+" "+typeName)
		}
	}
	return strings.Join(parts, ", ")
}

// formatExpr converts an AST type expression back to a readable string.
// This is a simplified version that handles the most common cases.
func formatExpr(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + formatExpr(e.X)
	case *ast.ArrayType:
		return "[]" + formatExpr(e.Elt)
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", formatExpr(e.Key), formatExpr(e.Value))
	case *ast.SelectorExpr:
		return formatExpr(e.X) + "." + e.Sel.Name
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.FuncType:
		return "func(...)"
	case *ast.ChanType:
		return "chan " + formatExpr(e.Value)
	case *ast.Ellipsis:
		return "..." + formatExpr(e.Elt)
	default:
		return "?"
	}
}

// validateNodeType returns an error if the user passed an unsupported node type.
func validateNodeType(t string) error {
	valid := map[string]bool{
		"func": true, "struct": true, "interface": true,
		"var": true, "const": true, "type": true,
	}
	if !valid[t] {
		return fmt.Errorf("unknown --type %q. Valid types: func, struct, interface, var, const, type", t)
	}
	return nil
}

// printResults renders the results to the terminal in a clean table format.
func printResults(results []Result, cfg Config) {
	fmt.Println()
	output.TitleColor.Printf("AST search — %s declarations", cfg.NodeType)
	if cfg.Name != "" {
		output.TitleColor.Printf(" matching %q", cfg.Name)
	}
	output.TitleColor.Println()
	output.DimColor.Println("────────────────────────────────────────────")
	fmt.Println()

	for _, r := range results {
		// Print the file + line number
		loc := fmt.Sprintf("%s:%d", r.File, r.Line)
		fmt.Printf("%s  ", output.FileColor.Sprint(loc))

		// Print the signature or name
		if r.Sig != "" {
			output.SuccessColor.Printf("func ")
			fmt.Printf("%s\n", r.Sig)
		} else {
			fmt.Printf("%s %s\n", output.DimColor.Sprint(r.NodeType), r.Name)
		}
	}

	fmt.Println()
	output.DimColor.Printf("%d %s declaration(s) found\n", len(results), cfg.NodeType)
	fmt.Println()
}
