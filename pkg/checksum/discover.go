package checksum

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
)

var ErrFiltered = errors.New("filtered")
var ErrFilteredSpecialFile = errors.New("filtered non-regular file")

// NOTE: Skips symlinks to directories!
func FilteredWalk(root string, matcher Matcher, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fn(path, d, err)
		}

		// NOTE: matcher matches on the relative path
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("failed to build a relative path: %w", err)
		}

		// NOTE better handling for symlinks and other speical files:
		//      we will only visit regular files, but notify
		//      about skipped files!
		//
		//      options:
		//      - skip non-regular files
		//        - scorch: skips non-regular files
		//        - `find ./foo/ -type f -print0 | xargs -0 sha1sum`
		//          also skips non-regular files
		//      - follow the symlink for files, record error for faulty links
		//        - is confusing, since we don't follow links to directories
		//          and doing that would be a completely different rabbit hole
		//        - also most tools don't follow symlinks when copying by
		//          default, e.g. rsync BUT cp does follow BUT only
		//          in file, not directory-mode :/
		//      - hash the contents of a symlink
		//        - would lead to confusing results for links that point
		//          to the same path, but different contents depending
		//          on the environment
		//      - record the symlink itself as a special entry
		//        - same drawback as hashing the link contents
		if d.IsDir() {
			// NOTE can't use match/allowed, since an intermediate dir
			//      won't match, e.g. **/*.go so we only should check for blocked
			if matcher.IsBlocked(relative) {
				fn(path, d, ErrFiltered)
				return fs.SkipDir
			}
		} else if d.Type().IsRegular() {
			if !matcher.Match(relative) {
				fn(path, d, ErrFiltered)
				return nil
			}
		} else {
			fn(path, d, ErrFilteredSpecialFile)
			return nil
		}

		return fn(path, d, err)
	})
}

func discoverHashFiles(root string, options *Options, progress ProgressFunc) ([]string, error) {
	hashFiles := []string{}
	err := FilteredWalk(root, options.HashFilesMatcher, func(path string, d fs.DirEntry, err error) error {
		relativePath, relErr := filepath.Rel(root, path)
		if relErr != nil {
			panic("bug: iteration path must be relative to walkdir root")
		}

		if err == ErrFiltered {
			if progress != nil && (d.IsDir() || isHashFile(path)) {
				progress(MostCurrentIgnoredPath{Path: relativePath})
			}
			return nil
		}
		// TODO ErrFilteredSpecialFile + progress

		if err != nil {
			return err
		}

		if options.DiscoverHashFilesDepth != -1 && d.IsDir() {
			depth, err := directoryDepth(root, path)
			if err != nil {
				return fmt.Errorf("failed to determine directory depth: %w", err)
			}

			if depth > options.DiscoverHashFilesDepth {
				if progress != nil {
					progress(MostCurrentIgnoredPath{Path: relativePath})
				}
				return fs.SkipDir
			}
		}

		if !d.IsDir() && isHashFile(path) {
			hashFiles = append(hashFiles, path)
			if progress != nil {
				progress(MostCurrentFoundFile{Path: relativePath})
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed iteration while discovering hash files: %w", err)
	}

	return hashFiles, nil
}

var hashFileExtensions = []string{
	".cshd",
	".md5",
	".sha1",
	".sha224",
	".sha256",
	".sha384",
	".sha512",
	".sha3_224",
	".sha3_256",
	".sha3_384",
	".sha3_512",
	// ".shake_128",
	// ".shake_256",
	// ".blake2b",
	// ".blake2s",
}

func isHashFile(path string) bool {
	ext := filepath.Ext(path)
	return slices.Contains(hashFileExtensions, ext)
}

func directoryDepth(base string, target string) (int, error) {
	// NOTE: assumption is that p == filepath.Clean(p) holds for all paths
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return 0, fmt.Errorf(
			"failed to make path relative to iteration root, "+
				"this must succeed: %w", err)
	}

	if rel == "" || rel == "." {
		// base is depth 0!
		return 0, nil
	}

	// NOTE: base = depth 0, so add one!
	depth := strings.Count(rel, string(filepath.Separator)) + 1
	return depth, nil
}

func discoverFiles(root string, options *Options, progress ProgressFunc) ([]string, error) {
	paths := []string{}
	ignored := uint64(0)
	err := FilteredWalk(root, options.AllFilesMatcher, func(path string, d fs.DirEntry, err error) error {
		relativePath, relErr := filepath.Rel(root, path)
		if relErr != nil {
			panic("bug: iteration path must be relative to walkdir root")
		}

		if err == ErrFiltered {
			if progress != nil {
				ignored += 1
				progress(DiscoverFilesIgnored{Path: relativePath})
			}
			return nil
		}
		// TODO ErrFilteredSpecialFile + progress

		if err != nil {
			return err
		}

		if !d.IsDir() {
			paths = append(paths, path)
			if progress != nil {
				progress(DiscoverFilesFound{Count: uint64(len(paths))})
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to discover files for hashing: %w", err)
	}

	if progress != nil {
		progress(DiscoverFilesDone{
			Found:   uint64(len(paths)),
			Ignored: ignored,
		})
	}

	return paths, nil
}
