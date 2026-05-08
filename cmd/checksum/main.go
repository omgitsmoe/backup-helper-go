package main

import (
	"fmt"
	"runtime"
	"time"

	"os"

    // "github.com/omgitsmoe/backup-helper-go/pkg/checksum"
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

func main() {
	args := os.Args[1:]
	root := args[0]
	println("root", root)
}
