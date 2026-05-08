package main

import (
	"fmt"
	"runtime"

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

type foo struct {
	name string
	id int
}

var fooToStr = map[foo]string {
	{ name: "foo", id: 3 }: "bar",
}

func main() {
	args := os.Args[1:]
	root := args[0]
	println("Using root", root)
	checksum.Foo()
}
