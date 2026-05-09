package filetree

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
	"path/filepath"
)

func (t *FileTree) Root() string {
	return t.root.Name
}

func FromDir(root string) (FileTree, map[*Node]int) {
	root = filepath.Clean(root)
	tree := NewTree(root)
	dirstack := []Frame{
		{
			Node: tree.root,
			Path: root,
		},
	}

	mapping := make(map[*Node]int)

	// TODO use ReadDir directly instead of abusing WalkDir
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("failure accessing path %q: %v\n", path, err)
			// return err
			// ignore the error, continue iteration
			return nil
		}

		// root also visited -> skip
		if path == root {
			return nil
		}

		parent := filepath.Dir(path)
		curr := dirstack[len(dirstack)-1]
		for parent != curr.Path {
			dirstack = dirstack[:len(dirstack)-1]
			curr = dirstack[len(dirstack)-1]
		}

		isdir := d.IsDir()
		name := filepath.Base(path);
		child := NewNode(name, isdir, curr.Node)
		curr.Node.AddChild(child)

		if isdir {
			dirstack = append(dirstack, Frame{
				Node: child,
				Path: path,
			})
		} else {
			mapping[child] = 0
		}

		return nil
	})

	return tree, mapping
}

type FileTree struct {
	root *Node
	Dirs []*Node
}

func NewTree(rootName string) FileTree {
	r := &Node{
		Name:  rootName,
		IsDir: true,
		Index: -1,
	}
	return FileTree{root: r}
}

func commonPrefixLen(a, b string) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return i
}

// FromPaths expects paths sorted lexicographically and relative.
// It builds a tree in one pass and avoids repeated traversal.
func FromPaths(paths []string) (FileTree, map[*Node]int) {
	tree := NewTree("")
	root := tree.root

	// stack of nodes for the current path, excluding root
	stack := make([]*Node, 0, 64)

	mapping := make(map[*Node]int)

	var prev string

	for _, p := range paths {
		if p == "" || p == "/" {
			continue
		}

		// find common byte prefix with previous path
		lcp := commonPrefixLen(prev, p)

		// backtrack to the last slash boundary so we keep only full components
		for lcp > 0 && p[lcp-1] != '/' {
			lcp--
		}

		// count how many components are kept
		keep := 0
		for i := 0; i < lcp; i++ {
			if p[i] == '/' {
				keep++
			}
		}

		if keep < len(stack) {
			stack = stack[:keep]
		}

		parent := root
		if keep > 0 {
			parent = stack[keep-1]
		}

		// create only the suffix that changed
		start := lcp
		for start < len(p) {
			end := start
			for end < len(p) && p[end] != '/' {
				end++
			}

			if end > start {
				isDir := end < len(p)
				n := &Node{
					Name:  p[start:end],
					IsDir: isDir,
					Index: -1,
				}
				if !n.IsDir {
					mapping[n] = 0
				}

				parent.AddChild(n)

				if isDir {
					n.Index = len(tree.Dirs)
					tree.Dirs = append(tree.Dirs, n)
				}

				stack = append(stack, n)
				parent = n
			}

			start = end + 1
		}

		prev = p
	}

	return tree, mapping
}

type Frame struct {
	Node *Node
	Path string
}

func Print(tree *FileTree) {
	stack := []Frame {
		{
			Node: tree.root,
			Path: tree.root.Name,
		},
	}

	sep := string(os.PathSeparator)

	files := 0
	pathsums := 0
	for len(stack) > 0 {
		curr := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		child := curr.Node.firstChild
		for child != nil {
			stack = append(stack, Frame{
				Node: child,
				Path: filepath.Join(curr.Path, child.Name),
			})

			child = child.nextSibling
		}

		path := curr.Path
		pathsums += len(path)
		files += 1

		if curr.Node.IsDir && !strings.HasSuffix(path, sep) {
			path += sep
		}

		// fmt.Println(path)
	}

	fmt.Printf("Files: %v PathSums: %v", files, pathsums)
}
