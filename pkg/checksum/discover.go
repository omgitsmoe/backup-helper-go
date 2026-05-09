package checksum

import (
	"io/fs"
	"path/filepath"
)

func FilteredWalk(root string, fn fs.WalkDirFunc) {
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fn(path, d, err)
		}

		return nil
	})
}
