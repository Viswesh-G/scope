package navigation

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

type Config struct {
	Path string
	Name string
	File string
	Line int
}

type Location struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Package string `json:"package"`
}

type Result struct {
	Name       string     `json:"name"`
	Kind       string     `json:"kind"`
	Definition Location   `json:"definition"`
	References []Location `json:"references"`
}

type definition struct {
	object   types.Object
	key      string
	location Location
}

func FindReferences(cfg Config) ([]Result, error) {
	if cfg.Path == "" {
		return nil, fmt.Errorf("package path must not be empty")
	}
	if cfg.Name == "" {
		return nil, fmt.Errorf("symbol name must not be empty")
	}
	if cfg.Line < 0 {
		return nil, fmt.Errorf("line must not be negative")
	}
	if cfg.Line > 0 && cfg.File == "" {
		return nil, fmt.Errorf("--line requires --file")
	}

	root, err := filepath.Abs(cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("resolving package path: %w", err)
	}
	var selectedFile string
	if cfg.File != "" {
		selectedFile = cfg.File
		if !filepath.IsAbs(selectedFile) {
			selectedFile = filepath.Join(root, selectedFile)
		}
		selectedFile, err = filepath.Abs(selectedFile)
		if err != nil {
			return nil, fmt.Errorf("resolving source file: %w", err)
		}
	}

	pkgs, err := packages.Load(&packages.Config{
		Mode:  packages.LoadAllSyntax,
		Dir:   root,
		Tests: true,
	}, "./...")
	if err != nil {
		return nil, fmt.Errorf("loading Go packages: %w", err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no Go packages found under %q", cfg.Path)
	}
	if err := packageErrors(pkgs); err != nil {
		return nil, err
	}

	var definitions []definition
	seenDefinitions := make(map[string]bool)
	for _, pkg := range pkgs {
		if pkg.TypesInfo == nil {
			continue
		}
		for ident, object := range pkg.TypesInfo.Defs {
			if object == nil || object.Name() != cfg.Name || !isInRoot(root, pkg.Fset.Position(ident.Pos()).Filename) {
				continue
			}
			position := pkg.Fset.Position(ident.Pos())
			file, err := filepath.Abs(position.Filename)
			if err != nil {
				return nil, fmt.Errorf("resolving definition path: %w", err)
			}
			if selectedFile != "" && !samePath(file, selectedFile) {
				continue
			}
			if cfg.Line > 0 && position.Line != cfg.Line {
				continue
			}
			location := makeLocation(root, position, pkg.PkgPath)
			key := fmt.Sprintf("%s:%d:%d:%s", file, position.Line, position.Column, object.Type().String())
			if seenDefinitions[key] {
				continue
			}
			seenDefinitions[key] = true
			definitions = append(definitions, definition{
				object:   object,
				key:      objectKey(pkg.Fset, object),
				location: location,
			})
		}
	}

	if len(definitions) == 0 {
		if cfg.File != "" {
			return nil, fmt.Errorf("no type-checked definition of %q found at %s:%d", cfg.Name, cfg.File, cfg.Line)
		}
		return nil, fmt.Errorf("no type-checked definition of %q found under %q", cfg.Name, cfg.Path)
	}
	sort.Slice(definitions, func(i, j int) bool {
		return locationKey(definitions[i].location) < locationKey(definitions[j].location)
	})
	if len(definitions) > 1 {
		return nil, ambiguousDefinitions(cfg.Name, definitions)
	}

	selected := definitions[0]
	result := Result{
		Name:       selected.object.Name(),
		Kind:       objectKind(selected.object),
		Definition: selected.location,
		References: make([]Location, 0),
	}

	seenReferences := make(map[string]bool)
	for _, pkg := range pkgs {
		if pkg.TypesInfo == nil {
			continue
		}
		addReferences(root, pkg, pkg.TypesInfo.Defs, selected, seenReferences, &result)
		addReferences(root, pkg, pkg.TypesInfo.Uses, selected, seenReferences, &result)
	}
	sort.Slice(result.References, func(i, j int) bool {
		return locationKey(result.References[i]) < locationKey(result.References[j])
	})
	return []Result{result}, nil
}

func objectKind(object types.Object) string {
	switch object.(type) {
	case *types.Func:
		return "func"
	case *types.TypeName:
		return "type"
	case *types.Var:
		return "var"
	case *types.Const:
		return "const"
	case *types.PkgName:
		return "package"
	case *types.Label:
		return "label"
	case *types.Builtin:
		return "builtin"
	default:
		return "symbol"
	}
}

func packageErrors(pkgs []*packages.Package) error {
	var problems []string
	for _, pkg := range pkgs {
		for _, problem := range pkg.Errors {
			problems = append(problems, fmt.Sprintf("%s: %s", pkg.PkgPath, problem.Msg))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	if len(problems) > 5 {
		problems = append(problems[:5], fmt.Sprintf("and %d more package errors", len(problems)-5))
	}
	return fmt.Errorf("could not type-check packages: %s", strings.Join(problems, "; "))
}

func objectKey(fset *token.FileSet, object types.Object) string {
	position := fset.Position(object.Pos())
	packagePath := ""
	if object.Pkg() != nil {
		packagePath = object.Pkg().Path()
	}
	return fmt.Sprintf("%s:%s:%s:%d:%d", packagePath, object.Name(), position.Filename, position.Line, position.Column)
}

func addReferences(root string, pkg *packages.Package, objects map[*ast.Ident]types.Object, target definition, seen map[string]bool, result *Result) {
	for ident, object := range objects {
		if object == nil {
			continue
		}
		if object != target.object && objectKey(pkg.Fset, object) != target.key {
			continue
		}
		if !isInRoot(root, pkg.Fset.Position(ident.Pos()).Filename) {
			continue
		}
		position := pkg.Fset.Position(ident.Pos())
		location := makeLocation(root, position, pkg.PkgPath)
		key := sourceLocationKey(location)
		if key == sourceLocationKey(result.Definition) || seen[key] {
			continue
		}
		seen[key] = true
		result.References = append(result.References, location)
	}
}

func makeLocation(root string, position token.Position, packagePath string) Location {
	file := position.Filename
	if relative, err := filepath.Rel(root, file); err == nil {
		file = relative
	}
	return Location{
		File:    filepath.ToSlash(file),
		Line:    position.Line,
		Column:  position.Column,
		Package: packagePath,
	}
}

func isInRoot(root, path string) bool {
	if path == "" {
		return false
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(root, absolute)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return false
	}
	return !filepath.IsAbs(relative)
}

func samePath(left, right string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

func sourceLocationKey(location Location) string {
	return fmt.Sprintf("%s:%08d:%08d", location.File, location.Line, location.Column)
}

func locationKey(location Location) string {
	return fmt.Sprintf("%s:%08d:%08d:%s", location.File, location.Line, location.Column, location.Package)
}

func ambiguousDefinitions(name string, definitions []definition) error {
	var locations []string
	for _, item := range definitions {
		locations = append(locations, fmt.Sprintf("%s:%d", item.location.File, item.location.Line))
	}
	return fmt.Errorf("symbol %q has multiple definitions (%s); narrow the search with --file and --line", name, strings.Join(locations, ", "))
}
