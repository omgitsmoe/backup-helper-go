package checksum

import (
	"crypto"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPathMissingRootOrName(t *testing.T) {
	tests := []string{
		"",
		"root/",
		"file.txt",
	}

	for _, root := range tests {
		t.Run(root, func(t *testing.T) {
			c := NewHashCollection(root)

			p, err := c.Path()

			assertErr(t, err)

			if p != "" {
				t.Fatalf("expected empty path, got %q", p)
			}
		})
	}
}

func TestPath(t *testing.T) {
	expected := filepath.Join("foo", "bar", "baz.txt")
	c := NewHashCollection(expected)

	actual, err := c.Path()
	assertNoErr(t, err)

	if actual != expected {
		assertEqual(t, actual, expected)
	}
}

func TestNewHashCollectionNormalizesPath(t *testing.T) {
	expected := filepath.Join("foo", "bar", "baz.txt")
	c := NewHashCollection("foo///./bar//../bar/baz.txt")

	actual, err := c.Path()
	assertNoErr(t, err)

	if actual != expected {
		assertEqual(t, actual, expected)
	}
}

func TestUpdateMtimePathError(t *testing.T) {
	tests := []string{
		"",
		"root/",
		"file.txt",
		"this/path/does/not/exist123/surely.txt",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			c := NewHashCollection(path)
			expected := time.Unix(1337, 0)
			c.mtime = expected

			err := c.UpdateMtime()

			assertErr(t, err)
			assertEqual(t, c.mtime, expected)
		})
	}
}

func TestUpdateMtime(t *testing.T) {
	temp := t.TempDir()
	path := filepath.Join(temp, "file.cshd")
	if err := os.WriteFile(path, []byte("foo"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	c := NewHashCollection(path)
	err := c.UpdateMtime()

	assertNoErr(t, err)

	if time.Since(c.MTime()) > time.Second*3 {
		t.Fatalf("mtime seems too old: %v", c.mtime)
	}
}

func TestNewHashCollectionFromDisk(t *testing.T) {
	root := t.TempDir()

	tests := []struct{
		name string
		path string
		fileContents []byte
		expected *HashCollection
		wantErr bool
	}{
		{
			name: "file not found",
			path: filepath.Join(root, "does", "not", "exist.cshd"),
			fileContents: nil,
			expected: nil,
			wantErr: true,
		},
		{
			name: "unexpected extension",
			path: filepath.Join(root, "does", "not", "exist.foo"),
			fileContents: []byte("foo"),
			expected: nil,
			wantErr: true,
		},
		{
			name: "invalid cshd file",
			path: filepath.Join(root, "file.cshd"),
			fileContents: []byte("foo"),
			expected: nil,
			wantErr: true,
		},
		{
			name: "invalid single-hash file",
			path: filepath.Join(root, "file.md5"),
			fileContents: []byte("foo"),
			expected: nil,
			wantErr: true,
		},
		{
			name: "valid cshd file",
			path: filepath.Join(root, "file.cshd"),
			fileContents: []byte("# version 1\n1337.00133,42069,md5,deadbeef foo/bar.txt\n"),
			expected: &HashCollection{
				root: root,
				name: "file.cshd",
				pathToFile: map[string]*File{
					filepath.Join(root, "foo", "bar.txt"): {
						path: filepath.Join(root, "foo", "bar.txt"),
						mtime: time.Unix(1337, 1_330_000),
						size: 42069,
						hashType: Hash{crypto.MD5},
						hash: []byte{ 0xde, 0xad, 0xbe, 0xef },
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid single-hash file",
			path: filepath.Join(root, "file.sha512"),
			fileContents: []byte("deadbeef foo/bar.txt\n"),
			expected: &HashCollection{
				root: root,
				name: "file.sha512",
				pathToFile: map[string]*File{
					filepath.Join(root, "foo", "bar.txt"): {
						path: filepath.Join(root, "foo", "bar.txt"),
						mtime: time.Time{},
						size: 0,
						hashType: Hash{crypto.SHA512},
						hash: []byte{ 0xde, 0xad, 0xbe, 0xef },
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.fileContents != nil {
				if err := os.MkdirAll(filepath.Dir(tt.path), 0777); err != nil {
					t.Fatalf("failed to create parent dirs for hash file: %v", err)
				}
				if err := os.WriteFile(tt.path, tt.fileContents, 0644); err != nil {
					t.Fatalf("failed to write test hash file: %v", err)
				}
				if tt.expected != nil {
					s, err := os.Stat(tt.path)
					if err != nil {
						t.Fatalf("failed to stat hash file: %v", err)
					}
					tt.expected.mtime = s.ModTime()
				}
			}

			hc, err := NewHashCollectionFromDisk(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			} else {
				assertNoErr(t, err)
			}

			assertHashCollectionsEqual(t, hc, tt.expected)
		})
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name string
		collection *HashCollection
		other *HashCollection
		expected *HashCollection
		wantErr bool
		errorKind error
	}{
		{
			name:       "self missing root: empty",
			collection: &HashCollection{},
			other:      &HashCollection{
				root: filepath.Join("foo"),
			},
			expected:   &HashCollection{},
			wantErr:    true,
			errorKind:  ErrMissingRootInMerge,
		},
		{
			name:       "self missing root: curdir",
			collection: &HashCollection{
				root: ".",
			},
			other:      &HashCollection{
				root: filepath.Join("foo"),
			},
			expected:   &HashCollection{},
			wantErr:    true,
			errorKind:  ErrMissingRootInMerge,
		},
		{
			name:       "other missing root: curdir",
			collection: &HashCollection{
				root: filepath.Join("foo"),
			},
			other:      &HashCollection{
				root: ".",
			},
			expected:   &HashCollection{},
			wantErr:    true,
			errorKind:  ErrMissingRootInMerge,
		},
		{
			name:       "err pardir",
			collection: &HashCollection{
				root: filepath.Join("foo", "bar"),
			},
			other:      &HashCollection{
				root: filepath.Join("foo"),
			},
			expected:   &HashCollection{},
			wantErr:    true,
			errorKind:  ErrMergePardirBlocked,
		},
		{
			name:       "both zero mtimes: keep ours",
			collection: &HashCollection{
				root: filepath.Join("foo"),
				pathToFile: map[string]*File{
					filepath.Join("foo", "conflict.txt"): {
						path:     filepath.Join("foo", "conflict.txt"),
						mtime:    time.Unix(1337, 1_330_000),
						size:     42069,
						hashType: Hash{crypto.MD5},
						hash:     []byte{0xde, 0xad, 0xbe, 0xef},
					},
					filepath.Join("ours", "bar.txt"): {
						path:     filepath.Join("ours", "bar.txt"),
						mtime:    time.Unix(12345, 0),
						size:     5678,
						hashType: Hash{crypto.SHA512},
						hash:     []byte{0xab, 0xab, 0xab, 0xab},
					},
				},
			},
			other:      &HashCollection{
				root: filepath.Join("foo", "bar"),
				pathToFile: map[string]*File{
					filepath.Join("foo", "conflict.txt"): {
						path:     filepath.Join("foo", "conflict.txt"),
					},
					filepath.Join("other", "xer.txt"): {
						path:     filepath.Join("other", "xer.txt"),
						mtime:    time.Unix(898989, 111),
						size:     3344,
						hashType: Hash{crypto.SHA3_256},
						hash:     []byte{0xaa, 0xaa, 0xaa, 0xaa},
					},
				},
			},
			expected: &HashCollection{
				root: filepath.Join("foo"),
				pathToFile: map[string]*File{
					filepath.Join("foo", "conflict.txt"): {
						path:     filepath.Join("foo", "conflict.txt"),
						mtime:    time.Unix(1337, 1_330_000),
						size:     42069,
						hashType: Hash{crypto.MD5},
						hash:     []byte{0xde, 0xad, 0xbe, 0xef},
					},
					filepath.Join("ours", "bar.txt"): {
						path:     filepath.Join("ours", "bar.txt"),
						mtime:    time.Unix(12345, 0),
						size:     5678,
						hashType: Hash{crypto.SHA512},
						hash:     []byte{0xab, 0xab, 0xab, 0xab},
					},
					filepath.Join("other", "xer.txt"): {
						path:     filepath.Join("other", "xer.txt"),
						mtime:    time.Unix(898989, 111),
						size:     3344,
						hashType: Hash{crypto.SHA3_256},
						hash:     []byte{0xaa, 0xaa, 0xaa, 0xaa},
					},
				},
			},
			wantErr:    false,
			errorKind:  nil,
		},
		{
			name:       "other zero mtime: keep ours",
			collection: &HashCollection{
				root: filepath.Join("foo"),
				mtime: time.Unix(1337, 0),
				pathToFile: map[string]*File{
					filepath.Join("foo", "conflict.txt"): {
						path:     filepath.Join("foo", "conflict.txt"),
						mtime:    time.Unix(1337, 1_330_000),
						size:     42069,
						hashType: Hash{crypto.MD5},
						hash:     []byte{0xde, 0xad, 0xbe, 0xef},
					},
					filepath.Join("ours", "bar.txt"): {
						path:     filepath.Join("ours", "bar.txt"),
						mtime:    time.Unix(12345, 0),
						size:     5678,
						hashType: Hash{crypto.SHA512},
						hash:     []byte{0xab, 0xab, 0xab, 0xab},
					},
				},
			},
			other:      &HashCollection{
				root: filepath.Join("foo", "bar"),
				pathToFile: map[string]*File{
					filepath.Join("foo", "conflict.txt"): {
						path:     filepath.Join("foo", "conflict.txt"),
					},
					filepath.Join("other", "xer.txt"): {
						path:     filepath.Join("other", "xer.txt"),
						mtime:    time.Unix(898989, 111),
						size:     3344,
						hashType: Hash{crypto.SHA3_256},
						hash:     []byte{0xaa, 0xaa, 0xaa, 0xaa},
					},
				},
			},
			expected: &HashCollection{
				root: filepath.Join("foo"),
				mtime: time.Unix(1337, 0),
				pathToFile: map[string]*File{
					filepath.Join("foo", "conflict.txt"): {
						path:     filepath.Join("foo", "conflict.txt"),
						mtime:    time.Unix(1337, 1_330_000),
						size:     42069,
						hashType: Hash{crypto.MD5},
						hash:     []byte{0xde, 0xad, 0xbe, 0xef},
					},
					filepath.Join("ours", "bar.txt"): {
						path:     filepath.Join("ours", "bar.txt"),
						mtime:    time.Unix(12345, 0),
						size:     5678,
						hashType: Hash{crypto.SHA512},
						hash:     []byte{0xab, 0xab, 0xab, 0xab},
					},
					filepath.Join("other", "xer.txt"): {
						path:     filepath.Join("other", "xer.txt"),
						mtime:    time.Unix(898989, 111),
						size:     3344,
						hashType: Hash{crypto.SHA3_256},
						hash:     []byte{0xaa, 0xaa, 0xaa, 0xaa},
					},
				},
			},
			wantErr:    false,
			errorKind:  nil,
		},
		{
			name:       "other older: keep ours",
			collection: &HashCollection{
				root: filepath.Join("foo"),
				mtime: time.Unix(1337, 0),
				pathToFile: map[string]*File{
					filepath.Join("foo", "conflict.txt"): {
						path:     filepath.Join("foo", "conflict.txt"),
						mtime:    time.Unix(1337, 1_330_000),
						size:     42069,
						hashType: Hash{crypto.MD5},
						hash:     []byte{0xde, 0xad, 0xbe, 0xef},
					},
					filepath.Join("ours", "bar.txt"): {
						path:     filepath.Join("ours", "bar.txt"),
						mtime:    time.Unix(12345, 0),
						size:     5678,
						hashType: Hash{crypto.SHA512},
						hash:     []byte{0xab, 0xab, 0xab, 0xab},
					},
				},
			},
			other:      &HashCollection{
				root: filepath.Join("foo", "bar"),
				mtime: time.Unix(1111, 0),
				pathToFile: map[string]*File{
					filepath.Join("foo", "conflict.txt"): {
						path:     filepath.Join("foo", "conflict.txt"),
					},
					filepath.Join("other", "xer.txt"): {
						path:     filepath.Join("other", "xer.txt"),
						mtime:    time.Unix(898989, 111),
						size:     3344,
						hashType: Hash{crypto.SHA3_256},
						hash:     []byte{0xaa, 0xaa, 0xaa, 0xaa},
					},
				},
			},
			expected: &HashCollection{
				root: filepath.Join("foo"),
				mtime: time.Unix(1337, 0),
				pathToFile: map[string]*File{
					filepath.Join("foo", "conflict.txt"): {
						path:     filepath.Join("foo", "conflict.txt"),
						mtime:    time.Unix(1337, 1_330_000),
						size:     42069,
						hashType: Hash{crypto.MD5},
						hash:     []byte{0xde, 0xad, 0xbe, 0xef},
					},
					filepath.Join("ours", "bar.txt"): {
						path:     filepath.Join("ours", "bar.txt"),
						mtime:    time.Unix(12345, 0),
						size:     5678,
						hashType: Hash{crypto.SHA512},
						hash:     []byte{0xab, 0xab, 0xab, 0xab},
					},
					filepath.Join("other", "xer.txt"): {
						path:     filepath.Join("other", "xer.txt"),
						mtime:    time.Unix(898989, 111),
						size:     3344,
						hashType: Hash{crypto.SHA3_256},
						hash:     []byte{0xaa, 0xaa, 0xaa, 0xaa},
					},
				},
			},
			wantErr:    false,
			errorKind:  nil,
		},
		{
			name:       "self zero mtime: keep other",
			collection: &HashCollection{
				root: filepath.Join("foo"),
				pathToFile: map[string]*File{
					filepath.Join("foo", "conflict.txt"): {
						path:     filepath.Join("foo", "conflict.txt"),
						mtime:    time.Unix(1337, 1_330_000),
						size:     42069,
						hashType: Hash{crypto.MD5},
						hash:     []byte{0xde, 0xad, 0xbe, 0xef},
					},
					filepath.Join("ours", "bar.txt"): {
						path:     filepath.Join("ours", "bar.txt"),
						mtime:    time.Unix(12345, 0),
						size:     5678,
						hashType: Hash{crypto.SHA512},
						hash:     []byte{0xab, 0xab, 0xab, 0xab},
					},
				},
			},
			other:      &HashCollection{
				root: filepath.Join("foo", "bar"),
				mtime: time.Unix(1337, 0),
				pathToFile: map[string]*File{
					filepath.Join("foo", "conflict.txt"): {
						path:     filepath.Join("foo", "conflict.txt"),
					},
					filepath.Join("other", "xer.txt"): {
						path:     filepath.Join("other", "xer.txt"),
						mtime:    time.Unix(898989, 111),
						size:     3344,
						hashType: Hash{crypto.SHA3_256},
						hash:     []byte{0xaa, 0xaa, 0xaa, 0xaa},
					},
				},
			},
			expected: &HashCollection{
				root: filepath.Join("foo"),
				pathToFile: map[string]*File{
					filepath.Join("foo", "conflict.txt"): {
						path:     filepath.Join("foo", "conflict.txt"),
					},
					filepath.Join("ours", "bar.txt"): {
						path:     filepath.Join("ours", "bar.txt"),
						mtime:    time.Unix(12345, 0),
						size:     5678,
						hashType: Hash{crypto.SHA512},
						hash:     []byte{0xab, 0xab, 0xab, 0xab},
					},
					filepath.Join("other", "xer.txt"): {
						path:     filepath.Join("other", "xer.txt"),
						mtime:    time.Unix(898989, 111),
						size:     3344,
						hashType: Hash{crypto.SHA3_256},
						hash:     []byte{0xaa, 0xaa, 0xaa, 0xaa},
					},
				},
			},
			wantErr:    false,
			errorKind:  nil,
		},
		{
			name:       "self older: keep other",
			collection: &HashCollection{
				root: filepath.Join("foo"),
				mtime: time.Unix(123, 0),
				pathToFile: map[string]*File{
					filepath.Join("foo", "conflict.txt"): {
						path:     filepath.Join("foo", "conflict.txt"),
						mtime:    time.Unix(1337, 1_330_000),
						size:     42069,
						hashType: Hash{crypto.MD5},
						hash:     []byte{0xde, 0xad, 0xbe, 0xef},
					},
					filepath.Join("ours", "bar.txt"): {
						path:     filepath.Join("ours", "bar.txt"),
						mtime:    time.Unix(12345, 0),
						size:     5678,
						hashType: Hash{crypto.SHA512},
						hash:     []byte{0xab, 0xab, 0xab, 0xab},
					},
				},
			},
			other:      &HashCollection{
				root: filepath.Join("foo", "bar"),
				mtime: time.Unix(1111, 0),
				pathToFile: map[string]*File{
					filepath.Join("foo", "conflict.txt"): {
						path:     filepath.Join("foo", "conflict.txt"),
					},
					filepath.Join("other", "xer.txt"): {
						path:     filepath.Join("other", "xer.txt"),
						mtime:    time.Unix(898989, 111),
						size:     3344,
						hashType: Hash{crypto.SHA3_256},
						hash:     []byte{0xaa, 0xaa, 0xaa, 0xaa},
					},
				},
			},
			expected: &HashCollection{
				root: filepath.Join("foo"),
				mtime: time.Unix(123, 0),
				pathToFile: map[string]*File{
					filepath.Join("foo", "conflict.txt"): {
						path:     filepath.Join("foo", "conflict.txt"),
					},
					filepath.Join("ours", "bar.txt"): {
						path:     filepath.Join("ours", "bar.txt"),
						mtime:    time.Unix(12345, 0),
						size:     5678,
						hashType: Hash{crypto.SHA512},
						hash:     []byte{0xab, 0xab, 0xab, 0xab},
					},
					filepath.Join("other", "xer.txt"): {
						path:     filepath.Join("other", "xer.txt"),
						mtime:    time.Unix(898989, 111),
						size:     3344,
						hashType: Hash{crypto.SHA3_256},
						hash:     []byte{0xaa, 0xaa, 0xaa, 0xaa},
					},
				},
			},
			wantErr:    false,
			errorKind:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.collection.Merge(tt.other)

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

			assertHashCollectionsEqual(t, tt.collection, tt.expected)
		})
	}
}
