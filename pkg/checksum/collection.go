package checksum

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"
)

type VerifyStage int

const (
	VerifyPre VerifyStage = iota
	VerifyDuring
	VerifyPost
)

var ErrFileExists = errors.New("file already exists")
var ErrMissingRootInMerge = errors.New("must have a root to supoort merging")
var ErrMergePardirBlocked = errors.New(
	"merge would result in references beyond the collection root")

type HashCollection struct {
	root       string
	name       string
	mtime      time.Time
	pathToFile map[string]*File
}

func newHashCollection(path string) *HashCollection {
	if !filepath.IsAbs(path) {
		panic("a HashCollection's path must be absolute, got: " + path)
	}

	clean := filepath.Clean(path)
	// NOTE: these return '.' on empty path
	// TODO only allow absolute paths?
	root := filepath.Dir(clean)
	filename := filepath.Base(clean)
	return &HashCollection{
		root:       root,
		name:       filename,
		pathToFile: make(map[string]*File, 0),
	}
}

func newHashCollectionFromDisk(path string) (*HashCollection, error) {
	if !filepath.IsAbs(path) {
		panic("a HashCollection's path must be absolute, got: " + path)
	}

	st, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file at '%q': %w", path, err)
	}

	mtime := st.ModTime()

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file at '%q': %w", path, err)
	}

	ext := filepath.Ext(path)
	var hc *HashCollection
	if ext == ".cshd" {
		hc, err = Parse(path, f)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to parse single-hash collection at '%q': %w", path, err)
		}
	} else {
		hashType, err := extensionToHashType(ext)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to determine hash type from extension: %w", err)
		}

		hc, err = ParseSingle(path, hashType, f)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to parse single-hash collection at '%q': %w", path, err)
		}
	}

	hc.mtime = mtime
	return hc, nil
}

func extensionToHashType(ext string) (Hash, error) {
	if len(ext) == 0 {
		return Hash{}, fmt.Errorf("empty extension")
	}

	id := ext[1:]
	return FromIdentifier(id)
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

func (c *HashCollection) Delete(path string) {
	delete(c.pathToFile, path)
}

func (c *HashCollection) ForEach(fn func(path string, file *File) bool) {
	for p, f := range c.pathToFile {
		if !fn(p, f) {
			return
		}
	}
}

func (c *HashCollection) Clear() {
	clear(c.pathToFile)
}

// Merges all entries in `other` into `self`. If there are conflicts:
// Keep the data from the collection with the more recent mtime.
// If both mtimes are zero then our entries are preferred.
func (c *HashCollection) Merge(other *HashCollection) error {
	if c.root == "" || c.root == "." {
		return fmt.Errorf("missing root on self: %w", ErrMissingRootInMerge)
	}
	if other.root == "" || other.root == "." {
		return fmt.Errorf("missing root on other: %w", ErrMissingRootInMerge)
	}

	rel, err := filepath.Rel(c.root, other.root)
	if err != nil {
		return fmt.Errorf(
			"failed to build relative path to other file in merge: %w",
			err)
	}
	rel = filepath.Clean(rel)
	// NOTE: going down the tree is allowed for merging, but not going up!
	//       otherwise, `c` would contain `..` paths after serializing!
	parts := filepath.SplitList(rel)
	if slices.Contains(parts, "..") {
		return fmt.Errorf(
			"merging not possible, relative paths would contain "+
				"pardir components: %w", ErrMergePardirBlocked)
	}

	keepOurs := c.mtime.After(other.mtime)
	if c.mtime.IsZero() {
		if other.mtime.IsZero() {
			keepOurs = true
		} else {
			keepOurs = false
		}
	}

	// TODO: decide on a file by file basis based on the stored mtime
	//       and only fall back to the collection mtime if no mtime
	//       is stored
	for p, f := range other.pathToFile {
		_, exists := c.pathToFile[p]

		if !exists || !keepOurs {
			c.pathToFile[p] = f
		}
	}

	return nil
}

type VerifyProgressCommon struct {
	TreeRoot            string
	RelativePath        string
	FileNumberProcessed uint64
	FileNumberTotal     uint64
	SizeProcessedBytes  uint64
	SizeTotalBytes      uint64
}

type VerifyProgress struct {
	Stage VerifyStage

	Common VerifyProgressCommon

	// Only used in During
	Done  uint64
	Total uint64

	// Only used in Post
	Result VerifyResult
}

func (c *HashCollection) SizeTotalBytes() int64 {
	total := int64(0)
	for _, f := range c.pathToFile {
		total += f.size
	}

	return total
}

// NOTE: needed to get deterministic iteration order for tests so
//
//	this variable will be overwritten during testing
var iterateMap func(map[string]*File, func(path string, file *File) error) error = defaultIterMap
var errStopIteration = errors.New("stop iteration")

func defaultIterMap(m map[string]*File, fn func(path string, file *File) error) error {
	for path, file := range m {
		err := fn(path, file)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *HashCollection) Verify(include func(path string) bool, progress func(VerifyProgress) bool) error {
	filesTotal := len(c.pathToFile)
	filesProcessed := 0
	sizeProcessedBytes := uint64(0)
	sizeTotalBytes := c.SizeTotalBytes()

	err := iterateMap(c.pathToFile, func(path string, file *File) error {
		relativePath, err := filepath.Rel(c.root, path)
		if err != nil {
			return fmt.Errorf("failed to build relative path for file: %w", err)
		}

		if include != nil && !include(relativePath) {
			filesProcessed += 1
			sizeProcessedBytes += uint64(file.size)
			return nil
		}

		common := VerifyProgressCommon{
			TreeRoot:            c.root,
			RelativePath:        relativePath,
			FileNumberProcessed: uint64(filesProcessed),
			FileNumberTotal:     uint64(filesTotal),
			SizeProcessedBytes:  sizeProcessedBytes,
			SizeTotalBytes:      uint64(sizeTotalBytes),
		}
		if progress != nil && !progress(VerifyProgress{
			Stage:  VerifyPre,
			Common: common,
		}) {
			return errStopIteration
		}

		result, err := file.Verify(func(done, total uint64) {
			if progress != nil {
				progress(
					VerifyProgress{
						Stage:  VerifyDuring,
						Common: common,
						Done:   done,
						Total:  total,
					},
				)
			}
		})

		// NOTE: Verify returns the error when result is VerifyFileMissing
		//       We ignore it if the file actually doesn't exist and
		//       only error on permission problems etc.
		if err != nil && result != VerifyFileMissing && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf(
				"failed to verify file at '%q': %w", file.path, err)
		}

		filesProcessed += 1
		common.FileNumberProcessed = uint64(filesProcessed)
		sizeProcessedBytes += uint64(file.size)
		common.SizeProcessedBytes = sizeProcessedBytes

		if progress != nil && !progress(
			VerifyProgress{
				Stage:  VerifyPost,
				Common: common,
				Result: result,
			},
		) {
			return errStopIteration
		}

		return nil
	})

	if err == errStopIteration {
		return nil
	}

	return err
}
