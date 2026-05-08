package main

import (
	"fmt"
	"runtime"

	"os"
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

// before:
//   Alloc = 247 KB
//   Sys   = 12630 KB
//   HeapAlloc = 247 KB
//   NumGC = 1

// failure accessing path "/home/m/.cache/yay/chatterino2/pkg": open /home/m/.cache/yay/chatterino2/pkg: permission denied
// after:
//   Alloc = 22094 KB
//   Sys   = 43258 KB
//   HeapAlloc = 22094 KB
//   NumGC = 9

// /home/m
// Files: 158213 PathSums: 13388270
func tree(root string) {
	runtime.GC()
	printMem("before")
	tree := filetree.FromDir(root)
	runtime.GC()
	printMem("after")
	use(&tree)
	filetree.Print(&tree)
}

// before:
//   Alloc = 247 KB
//   Sys   = 12118 KB
//   HeapAlloc = 247 KB
//   NumGC = 1

// failure accessing path "/home/m/.cache/yay/chatterino2/pkg": open /home/m/.cache/yay/chatterino2/pkg: permission denied
// after:
//   Alloc = 17586 KB
//   Sys   = 42942 KB
//   HeapAlloc = 17586 KB
//   NumGC = 11

// Files: 158620 PathSums: 13431054
func paths(root string) {
	runtime.GC()
	printMem("before")
	paths := []string{}
	filepath.WalkDir(root, func (path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("failure accessing path %q: %v\n", path, err)
			// return err
			// ignore the error, continue iteration
			return nil
		}

		paths = append(paths, path)
		return nil
	})
	runtime.GC()
	printMem("after")
	files := 0
	pathsums := 0
	for _, p := range paths {
		// println(p)
		pathsums += len(p)
		files += 1
	}

	fmt.Printf("Files: %v PathSums: %v", files, pathsums)
}

type StrRef struct {
	start uint32
	end_exclusive uint32
}

func pathArena(root string) {
	runtime.GC()
	printMem("before")

	arena := []byte{}
	paths := []StrRef{}
	filepath.WalkDir(root, func (path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("failure accessing path %q: %v\n", path, err)
			// return err
			// ignore the error, continue iteration
			return nil
		}

		start := len(arena) + 1
		arena = append(arena, path...)
		end_exclusive := len(arena)
		paths = append(paths, StrRef{ uint32(start), uint32(end_exclusive) })
		return nil
	})
	runtime.GC()
	printMem("after")
	files := 0
	pathsums := 0
	for _, p := range paths {
		path := string(arena[p.start:p.end_exclusive])
		pathsums += len(path)
		files += 1
	}

	fmt.Printf("Files: %v PathSums: %v", files, pathsums)
}

// tree (firstChild, nextSibling)
// 217835 KB
// Files: 1818124 PathSums: 123642288
// paths
// 166509 KB
// Files: 1817998 PathSums: 123644980
// pathArena
// 167196 KB
// Files: 1832434 PathSums: 124299417

// TODO just storing full paths uses less mem vs storing tree nodes
//      with just the name component + pointers + children
//      paths (16KB~) vs nodes (24KB~)
//      paths (17KB~) vs nodes (22KB~)
//      	-> with firstChild and nextSibling instead of storing a children []*Node
func main() {
	args := os.Args[1:]
	root := args[0]
	if args[1] == "tree" {
		tree(root)
	}
	if args[1] == "path" {
		paths(root)
	}
	if args[1] == "pathArena" {
		pathArena(root)
	}
}
