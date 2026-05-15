package checksum

import (
	"crypto"
	"fmt"
	"path/filepath"
	"time"
)

type Checker struct {
	root    string
	options Options
}

type Options struct {
	// Which hash algorithm to use for generating new hashes.
	HashType Hash

	// Whether to include files in the output, which did not change compared
	// to the previous latest available hash found.
	IncrementalIncludeUnchangedFiles bool

	// Whether to skip files when computing hashes if that files has the same
	// modification time as in the latest available hash found.
	IncrementalSkipUnchanged bool

	// If Some, periodically flushes the incremental hash collection
	// to disk upon the next modification after the specified time interval.
	IncrementalPeriodicWriteInterval time.Duration

	// Up to which depth should the root and its subdirectories be searched
	// for hash files (*.cshd, *.md5, *.sha512, etc.) to determine the
	// current state of hashes.
	// Zero means only files in the root directory will be considered.
	// One means at most one subdirectory will be allowed.
	// None means no depth limit.
	DiscoverHashFilesDepth int

	// Whether the most_current hash file should filter out all files that are
	// not found on disk at the time of generation.
	MostCurrentFilterDeleted bool

	// Allow/block list like matching for hash files which will be used
	// for building the most current state of hashes.
	// These hashes will be used when e.g. using the `incremental`
	// method.
	// NOTE: `*.cshd` only matches `foo.cshd`, not `bar/foo.cshd`, to match
	//       all `.cshd` files as well, specify `**/*.cshd`
	HashFilesMatcher Matcher

	// Allow/block list like matching for all files.
	// Affects all file discovery behaviour: which files get included
	// in an incremental hash file, which files are ignored when checking
	// for files that don't have checksums in `check_missing`, etc.
	// NOTE: `*.go` only matches `foo.go`, not `bar/foo.go`, to match
	//       all `.go` files as well, specify `**/*.go`
	AllFilesMatcher Matcher
}

func DefaultOptions() Options {
	return Options{
		HashType:                         Hash{crypto.SHA512},
		IncrementalIncludeUnchangedFiles: true,
		IncrementalSkipUnchanged:         false,
		IncrementalPeriodicWriteInterval: 0,
		DiscoverHashFilesDepth:            -1,
		MostCurrentFilterDeleted:         true,
	}
}

func NewChecker(root string) Checker {
	return Checker{
		root:    root,
		options: DefaultOptions(),
	}
}

func (c *Checker) Incremental(progress func()) {
	panic("Not implemented! TODO")
}

// Generate a [`HashCollection`], which only contains the hashes of
// files that do not have checksum in any matched hash file yet.
func (c *Checker) FillMissing(progress func()) {
	panic("Not implemented! TODO")
}

// Returns a result object containing all individual files that do not have checksums
// in `self.root` yet.
// If a directory has files and is completely missing it will be listed
// in `directories`.
// Note: The files of that directory will not appear in the file list.
func (c *Checker) CheckMissing(progress func()) {
	panic("Not implemented! TODO")
}

// Build a checksum file containing all the most current hashes found in all
// checksum files under [`ChecksumHelper::root`] if it isn't available yet.
// Then perform on action on it.
// If a most current collection is already available just the action
// the caller provides will be performed.
//
// The received `&HashCollection` can be written by using [`ChecksumHelper::write_collection`]
// or [`ChecksumHelper::write_into`].
//
//   - `progress`: Progress callback that receives a [`MostCurrentProgress`]
//     when progress is made.
//   - `action`: Closure that receives a reference to most current
//     [`HashCollection`].
func (c *Checker) BuildMostCurrent(progress func()) {
	panic("Not implemented! TODO")
}

// Verify all files matching predicated `include` in the [`HashCollection`]
//
//   - `include`: Predicate function which determines whether to include the
//     Path passed to it in verification. The path is relative
//     to the `file_tree.root()`.
//   - `progress`: Progress callback that receives a [`VerifyProgress`]
//     before and after processing the file.
func (c *Checker) Verify(progress func()) {
	panic("Not implemented! TODO")
}

// Verify all found checksum files found in the [`ChecksumHelper::root`].
//
// Verification results and progress in general is communicated via
// the [`progress`] callback.
//
//   - `include`: Predicate function which determines whether to include the
//     Path passed to it in verification. The path is relative
//     to the `file_tree.root()`.
//   - `progress`: Progress callback that receives a [`VerifyRootProgress`]
//     when building the most current checksum file
//     and on verification progress.
func (c *Checker) VerifyRoot(progress func()) {
	panic("Not implemented! TODO")
}

// Rebasing a [`HashCollection`] into a new `destination_directory` directory,
// changes its location to the `destination_directory` and removes all entries
// beyond the new location.
//
// If `destination_directory` is relative, it is interpreted relative to the collection root.
func (c *Checker) RebaseInto() {
	panic("Not implemented! TODO")
}

// Generates a default filename for a hash file, `fallback` is used
// if `root` does not have a filename.
func defaultHashFileName(root string, fallback string, infix string) string {
	prefix := filepath.Base(root)
	if prefix == "." || prefix == "" || prefix == "/" {
		prefix = fallback
	}

	t := time.Now()
	datetime := t.Format("2006-01-02T150405")

	return fmt.Sprintf("%s_%s%s.cshd", prefix, infix, datetime)
}
