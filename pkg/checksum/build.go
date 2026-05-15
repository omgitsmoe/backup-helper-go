package checksum

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	"sort"
)

func buildMostCurrent(root string, options *Options, progress func()) (*HashCollection, error) {
	hashFiles, err := discoverHashFiles(root, options, progress)
	if err != nil {
		return nil, fmt.Errorf("failed to discover hash files: %w", err)
	}

	hashFilesSorted, err := sortPathsByAscendingMTime(hashFiles)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to sort hash files to build most current: %w", err)
	}

	filename := defaultHashFileName(root, "most_current", "most_current_")
	mostCurrent := NewHashCollection(filepath.Join(root, filename))

	for _, pathWithMTime := range hashFilesSorted {
		hashFile, err := NewHashCollectionFromDisk(pathWithMTime.Path)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to read hash file at '%q': %w", pathWithMTime.Path, err)
		}

		err = mostCurrent.Merge(hashFile)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to merge hash file at '%q' into most current: %w",
				pathWithMTime.Path, err)
		}
	}

	if options.MostCurrentFilterDeleted {
		toDelete := []string{}
		for p := range mostCurrent.pathToFile {
			if _, err := os.Stat(p); os.IsNotExist(err) {
				toDelete = append(toDelete, p)
			}
		}

		for _, p := range  toDelete {
			delete(mostCurrent.pathToFile, p)
		}
	}

	return mostCurrent, nil
}

type pathWithMTime struct {
	Path string
	MTime time.Time
}

func sortPathsByAscendingMTime(paths []string) ([]pathWithMTime, error) {
	pathsWithMTime := make([]pathWithMTime, 0, len(paths))
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to get mtime for file '%q': %w",
				path, err)
		}

		mtime := info.ModTime()

		pathsWithMTime = append(
			pathsWithMTime, pathWithMTime{ Path: path, MTime: mtime })
	}

	// sort.Slice func receives indices and should return true
	// if item at i is smaller than item at j (for asc order)
	sort.Slice(pathsWithMTime, func(i, j int) bool {
		return pathsWithMTime[i].MTime.Before(pathsWithMTime[j].MTime)
	})

	return pathsWithMTime, nil
}
