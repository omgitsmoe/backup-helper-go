package main

import (
	// "errors"
	"fmt"
	"path/filepath"
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
	incremental(&checker)
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

	var currentFile string

	err = checker.Verify(collection, func(p checksum.VerifyProgress) bool {
		switch p.Stage {

		case checksum.VerifyPre:
			path := filepath.Join(p.Common.TreeRoot, p.Common.RelativePath)
			currentFile = path

			fmt.Printf(
				"\n[VERIFY] (%4d/%4d) %s\n",
				p.Common.FileNumberProcessed,
				p.Common.FileNumberTotal,
				path,
			)

			fmt.Printf(
				"[PROG  ] bytes %10d / %10d\n",
				p.Common.SizeProcessedBytes,
				p.Common.SizeTotalBytes,
			)

		case checksum.VerifyDuring:
			if currentFile != "" {
				percent := 0.0
				if p.Total > 0 {
					percent = float64(p.Done) / float64(p.Total) * 100.0
				}

				fmt.Printf(
					"\r[HASH  ] %-30s %8d/%8d bytes (%5.1f%%)",
					filepath.Base(currentFile),
					p.Done,
					p.Total,
					percent,
				)
			} else {
				fmt.Printf(
					"\r[HASH  ] %8d/%8d bytes",
					p.Done,
					p.Total,
				)
			}

		case checksum.VerifyPost:
			fmt.Print("\n")

			path := filepath.Join(p.Common.TreeRoot, p.Common.RelativePath)

			var status string
			switch p.Result {

			case checksum.VerifyOK:
				status = "[OK        ]"
			case checksum.VerifyFileMissing:
				status = "[ERR MISS  ]"
			case checksum.VerifyMismatch:
				status = "[ERR HASH  ]"
			case checksum.VerifyMismatchSize:
				status = "[ERR SIZE  ]"
			case checksum.VerifyMismatchCorrupted:
				status = "[ERR CORR  ]"
			case checksum.VerifyMismatchOutdatedHash:
				status = "[WARN STALE]"
			default:
				status = "[UNKNOWN   ]"
			}

			fmt.Printf("%s %s\n", status, path)
		}

		return true
	})

	if err != nil {
		fmt.Printf("Verify failed: %s\n", err)
		os.Exit(1)
	}
}

func incremental(checker *checksum.Checker) {
	inc, err := checker.Incremental(nil)
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
