package main

import (
	"fmt"
	"path/filepath"

	"github.com/omgitsmoe/backup-helper-go/pkg/checksum"
)

// ProgressReporter mirrors the Rust ProgressReporter.
type ProgressReporter struct {
	checksumFilesFound   uint64
	checksumFilesIgnored uint64
	filesFound           uint64
	filesIgnored         uint64
	currentFile          string
}

func NewProgressReporter() *ProgressReporter {
	return &ProgressReporter{}
}

func (r *ProgressReporter) Report(ev checksum.ProgressEvent) {
	switch v := ev.(type) {

	case checksum.MostCurrentMergeHashFile:
		fmt.Printf("\n[MERGE] %q\n", v.Path)

	case checksum.MostCurrentFoundFile:
		r.checksumFilesFound++
		fmt.Printf("\r\033[2KMost current: %03d files (+ %03d ignored)",
			r.checksumFilesFound, r.checksumFilesIgnored)

	case checksum.MostCurrentIgnoredPath:
		r.checksumFilesIgnored++
		fmt.Printf("\n\033[2K[IGN  ] %q\n", v.Path)

	case checksum.DiscoverFilesFound:
		r.filesFound = v.Count
		fmt.Printf("\r\033[2KFound files: %03d (+ %03d ignored)",
			r.filesFound, r.filesIgnored)

	case checksum.DiscoverFilesIgnored:
		r.filesIgnored++

	case checksum.DiscoverFilesDone:
		fmt.Printf("\n\033[2KIncremental: Discovering done, found %d (+ %d ignored)\n",
			v.Found, v.Ignored)

	case checksum.PreRead:
		r.currentFile = v.Path
		fmt.Printf("\n\033[2K[READ ] %q\n", v.Path)

	case checksum.ReadProgress:
		if r.currentFile != "" {
			fmt.Printf("\r\033[2K[READ ] %-8s %8d / %8d bytes",
				fileNameOrEmpty(r.currentFile), v.Read, v.Total)
		} else {
			fmt.Printf("\r\033[2K[READ ] %8d / %8d bytes", v.Read, v.Total)
		}

	case checksum.FileMatch:
		fmt.Printf("\r\033[2K[OK   ] %q unchanged\n", v.Path)

	case checksum.FileUnchangedSkipped:
		fmt.Printf("\r\033[2K[SKIP ] %q (unchanged, skipped)\n", v.Path)

	case checksum.FileChanged:
		fmt.Printf("\r\033[2K[CHG  ] %q modified\n", v.Path)

	case checksum.FileChangedCorrupted:
		fmt.Printf("\r\033[2K[CORR ] %q corrupted\n", v.Path)

	case checksum.FileChangedOlder:
		fmt.Printf("\r\033[2K[OLD  ] %q local newer than hash\n", v.Path)

	case checksum.FileNew:
		fmt.Printf("\r\033[2K[NEW  ] %q\n", v.Path)

	case checksum.FileRemoved:
		fmt.Printf("\r\033[2K[DEL  ] %q\n", v.Path)

	case checksum.Finished:
		fmt.Println("\nDone.")

	case checksum.VerifyProgress:
		r.ReportVerify(&v)
	}
}

func (r *ProgressReporter) ReportVerify(p *checksum.VerifyProgress) {
	switch p.Stage {

	case checksum.VerifyPre:
		path := filepath.Join(p.Common.TreeRoot, p.Common.RelativePath)
		r.currentFile = path

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
		if r.currentFile != "" {
			percent := 0.0
			if p.Total > 0 {
				percent = float64(p.Done) / float64(p.Total) * 100.0
			}

			fmt.Printf(
				"\r\033[2K[HASH  ] %-30s %8d/%8d bytes (%5.1f%%)",
				filepath.Base(r.currentFile),
				p.Done,
				p.Total,
				percent,
			)
		} else {
			fmt.Printf(
				"\r\033[2K[HASH  ] %8d/%8d bytes",
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
}

func fileNameOrEmpty(p string) string {
	if p == "" {
		return ""
	}
	base := filepath.Base(p)
	if base == "." || base == string(filepath.Separator) {
		return ""
	}
	return base
}
