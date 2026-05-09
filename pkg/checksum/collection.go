package checksum

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type HashCollection struct {
	root       string
	name       string
	mtime      time.Time
	pathToFile map[string]File
}

func NewHashCollection(path string) HashCollection {
	root, filename := filepath.Split(path)
	return HashCollection{
		root:       root,
		name:       filename,
		pathToFile: make(map[string]File, 0),
	}
}

func (c *HashCollection) Path() (string, error) {
	if len(c.root) == 0 || len(c.name) == 0 {
		return "", fmt.Errorf("collection must have a root and name set")
	}

	return filepath.Join(c.root, c.name), nil
}

func (c *HashCollection) UpdateMtime() error {
	path, err := c.Path()
	if err != nil {
		return err
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file at '%q': %w", path, err)
	}

	c.mtime = info.ModTime()

	return nil
}

func (c *HashCollection) Mtime() time.Time {
	return c.mtime
}
