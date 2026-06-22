package checksum

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"
)

func incremental(root string, mostCurrent *HashCollection, options *Options, progress ProgressFunc) (*HashCollection, error) {
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

	// TODO only create if IncrementalPeriodicWriteInterval>0
	f, err := os.Create(resultPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create incremental hash file: %w", err)
	}
	defer f.Close()

	lastFlush := time.Now()
	serializer := NewSerializer(f)

	for _, p := range allFiles {
		file := NewFile(p, options.HashType)
		relativePath, err := filepath.Rel(root, p)
		if err != nil {
			panic("bug: incremental file path must be relative to incremental root")
		}

		previous, hasPrevious := mostCurrent.Get(p)

		if progress != nil {
			progress(PreRead{Path: relativePath})
		}
		err = file.UpdateMetadata()
		if err != nil {
			return nil, fmt.Errorf(
				"failed to get file metadata during incremental hash file generation: %w",
				err)
		}

		if options.IncrementalSkipUnchanged && hasPrevious &&
			mtimeWithin(file.mtime, previous.mtime) {
			// we skip checking the hash on disk, since mtime is unchanged
			// and the user set incremental_skip_unchanged
			err := result.Insert(previous.Clone())
			if err != nil {
				return nil, fmt.Errorf(
					"failed adding file to incremental hash file: %w",
					err)
			}

			if progress != nil {
				progress(FileUnchangedSkipped{Path: relativePath})
			}

			continue
		}

		err = file.UpdateHash(func(done, total uint64) {
			if progress != nil {
				progress(ReadProgress{
					Read:  done,
					Total: total,
				})
			}
		})
		if err != nil {
			return nil, fmt.Errorf(
				"failed to compute hash for file at '%q': %w",
				p, err)
		}

		include := true
		if hasPrevious {
			include, err = incrementalInclude(
				&relativePath, &file, previous, options, progress)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to process hash file at '%q' during incremental generation: %w",
					p, err)
			}
		} else {
			if progress != nil {
				progress(FileNew{Path: relativePath})
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

	if progress != nil {
		seen := make(map[string]struct{}, len(allFiles))
		for _, p := range allFiles {
			seen[p] = struct{}{}
		}

		mostCurrent.ForEach(func(path string, _ *File) bool {
			if _, ok := seen[path]; !ok {
				relativePath, err := filepath.Rel(root, path)
				if err != nil {
					panic("bug: incremental file path must be relative to incremental root")
				}
				// file is missing on disk
				progress(FileRemoved{Path: relativePath})
			}

			return true
		})

		progress(Finished{})
	}
	return result, nil
}

func incrementalInclude(
	relativePath *string, onDisk *File, previous *File, options *Options,
	progress ProgressFunc) (bool, error) {
	hashToCompareToPrevious := onDisk.hash
	if onDisk.hashType != previous.hashType {
		var err error
		hashToCompareToPrevious, err = HashFile(
			onDisk.path, previous.hashType, func(done, total uint64) {
				if progress != nil {
					progress(ReadProgress{
						Read:  done,
						Total: total,
					})
				}
			})
		if err != nil {
			return false, fmt.Errorf(
				"failed to re-compute hash to compare against recorded hash type: %w",
				err)
		}
	}

	if slices.Equal(hashToCompareToPrevious, previous.hash) {
		if progress != nil {
			progress(FileMatch{Path: *relativePath})
		}
		return options.IncrementalIncludeUnchangedFiles, nil
	}

	if progress != nil {
		if onDisk.mtime.IsZero() || previous.mtime.IsZero() {
			progress(FileChanged{Path: *relativePath})
		} else if previous.mtime.After(onDisk.mtime) {
			progress(FileChanged{Path: *relativePath})
		} else if previous.mtime.Equal(onDisk.mtime) {
			progress(FileChangedCorrupted{Path: *relativePath})
		} else if previous.mtime.Before(onDisk.mtime) {
			progress(FileChangedOlder{Path: *relativePath})
		} else {
			panic("unreachable")
		}
	}

	return true, nil
}
