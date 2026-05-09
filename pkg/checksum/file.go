package checksum

import (
	"fmt"
	"io"
	"os"
	"time"
	// NOTE: these need to be imported such that they become hash.Available()
	_ "crypto/md5"
	_ "crypto/sha1"
	_ "crypto/sha256"
	_ "crypto/sha3"
	_ "crypto/sha512"
)

type File struct {
	path string

	mtime time.Time
	size  int64

	hashType Hash
	hash     []byte
}

func NewFile(path string, hash Hash) File {
	return File{
		path:     path,
		hashType: hash,
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

func (f *File) UpdateHash() error {
	hash, err := HashFile(f.path, f.hashType)
	if err != nil {
		return err
	}

	f.hash = hash

	return nil
}

func HashFile(path string, hash Hash) ([]byte, error) {
	h := hash.Hash
	if !h.Available() {
		return nil, fmt.Errorf("hash type '%s' not available!", h.String())
	}

	hasher := h.New()
	f, err := os.Open(path)

	if err != nil {
		return nil, fmt.Errorf("failed to open file at '%s' for hashing: %w", path, err)
	}
	defer f.Close()

	_, err = io.Copy(hasher, f)
	if err != nil {
		return nil, err
	}

	return hasher.Sum(nil), nil
}
