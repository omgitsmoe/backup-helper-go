package checksum

import (
	"crypto"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
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
			sum, err := HashFile(path, tt.hash, nil)
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
	h, err := HashFile("foobarbazxer42069", Hash{crypto.SHA512}, nil)

	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}

	if h != nil {
		t.Fatalf("expected nil hash, got %v", h)
	}
}

func TestHashFileHashNotAvailable(t *testing.T) {
	// WARN: assumes crypto/md4 is not loaded/imported in this package!!!
	unimportedHash := Hash{crypto.MD4}
	h, err := HashFile("foobarbazxer42069", unimportedHash, nil)

	if !errors.Is(err, ErrHashTypeNotAvailable) {
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
			progressExpected := []struct { done, total uint64 } {
				{ 11, 11 },
				{ 11, 11 },
			}
			progressReceived := []struct { done, total uint64 } {
			}
			progress := func(done, total uint64) {
				progressReceived = append(
					progressReceived,
					struct { done, total uint64 }{
						done: done, total: total, })
			}

			file := NewFile(path, tt.hash)
			err := file.UpdateHash(progress)
			if err != nil {
				t.Fatal(err)
			}

			got := hex.EncodeToString(file.hash)
			if got != tt.want {
				t.Fatalf("hash mismatch\n got %s\nwant %s", got, tt.want)
			}

			assertSliceEqual(t, progressReceived, progressExpected)
		})
	}
}

func TestUpdateHashFileNotFound(t *testing.T) {
	f := NewFile("foobarbazxer42069", Hash{crypto.MD5})
	hash := []byte{123, 55, 33}
	f.hash = hash

	err := f.UpdateHash(nil)
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

func TestVerify(t *testing.T) {
	tests := []struct {
		name string
		input *File
		fileContents []byte
		fileMtime time.Time
		expected VerifyResult
		wantErr bool
		errorKind error
	}{
		{
			name: "err: missing hash",
			input: &File{
				path:     filepath.Join("foo", "bar", "file.txt"),
				mtime:    time.Time{},
				size:     0,
				hashType: Hash{},
				hash:     []byte{},
			},
			fileContents: nil,
			fileMtime: time.Time{},
			expected: 0,
			wantErr: true,
			errorKind: ErrMissingHash,
		},
		{
			name: "mismatch: missing file",
			input: &File{
				path:     filepath.Join("foo", "bar", "file.txt"),
				mtime:    time.Time{},
				size:     0,
				hashType: Hash{crypto.MD5},
				hash:     []byte("deadbeef"),
			},
			fileContents: nil,
			fileMtime: time.Time{},
			expected: VerifyFileMissing,
			wantErr: true,
			errorKind: os.ErrNotExist,
		},
		{
			name: "mismatch: size",
			input: &File{
				path:     filepath.Join("foo", "bar", "file.txt"),
				mtime:    time.Time{},
				size:     5,
				hashType: Hash{crypto.MD5},
				hash:     []byte("deadbeef"),
			},
			fileContents: []byte("123456"),
			fileMtime: time.Time{},
			expected: VerifyMismatchSize,
			wantErr: false,
			errorKind: nil,
		},
		{
			name: "mismatch",
			input: &File{
				path:     filepath.Join("foo", "bar", "file.txt"),
				mtime:    time.Time{},
				size:     5,
				hashType: Hash{crypto.MD5},
				hash:     []byte("deadbeef"),
			},
			fileContents: []byte("12345"),
			fileMtime: time.Time{},
			expected: VerifyMismatch,
			wantErr: false,
			errorKind: nil,
		},
		{
			name: "mismatch: corrupted",
			input: &File{
				path:     filepath.Join("foo", "bar", "file.txt"),
				mtime:    time.Unix(1337, 1_330_000),
				size:     0,
				hashType: Hash{crypto.MD5},
				hash:     []byte("deadbeef"),
			},
			fileContents: []byte("corrupted"),
			fileMtime: time.Unix(1337, 1_330_000),
			expected: VerifyMismatchCorrupted,
			wantErr: false,
			errorKind: nil,
		},
		{
			name: "mismatch: outdated",
			input: &File{
				path:     filepath.Join("foo", "bar", "file.txt"),
				mtime:    time.Unix(1337, 0),
				size:     0,
				hashType: Hash{crypto.MD5},
				hash:     []byte("deadbeef"),
			},
			fileContents: []byte("corrupted"),
			fileMtime: time.Unix(1337, 1_330_000),
			expected: VerifyMismatchOutdatedHash,
			wantErr: false,
			errorKind: nil,
		},
		{
			name: "ok",
			input: &File{
				path:     filepath.Join("foo", "bar", "file.txt"),
				mtime:    time.Time{},
				size:     0,
				hashType: Hash{crypto.MD5},
				hash:     []byte{
					0x5e, 0xb6, 0x3b, 0xbb, 0xe0, 0x1e, 0xee, 0xd0,
					0x93, 0xcb, 0x22, 0xbb, 0x8f, 0x5a, 0xcd, 0xc3,
				},
			},
			fileContents: []byte("hello world"),
			fileMtime: time.Time{},
			expected: VerifyOK,
			wantErr: false,
			errorKind: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, tt.input.path)

			if tt.fileContents != nil {
				if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
					t.Fatalf("failed to create parent dirs for hash file: %v", err)
				}
				if err := os.WriteFile(path, tt.fileContents, 0644); err != nil {
					t.Fatalf("failed to write test hash file: %v", err)
				}

				if !tt.fileMtime.IsZero() {
					err := os.Chtimes(path, tt.fileMtime, tt.fileMtime)
					if err != nil {
						t.Fatalf("failed to write mtime of hash file: %v", err)
					}
				}
			}

			// patch actual path
			tt.input.path = path

			result, err := tt.input.Verify(nil)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}

				if tt.errorKind != nil {
					if !errors.Is(err, tt.errorKind) {
						t.Fatalf(
							"expected error of kind '%v', got '%v'",
							tt.errorKind, err,
						)
					}
				}
				return
			} else {
				assertNoErr(t, err)
			}

			assertEqual(t, result, tt.expected)
		})
	}
}

func TestVerifyReportsProgress(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "test.txt")

	if err := os.WriteFile(path, []byte("hello world"), 0644); err != nil {
		t.Fatalf("failed to write test hash file: %v", err)
	}

	progressExpected := []struct { done, total uint64 } {
		{ 11, 11 },
		{ 11, 11 },
	}
	progressReceived := []struct { done, total uint64 } {
	}
	progress := func(done, total uint64) {
		progressReceived = append(
			progressReceived,
			struct { done, total uint64 }{
				done: done, total: total, })
	}

	file :=  &File{
		path:     path,
		mtime:    time.Time{},
		size:     0,
		hashType: Hash{crypto.MD5},
		hash:     []byte{
			0x5e, 0xb6, 0x3b, 0xbb, 0xe0, 0x1e, 0xee, 0xd0,
			0x93, 0xcb, 0x22, 0xbb, 0x8f, 0x5a, 0xcd, 0xc3,
		},
	}

	result, err := file.Verify(progress)

	assertNoErr(t, err)
	assertEqual(t, result, VerifyOK)
	assertSliceEqual(t, progressReceived, progressExpected)
}
