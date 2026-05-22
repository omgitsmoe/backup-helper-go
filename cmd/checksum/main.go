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
	// path := args[1]

	options := checksum.DefaultOptions()
	checker, err := checksum.NewCheckerWithOptions(root, options)
	if err != nil {
		fmt.Printf("failed to create checker: %s\n", err)
		os.Exit(1)
	}

	// buildMostCurrent(&checker)
	// verify(&checker, path)
	// incremental(&checker)
	checkMissing(&checker)
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

func verify(checker *checksum.Checker, path string) {
	collection, err := checker.Read(path)
	if err != nil {
		fmt.Printf("Failed to read collection: %s\n", err)
		os.Exit(1)
	}

	reporter := NewProgressReporter()

	err = checker.Verify(collection, nil, func(p checksum.VerifyProgress) bool {
		reporter.ReportVerify(&p)

		return true
	})

	if err != nil {
		fmt.Printf("Verify failed: %s\n", err)
		os.Exit(1)
	}
}

func incremental(checker *checksum.Checker) {
	reporter := NewProgressReporter()

	inc, err := checker.Incremental(func(p checksum.ProgressEvent) {
		reporter.Report(p)
	})
	if err != nil {
		fmt.Printf("incremental failed: %s\n", err)
		os.Exit(1)
	}

	path, err := inc.Path()
	if err != nil {
		fmt.Printf("incremental failed: %s\n", err)
		os.Exit(1)
	}

	f, err := os.Create(path)
	if err != nil {
		fmt.Printf("incremental failed to create file: %s\n", err)
		os.Exit(1)
	}
	defer f.Close()

	ser := checksum.NewSerializer(f)
	err = ser.Flush(inc)
	if err != nil {
		fmt.Printf("incremental failed to write file: %s\n", err)
		os.Exit(1)
	}
}

func checkMissing(checker *checksum.Checker) {
	reporter := NewProgressReporter()

	missing, err := checker.CheckMissing(func(p checksum.ProgressEvent) {
		reporter.Report(p)
	})
	if err != nil {
		fmt.Printf("incremental failed: %s\n", err)
		os.Exit(1)
	}

	if len(missing.Directories) > 0 {
		fmt.Println("Directories that are completely missing:")
		for _, d := range missing.Directories {
			fmt.Printf("\t%q\n", d)
		}
	}
	if len(missing.Files) > 0 {
		fmt.Println("Files that are missing:")
		for _, f := range missing.Files {
			fmt.Printf("\t%q\n", f)
		}
	}

	if len(missing.Files) == 0 && len(missing.Directories) == 0 {
		fmt.Println("Success! All files have a known hash! (No mtime check was made!)")
	}
}
