package main

import (
    // "github.com/omgitsmoe/backup-helper-go/pkg/checksum"
    "github.com/omgitsmoe/backup-helper-go/internal/filetree"
)

func main() {
	filetree.FromDir("/mnt/wdata/recs/comps")
	// filetree.Print(&tree)
}
