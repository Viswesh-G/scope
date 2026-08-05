// This file implements `scope graph` — visualising the repository structure
// as either a terminal ASCII tree or a Graphviz DOT file.
//
// The tree is built by walking the filesystem and constructing a Node tree
// in memory, then rendered top-down using recursive print functions.
package analysis

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Viswesh-G/scope/internal/ignore"
	"github.com/Viswesh-G/scope/internal/output"
)

// Node represents one entry (file or directory) in the directory tree.
type Node struct {
	Name     string  // display name (basename only)
	IsDir    bool    // true for directories, false for files
	Children []*Node // child nodes (only non-empty for directories)
}

// RunGraph builds a tree of the repository structure and prints it.
// If dot is true, it prints Graphviz DOT format instead of an ASCII tree.
func RunGraph(path string, dot bool) error {
	ig, err := ignore.LoadIgnoreFile(filepath.Join(path, ".scope-ignore"))
	if err != nil {
		return fmt.Errorf("loading .scope-ignore: %w", err)
	}

	// Create the root node. If path is "." resolve to the real directory name.
	rootName := filepath.Base(path)
	if rootName == "." || rootName == "" {
		abs, _ := filepath.Abs(path)
		rootName = filepath.Base(abs)
	}
	root := &Node{Name: rootName, IsDir: true}

	// nodes maps each filesystem path to its Node so we can find parents quickly.
	nodes := map[string]*Node{path: root}

	_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil || p == path {
			return nil // skip the root itself (already added)
		}
		if d.IsDir() && ignore.ShouldSkipDir(d.Name(), ig) {
			return filepath.SkipDir
		}
		if !d.IsDir() && ignore.ShouldSkipFile(p, ig) {
			return nil
		}

		// Find this entry's parent node and attach a new child node to it.
		parentPath := filepath.Dir(p)
		parentNode, ok := nodes[parentPath]
		if !ok {
			return nil // shouldn't happen in WalkDir's lexical traversal
		}

		node := &Node{Name: d.Name(), IsDir: d.IsDir()}
		nodes[p] = node
		parentNode.Children = append(parentNode.Children, node)
		return nil
	})

	if dot {
		// Output Graphviz DOT format for tools like `dot -Tsvg`.
		fmt.Println("digraph G {")
		fmt.Println(`  node [shape=box, style=filled, color="#E0E0E0"];`)
		fmt.Println(`  edge [color="#666666"];`)
		printDot(root, root.Name)
		fmt.Println("}")
	} else {
		// Output a coloured ASCII tree.
		output.PrintHeader("Directory Graph", path)
		fmt.Println(output.FileColor.Sprint(root.Name))
		printTree(root, "")
		fmt.Println()
	}

	return nil
}

// printTree recursively renders the tree as indented ASCII art using
// box-drawing characters (├── and └──).
func printTree(node *Node, prefix string) {
	for i, child := range node.Children {
		isLast := i == len(node.Children)-1

		fmt.Print(prefix)
		if isLast {
			fmt.Print(" └── ")
		} else {
			fmt.Print(" ├── ")
		}

		if child.IsDir {
			// Directories are colored and recursed into.
			fmt.Println(output.FileColor.Sprint(child.Name))
			// Extend the prefix for children: vertical bar if more siblings follow,
			// blank space if this was the last sibling.
			newPrefix := prefix
			if isLast {
				newPrefix += "     "
			} else {
				newPrefix += " │   "
			}
			printTree(child, newPrefix)
		} else {
			fmt.Println(child.Name)
		}
	}
}

// printDot recursively emits Graphviz DOT edges and node labels.
// Node IDs are derived from paths with special characters sanitized.
func printDot(node *Node, parentID string) {
	for _, child := range node.Children {
		// Build a unique ID for this node by appending the name to the parent's ID.
		childID := parentID + "_" + child.Name
		// Graphviz node IDs can't contain dots or hyphens — replace them.
		childID = strings.ReplaceAll(childID, ".", "_")
		childID = strings.ReplaceAll(childID, "-", "_")

		parentSanitized := strings.ReplaceAll(parentID, ".", "_")
		parentSanitized = strings.ReplaceAll(parentSanitized, "-", "_")

		// Emit an edge from parent to child.
		fmt.Printf("  %s -> %s;\n", parentSanitized, childID)

		if child.IsDir {
			fmt.Printf("  %s [label=\"%s\", shape=folder];\n", childID, child.Name)
			printDot(child, childID)
		} else {
			fmt.Printf("  %s [label=\"%s\"];\n", childID, child.Name)
		}
	}
}
