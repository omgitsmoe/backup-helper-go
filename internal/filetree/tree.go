package filetree

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
	"path/filepath"
)

type FileTree struct {
	root *Node
}

func NewTree(root string) FileTree {
	return FileTree{
		root: NewNode(root, true, nil),
	}
}

func (t *FileTree) Root() string {
	return t.root.Name
}

func FromDir(root string) FileTree {
	root = filepath.Clean(root)
	tree := NewTree(root)
	dirstack := []Frame{
		{
			Node: tree.root,
			Path: root,
		},
	}

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
		}

		return nil
	})

	return tree
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
