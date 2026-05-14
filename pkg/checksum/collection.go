package checksum

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	"errors"
)

var ErrFileExists = errors.New("file already exists")

type HashCollection struct {
	root       string
	name       string
	mtime      time.Time
	pathToFile map[string]*File
}

func NewHashCollection(path string) HashCollection {
	clean := filepath.Clean(path)
	// NOTE: these return '.' on empty path
	// TODO only allow absolute paths?
	root := filepath.Dir(clean)
	filename := filepath.Base(clean)
	return HashCollection{
		root:       root,
		name:       filename,
		pathToFile: make(map[string]*File, 0),
	}
}

func (c *HashCollection) Path() (string, error) {
	if len(c.root) == 0 || len(c.name) == 0 || c.root == "." || c.name == "." {
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

func (c *HashCollection) Root() string {
	return c.root
}

func (c *HashCollection) Name() string {
	return c.name
}

func (c *HashCollection) SetName(name string) {
	c.name = name
}

func (c *HashCollection) MTime() time.Time {
	return c.mtime
}

func (c *HashCollection) Get(path string) (*File, bool) {
	path = filepath.Clean(path)
	f, ok := c.pathToFile[path]
	return f, ok
}

func (c *HashCollection) Set(path string, file *File) {
	path = filepath.Clean(path)
	c.pathToFile[path] = file
}

func (c *HashCollection) Insert(file *File) error {
	if _, exists := c.pathToFile[file.path]; exists {
		return fmt.Errorf("%w: '%v'", ErrFileExists, file.path)
	}

	c.pathToFile[file.path] = file
	return nil
}

func (c *HashCollection) ForEach(fn func (path string, file *File) bool) {
	for p, f := range c.pathToFile {
		if !fn(p, f) {
			return
		}
	}
}

func (c *HashCollection) Clear() {
	clear(c.pathToFile)
}
