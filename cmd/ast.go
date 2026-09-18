// cmd/ast.go is the "scp ast" command: search Go source files by code structure,
// not raw text. You can find all functions, structs, interfaces etc. whose names
// match a glob pattern — something a regex search fundamentally cannot do.
//
// This uses Go's own "go/ast" and "go/parser" packages from the standard library.
// No CGO, no external tools needed.
//
// Examples:
//
//	scp ast --type func --name "handle*"   → finds all functions starting with "handle"
//	scp ast --type struct                  → lists every struct in the codebase
//	scp ast --type interface --name "*er"  → finds all interface types ending in "er"
package cmd

import (
	"github.com/Viswesh-G/scope/internal/ast"
	"github.com/spf13/cobra"
)

var (
	astNodeType string
	astName     string
	astPath     string
)

var astCmd = &cobra.Command{
	Use:   "ast",
	Short: "Search Go source by code structure (func, struct, interface...)",
	Long: `Search Go source files by AST node type and name pattern.

This is structural search: it understands Go code, so it only matches
real declarations — not comments, strings, or variable names that happen
to look like function names.

Supported types: func, struct, interface, var, const, type

Examples:
  scp ast --type func --name "handle*"
  scp ast --type struct --path ./internal
  scp ast --type interface --name "*er"
  scp ast --type func`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := ast.Config{
			NodeType: astNodeType,
			Name:     astName,
			Path:     astPath,
		}
		return ast.Run(cfg)
	},
}

func init() {
	rootCmd.AddCommand(astCmd)

	f := astCmd.Flags()
	f.StringVar(&astNodeType, "type", "", "node type to search: func, struct, interface, var, const, type (required)")
	f.StringVar(&astName, "name", "", "glob pattern to match the declaration name (e.g. handle*, *er)")
	f.StringVar(&astPath, "path", ".", "directory to search (defaults to current directory)")

	_ = astCmd.MarkFlagRequired("type")
}
