package filetree

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

type Dir struct {
	Path string
}

type File struct {
	Dir *Dir
	Name string
}

func (f *File) Path() string {
	return filepath.Join(f.Dir.Path, f.Name)
}

type FileTree struct {
	root string
	dirToNode map[string]*Dir
}

func NewTree(root string) FileTree {
	tree := FileTree{
		root: root,
	}

	tree.dirToNode = make(map[string]*Dir)
	tree.dirToNode[root] = &Dir{ Path: root }

	return tree
}

func (t *FileTree) Root() string {
	return t.root
}

func FromDir(root string) (FileTree, []File) {
	root = filepath.Clean(root)
	tree := NewTree(root)
	files := []File{}

	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("failure accessing path %q: %v\n", path, err)
			// return err
			// ignore the error, continue iteration
			return nil
		}

		if d.IsDir() {
			return nil
		}

		// root also visited -> skip
		if path == root {
			return nil
		}

		dir := filepath.Dir(path)
		name := filepath.Base(path);
		
		dirNode, present := tree.dirToNode[dir]
		if !present {
			dirNode = &Dir{ Path: dir }
			tree.dirToNode[dir] = dirNode
		}
		files = append(files, File{Name: name, Dir: dirNode})

		return nil
	})

	return tree, files
}
