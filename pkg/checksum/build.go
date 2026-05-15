package checksum

import (
	"fmt"
	"path/filepath"
)

func buildMostCurrent(root string, options *Options, progress func()) (*HashCollection, error) {
	hashFiles, err := discoverHashFiles(root, options, progress)
	if err != nil {
		return nil, fmt.Errorf("failed to discover hash files: %w", err)
	}

	filename := defaultHashFileName(root, "most_current", "most_current")
	mostCurrent := NewHashCollection(filepath.Join(root, filename))

	for _, hashFilePath := range hashFiles {
		hashFile, err := NewHashCollectionFromDisk(hashFilePath)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to read hash file at '%q': %w", hashFilePath, err)
		}
		_ = hashFile
		panic("TODO")
	}

	return mostCurrent, nil
}
