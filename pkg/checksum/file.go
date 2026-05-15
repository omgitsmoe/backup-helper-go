package checksum

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	// NOTE: these need to be imported such that they become hash.Available()
	_ "crypto/md5"
	_ "crypto/sha1"
	_ "crypto/sha256"
	_ "crypto/sha3"
	_ "crypto/sha512"
)

type VerifyResult int

const (
	VerifyOK VerifyResult = iota
	VerifyFileMissing
	VerifyMismatch
	VerifyMismatchSize
	VerifyMismatchCorrupted
	VerifyMismatchOutdatedHash
)

var ErrHashTypeNotAvailable = errors.New("hash type not available")
var ErrMissingHash = errors.New("missing hash")

type HashNotAvailableError struct {
	Hash Hash
}

func (e *HashNotAvailableError) Error() string {
	return fmt.Sprintf("hash not available: %s", e.Hash)
}

func (e *HashNotAvailableError) Is(target error) bool {
	return target == ErrHashTypeNotAvailable
}

type File struct {
	path string

	mtime time.Time
	size  int64

	hashType Hash
	hash     []byte
}

func NewFile(path string, hashType Hash) File {
	path = filepath.Clean(path)
	return File{
		path:     path,
		hashType: hashType,
	}
}

func FileFromDisk(path string) (File, error) {
	info, err := os.Stat(path)
	if err != nil {
		return File{}, fmt.Errorf("failed to stat file at '%q': %w", path, err)
	}

	return File{
		path:  path,
		mtime: info.ModTime(),
		size:  info.Size(),
	}, nil
}

func (f *File) UpdateMetadata() error {
	info, err := os.Stat(f.path)
	if err != nil {
		return fmt.Errorf("failed to stat file at '%q': %w", f.path, err)
	}

	f.size = info.Size()
	f.mtime = info.ModTime()

	return nil
}

func (f *File) UpdateHash(progress func(done, total uint64)) error {
	hash, err := HashFile(f.path, f.hashType, progress)
	if err != nil {
		return err
	}

	f.hash = hash

	return nil
}

type progressReader struct {
	r        io.Reader
	read     uint64
	total    uint64
	progress func(done, total uint64)
}

func (p *progressReader) Read(buf []byte) (int, error) {
	n, err := p.r.Read(buf)
	p.read += uint64(n)

	if p.progress != nil {
		p.progress(p.read, p.total)
	}

	return n, err
}

func HashFile(path string, hash Hash, progress func(done, total uint64)) ([]byte, error) {
	h := hash.Hash
	if !h.Available() {
		return nil, &HashNotAvailableError{hash}
	}

	hasher := h.New()
	f, err := os.Open(path)

	if err != nil {
		return nil, fmt.Errorf("failed to open file at '%s' for hashing: %w", path, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	var r io.Reader = f

	// wrap the reader to enable progress reporting
	if progress != nil {
		r = &progressReader{
			r:        f,
			total:    uint64(info.Size()),
			progress: progress,
		}
	}

	_, err = io.Copy(hasher, r)
	if err != nil {
		return nil, err
	}

	return hasher.Sum(nil), nil
}

func (f *File) Verify(progress func(done, total uint64)) (VerifyResult, error) {
	if len(f.hash) == 0 {
		return 0, ErrMissingHash
	}

	path := f.path

	meta, err := os.Stat(path)
	if err != nil {
		return VerifyFileMissing, err
	}

	if f.size != 0 {
		if meta.Size() != f.size {
			return VerifyMismatchSize, nil
		}
	}

	hashOnDisk, err := HashFile(path, f.hashType, progress)
	if err != nil {
		return 0, err
	}

	if bytes.Equal(hashOnDisk, f.hash) {
		return VerifyOK, nil
	}

	if f.mtime.IsZero() {
		return VerifyMismatch, nil
	}

	mtimeOnDisk := meta.ModTime()

	if mtimeOnDisk.Equal(f.mtime) {
		return VerifyMismatchCorrupted, nil
	}

	return VerifyMismatchOutdatedHash, nil
}
