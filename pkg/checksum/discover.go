package checksum

import (
	"fmt"
	"errors"
	"io/fs"
	"path/filepath"
)

var ErrFiltered = errors.New("filtered")

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

		if d.IsDir() {
			// NOTE can't use match/allowed, since an intermediate dir
			//      won't match, e.g. *.go so we only should check for blocked
			if matcher.IsBlocked(relative) {
				fn(path, d, ErrFiltered)
				return fs.SkipDir
			}
		} else {
			if !matcher.Match(relative) {
				fn(path, d, ErrFiltered)
				return nil
			}
		}

		return fn(path, d, err)
	})
}
