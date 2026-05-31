package analysis

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Viswesh-G/scope/internal/ignore"
	"github.com/Viswesh-G/scope/internal/output"
)

type Node struct {
	Name     string
	IsDir    bool
	Children []*Node
}

func RunGraph(path string, dot bool) error {
	ig, err := ignore.LoadIgnoreFile(filepath.Join(path, ".scope-ignore"))
	if err != nil {
		return fmt.Errorf("loading .scope-ignore: %w", err)
	}

	root := &Node{
		Name:  filepath.Base(path),
		IsDir: true,
	}
	if root.Name == "." || root.Name == "" {
		abs, _ := filepath.Abs(path)
		root.Name = filepath.Base(abs)
	}

	nodes := map[string]*Node{
		path: root,
	}

	_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil || p == path {
			return nil
		}
		if d.IsDir() && ignore.ShouldSkipDir(d.Name(), ig) {
			return filepath.SkipDir
		}
		if !d.IsDir() && ignore.ShouldSkipFile(p, ig) {
			return nil
		}

		parentPath := filepath.Dir(p)
		parentNode, ok := nodes[parentPath]
		if !ok {
			return nil // Should not happen in WalkDir
		}

		node := &Node{
			Name:  d.Name(),
			IsDir: d.IsDir(),
		}
		nodes[p] = node
		parentNode.Children = append(parentNode.Children, node)

		return nil
	})

	if dot {
		fmt.Println("digraph G {")
		fmt.Println(`  node [shape=box, style=filled, color="#E0E0E0"];`)
		fmt.Println(`  edge [color="#666666"];`)
		printDot(root, root.Name)
		fmt.Println("}")
	} else {
		fmt.Println(output.FileColor.Sprint(root.Name))
		printTree(root, "")
	}

	return nil
}

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
			fmt.Println(output.FileColor.Sprint(child.Name))
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

func printDot(node *Node, parentID string) {
	for _, child := range node.Children {
		childID := parentID + "_" + child.Name
		childID = strings.ReplaceAll(childID, ".", "_")
		childID = strings.ReplaceAll(childID, "-", "_")
		parentSanitized := strings.ReplaceAll(parentID, ".", "_")
		parentSanitized = strings.ReplaceAll(parentSanitized, "-", "_")
		
		fmt.Printf("  %s -> %s;\n", parentSanitized, childID)
		if child.IsDir {
			fmt.Printf("  %s [label=\"%s\", shape=folder];\n", childID, child.Name)
			printDot(child, childID)
		} else {
			fmt.Printf("  %s [label=\"%s\"];\n", childID, child.Name)
		}
	}
}
