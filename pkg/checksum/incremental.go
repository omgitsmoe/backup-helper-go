package checksum

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"
)

func incremental(root string, mostCurrent *HashCollection, options *Options, progress func()) (*HashCollection, error) {
	allFiles, err := discoverFiles(root, options, progress)
	if err != nil {
		return nil, fmt.Errorf("failed to discover files for hashing: %w", err)
	}

	filename := defaultHashFileName(root, "incremental", "")
	result := newHashCollection(filepath.Join(root, filename))

	resultPath, err := result.Path()
	if err != nil {
		// unreachable since we assigned with path above
		panic("bug: unreachable")
	}

	f, err := os.Create(resultPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create incremental hash file: %w", err)
	}
	defer f.Close()

	lastFlush := time.Now()
	serializer := NewSerializer(f)

	for _, p := range allFiles {
		file := NewFile(p, options.HashType)
		err := file.UpdateMetadata()
		if err != nil {
			return nil, fmt.Errorf(
				"failed to get file metadata during incremental hash file generation: %w",
				err)
		}

		previous, hasPrevious := mostCurrent.Get(p)

		if options.IncrementalSkipUnchanged && hasPrevious &&
			file.mtime.Equal(previous.mtime) {
			// we skip checking the hash on disk, since mtime is unchanged
			// and the user set incremental_skip_unchanged
			err := result.Insert(previous.Clone())
			if err != nil {
				return nil, fmt.Errorf(
					"failed adding file to incremental hash file: %w",
					err)
			}

			continue
		}

		// TODO progress
		err = file.UpdateHash(nil)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to compute hash for file at '%q': %w",
				p, err)
		}

		include := true
		if hasPrevious {
			include, err = incrementalInclude(&file, previous, options, progress)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to process hash file at '%q' during incremental generation: %w",
					p, err)
			}
		}

		if include {
			err := result.Insert(&file)
			if err != nil {
				return nil, fmt.Errorf(
					"failed adding file to incremental hash file: %w",
					err)
			}

			if options.IncrementalPeriodicWriteInterval != 0 &&
				time.Since(lastFlush) > options.IncrementalPeriodicWriteInterval {
				serializer.Flush(result)
			}
		}
	}

	if options.IncrementalPeriodicWriteInterval != 0 {
		// flush remaining entries
		serializer.Flush(result)
	}

	// TODO missing files

	return result, nil
}

func incrementalInclude(onDisk *File, previous *File, options *Options, progress func()) (bool, error) {
	hashToCompareToPrevious := onDisk.hash
	if onDisk.hashType != previous.hashType {
		// TODO progress
		var err error
		hashToCompareToPrevious, err = HashFile(onDisk.path, previous.hashType, nil)
		if err != nil {
			return false, fmt.Errorf(
				"failed to re-compute hash to compare against recorded hash type: %w",
				err)
		}
	}

	if slices.Equal(hashToCompareToPrevious, previous.hash) {
		return options.IncrementalIncludeUnchangedFiles, nil
	}

	// TODO progress with changed/corrupted etc.

	return true, nil
}
