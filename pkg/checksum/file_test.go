package checksum

import (
	"crypto"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
	"strings"
)

func TestHashFile(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "test.txt")
	content := []byte("hello world")

	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	tests := []struct {
		hash Hash
		want string
	}{
		{Hash{crypto.MD5}, "5eb63bbbe01eeed093cb22bb8f5acdc3"},
		{Hash{crypto.SHA512}, "309ecc489c12d6eb4cc40f50c902f2b4d0ed77ee511a7c7a9bcd3ca86d4cd86f989dd35bc5ff499670da34255b45b0cfd830e81f605dcf7dc5542e93ae9cd76f"},
	}

	for _, tt := range tests {
		t.Run(tt.hash.String(), func(t *testing.T) {
			sum, err := HashFile(path, tt.hash)
			if err != nil {
				t.Fatal(err)
			}

			got := hex.EncodeToString(sum)
			if got != tt.want {
				t.Fatalf("hash mismatch\n got %s\nwant %s", got, tt.want)
			}
		})
	}
}

func TestHashFileDoesNotExist(t *testing.T) {
	h, err := HashFile("foobarbazxer42069", Hash{crypto.SHA512})

	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}

	if h != nil {
		t.Fatalf("expected nil hash, got %v", h)
	}
}

func TestHashFileHashNotAvailable(t *testing.T) {
	// WARN: assumes crypt/md4 is not loaded/imported in this package!!!
	unimportedHash := Hash{crypto.MD4}
	h, err := HashFile("foobarbazxer42069", unimportedHash)

	if !errors.Is(err, ErrHashNotAvailable) {
		t.Fatalf("expected ErrHashNotAvailable got %v", err)
	}

	var e *HashNotAvailableError
	if errors.As(err, &e) {
		if e.Hash != unimportedHash {
			t.Fatalf("expected err.Hash == MD4, got %v", e.Hash)
		}
		
		if !strings.Contains(e.Error(), "MD4") {
			t.Fatalf("expected err.Error to contain 'MD4'")
		}
	} else {
		t.Fatalf("expected HashNotAvailableError, got %v", err)
	}

	if h != nil {
		t.Fatalf("expected nil hash, got %v", h)
	}
}

func TestUpdateHash(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "test.txt")
	content := []byte("hello world")

	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	tests := []struct {
		hash Hash
		want string
	}{
		{Hash{crypto.MD5}, "5eb63bbbe01eeed093cb22bb8f5acdc3"},
		{Hash{crypto.SHA512}, "309ecc489c12d6eb4cc40f50c902f2b4d0ed77ee511a7c7a9bcd3ca86d4cd86f989dd35bc5ff499670da34255b45b0cfd830e81f605dcf7dc5542e93ae9cd76f"},
	}

	for _, tt := range tests {
		t.Run(tt.hash.String(), func(t *testing.T) {
			file := NewFile(path, tt.hash)
			err := file.UpdateHash()
			if err != nil {
				t.Fatal(err)
			}

			got := hex.EncodeToString(file.hash)
			if got != tt.want {
				t.Fatalf("hash mismatch\n got %s\nwant %s", got, tt.want)
			}
		})
	}
}

func TestUpdateHashFileNotFound(t *testing.T) {
	f := NewFile("foobarbazxer42069", Hash{crypto.MD5})
	hash := []byte{123, 55, 33}
	f.hash = hash

	err := f.UpdateHash()
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}

	if !reflect.DeepEqual(f.hash, hash) {
		t.Fatalf("hash was overwritten")
	}
}

func TestUpdateMetadataSuccess(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "testfile.txt")
	content := []byte("hello world")

	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	f := NewFile(path, Hash{crypto.MD5})

	if err := f.UpdateMetadata(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.size != int64(len(content)) {
		t.Fatalf("expected size %d, got %d", len(content), f.size)
	}

	if time.Since(f.mtime) > time.Minute {
		t.Fatalf("mtime seems too old: %v", f.mtime)
	}
}

func TestUpdateMetadataNotFound(t *testing.T) {
	f := &File{
		path: "this/path/does/not/exist/definitely_12345",
	}

	err := f.UpdateMetadata()

	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist error, got %v", err)
	}
}

func TestHashFileNewCleansPath(t *testing.T) {
	tests := []struct { path string; expectedPath string } {
		{ path: "foo/.//./bar", expectedPath: filepath.Join("foo", "bar") },
		{ path: "foo///../bar", expectedPath: "bar" },
	}

	for _, tt := range tests {
		f := NewFile(tt.path, Hash{crypto.MD5})
		assertEqual(t, f.path, tt.expectedPath)
	}
}
