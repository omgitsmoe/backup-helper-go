package main

import (
	// "errors"
	"fmt"
	"runtime"
	"time"

	"os"

	"github.com/omgitsmoe/backup-helper-go/pkg/checksum"
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
		memBefore.HeapAlloc/1024, memAfter.HeapAlloc/1024, (memAfter.HeapAlloc-memBefore.HeapAlloc)/1024)
}

func main() {
	args := os.Args[1:]
	root := args[0]
	_ = root

	options := checksum.DefaultOptions()
	checker, err := checksum.NewCheckerWithOptions(root, options)
	if err != nil {
		fmt.Printf("failed to create checker: %s\n", err)
		os.Exit(1)
	}

	buildMostCurrent(&checker)
}

func buildMostCurrent(checker *checksum.Checker) {
	mostCurrent, err := checker.BuildMostCurrent(nil)
	if err != nil {
		fmt.Printf("buildMostCurrent failed: %s\n", err)
		os.Exit(1)
	}

	path, err := mostCurrent.Path()
	if err != nil {
		fmt.Printf("buildMostCurrent failed: %s\n", err)
		os.Exit(1)
	}

	f, err := os.Create(path)
	if err != nil {
		fmt.Printf("buildMostCurrent failed to create file: %s\n", err)
		os.Exit(1)
	}
	defer f.Close()

	ser := checksum.NewSerializer(f)
	err = ser.Flush(mostCurrent)
	if err != nil {
		fmt.Printf("buildMostCurrent failed to write file: %s\n", err)
		os.Exit(1)
	}
}
