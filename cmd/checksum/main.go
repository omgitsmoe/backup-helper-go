package main

import (
	"fmt"
	"runtime"
	"time"

	"os"
	"path/filepath"
	"io/fs"

    // "github.com/omgitsmoe/backup-helper-go/pkg/checksum"
    "github.com/omgitsmoe/backup-helper-go/internal/filetree"
    "github.com/omgitsmoe/backup-helper-go/pkg/checksum/pathstore"
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

func measure(name string, fn func()) {
    start := time.Now()
    fn()
    fmt.Printf("%s took %s\n", name, time.Since(start))
}

func use(tree *filetree.FileTree) {
	println(tree.Root())
}

// File(name) with *Dir(storing the path) + dirpath->*Dir map
// HeapAlloc = 179468 KB
//
// Files: 1733975 PathSums: 116735436
func tree(root string) {
	runtime.GC()
	printMem("before")
	tree, mapping := filetree.FromDir(root)
	runtime.GC()
	printMem("after")

	files := 0
	pathsums := 0
	measure("tree", func() {
		for n, _ := range mapping {
			// println(p)
			pathsums += len(n.Path())
			files += 1
		}
	})

	fmt.Printf("Files: %v PathSums: %v", files, pathsums)
	use(&tree)
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
	mapping := make(map[string]int)
	filepath.WalkDir(root, func (path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("failure accessing path %q: %v\n", path, err)
			// return err
			// ignore the error, continue iteration
			return nil
		}

		paths = append(paths, path)
		if !d.IsDir() {
			mapping[path] = 0
		}
		return nil
	})
	runtime.GC()
	printMem("after")
	files := 0
	pathsums := 0
	measure("paths", func() {
		for _, p := range paths {
			// println(p)
			pathsums += len(p)
			files += 1
		}
	})
	println("mappinglen", len(mapping))

	fmt.Printf("Files: %v PathSums: %v", files, pathsums)
}

func store(root string) {
	runtime.GC()
	printMem("before")

	ids := []pathstore.PathId{}
	store := pathstore.NewStore()
	filepath.WalkDir(root, func (path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("failure accessing path %q: %v\n", path, err)
			// return err
			// ignore the error, continue iteration
			return nil
		}

		id := store.Store(path)
		ids = append(ids, id)
		return nil
	})
	runtime.GC()
	printMem("after")
	files := 0
	pathsums := 0
	measure("store", func() {
		for _, p := range ids {
			path := store.Lookup(p)
			pathsums += len(path)
			files += 1
		}
	})

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
	if args[1] == "store" {
		store(root)
	}
}


// tree ---
// after:
//   HeapAlloc = 290917 KB
// tree took 543.896098ms
// Files: 1843411 PathSums: 124399683

// paths ---
// after:
//   HeapAlloc = 260175 KB
// paths took 789.645µs
// mappinglen 1843557
// Files: 1972790 PathSums: 131550844

// store --- (with interned paths, but +1 store for fast lookup)
//   HeapAlloc = 300205 KB
// store took 1.187969ms
// Files: 1973374 PathSums: 131773893
