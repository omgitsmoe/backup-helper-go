package main

import (
	"fmt"
	"runtime"

	"path/filepath"
	"io/fs"

    // "github.com/omgitsmoe/backup-helper-go/pkg/checksum"
    "github.com/omgitsmoe/backup-helper-go/internal/filetree"
)


func printMem(tag string) {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)

    fmt.Printf("%s:\n", tag)
    fmt.Printf("  Alloc = %v KB\n", m.Alloc/1024)
    fmt.Printf("  Sys   = %v KB\n", m.Sys/1024)
    fmt.Printf("  HeapAlloc = %v KB\n", m.HeapAlloc/1024)
    fmt.Printf("  NumGC = %v\n\n", m.NumGC)
}

func use(tree *filetree.FileTree) {
	println(tree.Root())
}

// TODO just storing full paths uses less mem vs storing tree nodes
//      with just the name component + pointers + children
//      paths (16KB~) vs nodes (24KB~)
func main() {
	runtime.GC()
	printMem("before")
	root := "/home/m/"
	// tree := filetree.FromDir(root)
	paths := []string{}
	filepath.WalkDir(root, func (path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("failure accessing path %q: %v\n", path, err)
			return err
		}

		paths = append(paths, path)
		return nil
	})
	runtime.GC()
	printMem("after")
	for _, p := range paths {
		println(p)
		break;
	}
	// use(&tree)
	// filetree.Print(&tree)
}
