package main

import (
	// "errors"
	"fmt"
	"runtime"
	"sort"
	"time"

	"io/fs"
	"os"
	"path/filepath"

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
    var memBefore runtime.MemStats
    runtime.ReadMemStats(&memBefore)

    start := time.Now()
    fn()

    var memAfter runtime.MemStats
    runtime.ReadMemStats(&memAfter)

    fmt.Printf("[%s] took %s\n", name, time.Since(start))
    fmt.Printf("[%s] mem b4 %v KB after %v KB diff %v KB\n", name,
		memBefore.HeapAlloc / 1024, memAfter.HeapAlloc / 1024, (memAfter.HeapAlloc - memBefore.HeapAlloc) / 1024)
}

func use(tree *filetree.FileTree) {
	println(tree.Root())
}

// File(name) with *Dir(storing the path) + dirpath->*Dir map
// HeapAlloc = 179468 KB
//
// Files: 1733975 PathSums: 116735436
func tree(paths []string) {
	var tree filetree.FileTree
	var mapping map[*filetree.Node]int
	measure("tree collect", func() {
		tree, mapping = filetree.FromPaths(paths)
	})

	files := 0
	pathsums := 0
	measure("tree iter", func() {
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
func paths(allPaths []string) {
	paths := []string{}
	mapping := make(map[string]int)
	measure("paths collect", func() {
		for _, path := range allPaths {

			paths = append(paths, path)
			mapping[path] = 0
		}
	})

	files := 0
	pathsums := 0
	measure("paths iter", func() {
		for _, p := range paths {
			// println(p)
			pathsums += len(p)
			files += 1
		}
	})
	println("mappinglen", len(mapping))

	fmt.Printf("Files: %v PathSums: %v", files, pathsums)
}

func store(allPaths []string) {
	ids := []pathstore.PathId{}
	store := pathstore.NewStore()

	measure("store collect", func() {
		for _, path := range allPaths {

			id := store.Store(path)
			ids = append(ids, id)
		}
	})

	files := 0
	pathsums := 0
	measure("store iter", func() {
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
	_ = root

	allPaths := []string{}
	filepath.WalkDir(root, func (path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("failure accessing path %q: %v\n", path, err)
			// return err
			// ignore the error, continue iteration
			return nil
		}

		allPaths = append(allPaths, path)

		return nil
	})

	allPaths = append(allPaths, generatePaths(8_000_000)...)
	sort.Slice(allPaths, func(i, j int) bool {
		return i < j
	})

	if args[1] == "tree" {
		tree(allPaths)
	}
	if args[1] == "path" {
		paths(allPaths)
	}
	if args[1] == "store" {
		store(allPaths)
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

// [store collect] took 2.180165926s
// [store collect] mem b4 233 KB after 294792 KB diff 294558 KB
// [store iter] took 1.047266ms
// [store iter] mem b4 294792 KB after 294792 KB diff 0 KB
// Files: 1785519 PathSums: 127327533

// [paths collect] took 2.813949478s
// [paths collect] mem b4 228 KB after 230456 KB diff 230228 KB
// [paths iter] took 849.296µs
// [paths iter] mem b4 230456 KB after 230456 KB diff 0 KB
// mappinglen 1671910
// Files: 1793529 PathSums: 127577597

// [tree collect] took 5.004839904s
// [tree collect] mem b4 228 KB after 279688 KB diff 279459 KB
// [tree iter] took 479.733522ms
// [tree iter] mem b4 279688 KB after 371708 KB diff 92020 KB
// Files: 1676075 PathSums: 120634694


// --------- 3.8m~ paths -----
// [tree collect] took 960.273521ms
// [tree collect] mem b4 486013 KB after 1376323 KB diff 890309 KB
// [tree iter] took 1.22423254s
// [tree iter] mem b4 1376325 KB after 1401412 KB diff 25086 KB
// Files: 3834988 PathSums: 206726838

// [paths collect] took 554.336534ms
// [paths collect] mem b4 447961 KB after 608179 KB diff 160218 KB
// [paths iter] took 1.661087ms
// [paths iter] mem b4 608182 KB after 608182 KB diff 0 KB
// mappinglen 3515822
// Files: 3835323 PathSums: 195202461

// [store collect] took 679.386402ms
// [store collect] mem b4 488256 KB after 558451 KB diff 70194 KB
// [store iter] took 3.07334ms
// [store iter] mem b4 558453 KB after 558453 KB diff 0 KB
// Files: 3833484 PathSums: 195121506


// --------- 10m~ paths -----
// [tree collect] took 2.892981609s
// [tree collect] mem b4 994752 KB after 4166982 KB diff 3172230 KB
// [tree iter] took 3.180931648s
// [tree iter] mem b4 4166984 KB after 4942590 KB diff 775605 KB
// Files: 9841460 PathSums: 422416490

// [paths collect] took 1.914580014s
// [paths collect] mem b4 989676 KB after 1492862 KB diff 503186 KB
// [paths iter] took 4.183512ms
// [paths iter] mem b4 1492864 KB after 1492864 KB diff 0 KB
// mappinglen 8424017
// Files: 9829023 PathSums: 392486077

// [store collect] took 2.039597337s
// [store collect] mem b4 759038 KB after 1915315 KB diff 1156277 KB
// [store iter] took 10.683454ms
// [store iter] mem b4 1915317 KB after 1915317 KB diff 0 KB
// Files: 9821566 PathSums: 392308039
