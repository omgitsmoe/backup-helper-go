package checksum

import (
	"crypto"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

func TestPathMissingRootOrName(t *testing.T) {
	tests := []struct {
		root string
		name string
	}{
		{root: "", name: ""},
		{root: ".", name: ""},
		{root: "", name: ""},
		{root: "", name: "."},
		{root: "root/", name: ""},
		{root: ".", name: "file.txt"},
	}

	for _, tt := range tests {
		t.Run(filepath.Join(tt.root, tt.name), func(t *testing.T) {
			c := HashCollection{
				root: tt.root,
				name: tt.name,
			}

			p, err := c.Path()

			assertErr(t, err)

			if p != "" {
				t.Fatalf("expected empty path, got %q", p)
			}
		})
	}
}

func TestPath(t *testing.T) {
	expected := makeAbsOrFail(t, filepath.Join("foo", "bar", "baz.txt"))
	c := newHashCollection(expected)

	actual, err := c.Path()
	assertNoErr(t, err)

	if actual != expected {
		assertEqual(t, actual, expected)
	}
}

func TestNewHashCollectionNormalizesPath(t *testing.T) {
	expected := makeAbsOrFail(t, filepath.Join("foo", "bar", "baz.txt"))
	c := newHashCollection(makeAbsOrFail(t, "foo///./bar//../bar/baz.txt"))

	actual, err := c.Path()
	assertNoErr(t, err)

	if actual != expected {
		assertEqual(t, actual, expected)
	}
}

func TestNewHashCollectionPanicsOnRelativePath(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic, but did not panic")
		}
	}()

	newHashCollection(filepath.Join("relative", "path")) // should panic
}

func TestNewHashCollectionFromDiskPanicsOnRelativePath(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic, but did not panic")
		}
	}()

	newHashCollectionFromDisk(filepath.Join("relative", "path")) // should panic
}

func TestUpdateMtimePathError(t *testing.T) {
	tests := []struct {
		root string
		name string
	}{
		{root: "", name: ""},
		{root: ".", name: ""},
		{root: "", name: ""},
		{root: "", name: "."},
		{root: "root/", name: ""},
		{root: ".", name: "file.txt"},
		{root: "this/path/does/not/exist123", name: "surely.txt"},
	}

	for _, tt := range tests {
		t.Run(filepath.Join(tt.root, tt.name), func(t *testing.T) {
			c := HashCollection{
				root: tt.root,
				name: tt.name,
			}
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

	c := newHashCollection(path)
	err := c.UpdateMtime()

	assertNoErr(t, err)

	if time.Since(c.MTime()) > time.Second*3 {
		t.Fatalf("mtime seems too old: %v", c.mtime)
	}
}

func TestNewHashCollectionFromDisk(t *testing.T) {
	root := t.TempDir()

	tests := []struct {
		name         string
		path         string
		fileContents []byte
		expected     *HashCollection
		wantErr      bool
	}{
		{
			name:         "file not found",
			path:         filepath.Join(root, "does", "not", "exist.cshd"),
			fileContents: nil,
			expected:     nil,
			wantErr:      true,
		},
		{
			name:         "unexpected extension",
			path:         filepath.Join(root, "does", "not", "exist.foo"),
			fileContents: []byte("foo"),
			expected:     nil,
			wantErr:      true,
		},
		{
			name:         "invalid cshd file",
			path:         filepath.Join(root, "file.cshd"),
			fileContents: []byte("foo"),
			expected:     nil,
			wantErr:      true,
		},
		{
			name:         "invalid single-hash file",
			path:         filepath.Join(root, "file.md5"),
			fileContents: []byte("foo"),
			expected:     nil,
			wantErr:      true,
		},
		{
			name:         "valid cshd file",
			path:         filepath.Join(root, "file.cshd"),
			fileContents: []byte("# version 1\n1337.00133,42069,md5,deadbeef foo/bar.txt\n"),
			expected: &HashCollection{
				root: root,
				name: "file.cshd",
				pathToFile: map[string]*File{
					filepath.Join(root, "foo", "bar.txt"): {
						path:     filepath.Join(root, "foo", "bar.txt"),
						mtime:    time.Unix(1337, 1_330_000),
						size:     42069,
						hashType: Hash{crypto.MD5},
						hash:     []byte{0xde, 0xad, 0xbe, 0xef},
					},
				},
			},
			wantErr: false,
		},
		{
			name:         "valid single-hash file",
			path:         filepath.Join(root, "file.sha512"),
			fileContents: []byte("deadbeef foo/bar.txt\n"),
			expected: &HashCollection{
				root: root,
				name: "file.sha512",
				pathToFile: map[string]*File{
					filepath.Join(root, "foo", "bar.txt"): {
						path:     filepath.Join(root, "foo", "bar.txt"),
						mtime:    time.Time{},
						size:     0,
						hashType: Hash{crypto.SHA512},
						hash:     []byte{0xde, 0xad, 0xbe, 0xef},
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

			hc, err := newHashCollectionFromDisk(tt.path)
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
		name       string
		collection *HashCollection
		other      *HashCollection
		expected   *HashCollection
		wantErr    bool
		errorKind  error
	}{
		{
			name:       "self missing root: empty",
			collection: &HashCollection{},
			other: &HashCollection{
				root: filepath.Join("foo"),
			},
			expected:  &HashCollection{},
			wantErr:   true,
			errorKind: ErrMissingRootInMerge,
		},
		{
			name: "self missing root: curdir",
			collection: &HashCollection{
				root: ".",
			},
			other: &HashCollection{
				root: filepath.Join("foo"),
			},
			expected:  &HashCollection{},
			wantErr:   true,
			errorKind: ErrMissingRootInMerge,
		},
		{
			name: "other missing root: curdir",
			collection: &HashCollection{
				root: filepath.Join("foo"),
			},
			other: &HashCollection{
				root: ".",
			},
			expected:  &HashCollection{},
			wantErr:   true,
			errorKind: ErrMissingRootInMerge,
		},
		{
			name: "err pardir",
			collection: &HashCollection{
				root: filepath.Join("foo", "bar"),
			},
			other: &HashCollection{
				root: filepath.Join("foo"),
			},
			expected:  &HashCollection{},
			wantErr:   true,
			errorKind: ErrMergePardirBlocked,
		},
		{
			name: "both zero mtimes: keep ours",
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
			other: &HashCollection{
				root: filepath.Join("foo", "bar"),
				pathToFile: map[string]*File{
					filepath.Join("foo", "conflict.txt"): {
						path: filepath.Join("foo", "conflict.txt"),
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
			wantErr:   false,
			errorKind: nil,
		},
		{
			name: "other zero mtime: keep ours",
			collection: &HashCollection{
				root:  filepath.Join("foo"),
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
			other: &HashCollection{
				root: filepath.Join("foo", "bar"),
				pathToFile: map[string]*File{
					filepath.Join("foo", "conflict.txt"): {
						path: filepath.Join("foo", "conflict.txt"),
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
				root:  filepath.Join("foo"),
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
			wantErr:   false,
			errorKind: nil,
		},
		{
			name: "other older: keep ours",
			collection: &HashCollection{
				root:  filepath.Join("foo"),
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
			other: &HashCollection{
				root:  filepath.Join("foo", "bar"),
				mtime: time.Unix(1111, 0),
				pathToFile: map[string]*File{
					filepath.Join("foo", "conflict.txt"): {
						path: filepath.Join("foo", "conflict.txt"),
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
				root:  filepath.Join("foo"),
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
			wantErr:   false,
			errorKind: nil,
		},
		{
			name: "self zero mtime: keep other",
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
			other: &HashCollection{
				root:  filepath.Join("foo", "bar"),
				mtime: time.Unix(1337, 0),
				pathToFile: map[string]*File{
					filepath.Join("foo", "conflict.txt"): {
						path: filepath.Join("foo", "conflict.txt"),
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
						path: filepath.Join("foo", "conflict.txt"),
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
			wantErr:   false,
			errorKind: nil,
		},
		{
			name: "self older: keep other",
			collection: &HashCollection{
				root:  filepath.Join("foo"),
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
			other: &HashCollection{
				root:  filepath.Join("foo", "bar"),
				mtime: time.Unix(1111, 0),
				pathToFile: map[string]*File{
					filepath.Join("foo", "conflict.txt"): {
						path: filepath.Join("foo", "conflict.txt"),
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
				root:  filepath.Join("foo"),
				mtime: time.Unix(123, 0),
				pathToFile: map[string]*File{
					filepath.Join("foo", "conflict.txt"): {
						path: filepath.Join("foo", "conflict.txt"),
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
			wantErr:   false,
			errorKind: nil,
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

func TestCollectionVerify(t *testing.T) {
	root := t.TempDir()
	createFromTestFiles(t, root, []testFile{
		{
			relativePath: filepath.Join("file.txt"),
			mtime:        time.Unix(100, 0),
			contents:     []byte("file.txt"),
		},
		{
			relativePath: filepath.Join("foo", "bar", "vid.mp4"),
			mtime:        time.Unix(200, 0),
			contents:     []byte("foo/bar/vid.mp4"),
		},
		{
			relativePath: filepath.Join("baz", "omg.doc"),
			mtime:        time.Unix(300, 0),
			contents:     []byte("baz/omg.doc"),
		},
	})

	old := iterateMap
	defer func() { iterateMap = old }()

	iterateMap = func(m map[string]*File, fn func(path string, file *File) error) error {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, path := range keys {
			file := m[path]
			err := fn(path, file)
			if err != nil {
				return err
			}
		}

		return nil
	}

	tests := []struct {
		name             string
		collection       *HashCollection
		include          func(path string) bool
		expectedProgress []VerifyProgress
		wantErr          bool
	}{
		{
			name: "failed to build relative path",
			collection: &HashCollection{
				root:  root,
				name:  "foo.cshd",
				mtime: time.Time{},
				pathToFile: map[string]*File{
					filepath.Join("foo", "bar", "vid.mp4"): {
						path:     filepath.Join("foo", "bar", "vid.mp4"),
						mtime:    time.Time{},
						size:     0,
						hashType: Hash{},
						hash:     []byte{0xab, 0xab},
					},
				},
			},
			expectedProgress: []VerifyProgress{},
			wantErr:          true,
		},
		{
			name: "VerifyFileMissing: file missing is not an error",
			collection: &HashCollection{
				root:  root,
				name:  "foo.cshd",
				mtime: time.Time{},
				pathToFile: map[string]*File{
					filepath.Join(root, "this", "path", "does", "not", "exist1234"): {
						path: filepath.Join(
							root, "this", "path", "does", "not", "exist1234"),
						mtime:    time.Time{},
						size:     0,
						hashType: Hash{},
						hash:     []byte{0xab, 0xab},
					},
				},
			},
			expectedProgress: []VerifyProgress{
				{
					Stage: VerifyPre,
					Common: VerifyProgressCommon{
						TreeRoot: root,
						RelativePath: filepath.Join(
							"this", "path", "does", "not", "exist1234"),
						FileNumberProcessed: 0,
						FileNumberTotal:     1,
						SizeProcessedBytes:  0,
						SizeTotalBytes:      0,
					},
					Done:   0,
					Total:  0,
					Result: 0,
				},
				{
					Stage: VerifyPost,
					Common: VerifyProgressCommon{
						TreeRoot: root,
						RelativePath: filepath.Join(
							"this", "path", "does", "not", "exist1234"),
						FileNumberProcessed: 1,
						FileNumberTotal:     1,
						SizeProcessedBytes:  0,
						SizeTotalBytes:      0,
					},
					Done:   0,
					Total:  0,
					Result: VerifyFileMissing,
				},
			},
			wantErr: false,
		},
		{
			name: "VerifyOK",
			collection: &HashCollection{
				root:  root,
				name:  "foo.cshd",
				mtime: time.Time{},
				pathToFile: map[string]*File{
					filepath.Join(root, "file.txt"): {
						path: filepath.Join(
							root, "file.txt"),
						mtime:    time.Time{},
						size:     8,
						hashType: Hash{crypto.MD5},
						hash: []byte{
							0x3d, 0x8e, 0x57, 0x7b, 0xdd, 0xb1, 0x7d, 0xb3,
							0x39, 0xea, 0xe0, 0xb3, 0xd9, 0xbc, 0xf1, 0x80,
						},
					},
				},
			},
			expectedProgress: []VerifyProgress{
				{
					Stage: VerifyPre,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 0,
						FileNumberTotal:     1,
						SizeProcessedBytes:  0,
						SizeTotalBytes:      8,
					},
					Done:   0,
					Total:  0,
					Result: 0,
				},
				{
					Stage: VerifyDuring,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 0,
						FileNumberTotal:     1,
						SizeProcessedBytes:  0,
						SizeTotalBytes:      8,
					},
					Done:   8,
					Total:  8,
					Result: 0,
				},
				{
					Stage: VerifyDuring,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 0,
						FileNumberTotal:     1,
						SizeProcessedBytes:  0,
						SizeTotalBytes:      8,
					},
					Done:   8,
					Total:  8,
					Result: 0,
				},
				{
					Stage: VerifyPost,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 1,
						FileNumberTotal:     1,
						SizeProcessedBytes:  8,
						SizeTotalBytes:      8,
					},
					Done:   0,
					Total:  0,
					Result: VerifyOK,
				},
			},
			wantErr: false,
		},
		{
			name: "VerifyMismatch",
			collection: &HashCollection{
				root:  root,
				name:  "foo.cshd",
				mtime: time.Time{},
				pathToFile: map[string]*File{
					filepath.Join(root, "file.txt"): {
						path: filepath.Join(
							root, "file.txt"),
						mtime:    time.Time{},
						size:     0,
						hashType: Hash{crypto.MD5},
						hash: []byte{
							0xff, 0x8e, 0x57, 0x7b, 0xdd, 0xb1, 0x7d, 0xb3,
							0x39, 0xea, 0xe0, 0xb3, 0xd9, 0xbc, 0xf1, 0x80,
						},
					},
				},
			},
			expectedProgress: []VerifyProgress{
				{
					Stage: VerifyPre,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 0,
						FileNumberTotal:     1,
						SizeProcessedBytes:  0,
						SizeTotalBytes:      0,
					},
					Done:   0,
					Total:  0,
					Result: 0,
				},
				{
					Stage: VerifyDuring,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 0,
						FileNumberTotal:     1,
						SizeProcessedBytes:  0,
						SizeTotalBytes:      0,
					},
					Done:   8,
					Total:  8,
					Result: 0,
				},
				{
					Stage: VerifyDuring,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 0,
						FileNumberTotal:     1,
						SizeProcessedBytes:  0,
						SizeTotalBytes:      0,
					},
					Done:   8,
					Total:  8,
					Result: 0,
				},
				{
					Stage: VerifyPost,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 1,
						FileNumberTotal:     1,
						SizeProcessedBytes:  0,
						SizeTotalBytes:      0,
					},
					Done:   0,
					Total:  0,
					Result: VerifyMismatch,
				},
			},
			wantErr: false,
		},
		{
			name: "VerifyMismatchSize",
			collection: &HashCollection{
				root:  root,
				name:  "foo.cshd",
				mtime: time.Time{},
				pathToFile: map[string]*File{
					filepath.Join(root, "file.txt"): {
						path: filepath.Join(
							root, "file.txt"),
						mtime:    time.Time{},
						size:     4,
						hashType: Hash{crypto.MD5},
						hash: []byte{
							0xff, 0x8e, 0x57, 0x7b, 0xdd, 0xb1, 0x7d, 0xb3,
							0x39, 0xea, 0xe0, 0xb3, 0xd9, 0xbc, 0xf1, 0x80,
						},
					},
				},
			},
			expectedProgress: []VerifyProgress{
				{
					Stage: VerifyPre,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 0,
						FileNumberTotal:     1,
						SizeProcessedBytes:  0,
						SizeTotalBytes:      4,
					},
					Done:   0,
					Total:  0,
					Result: 0,
				},
				{
					Stage: VerifyPost,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 1,
						FileNumberTotal:     1,
						SizeProcessedBytes:  4,
						SizeTotalBytes:      4,
					},
					Done:   0,
					Total:  0,
					Result: VerifyMismatchSize,
				},
			},
			wantErr: false,
		},
		{
			name: "VerifyMismatchOutdatedHash",
			collection: &HashCollection{
				root:  root,
				name:  "foo.cshd",
				mtime: time.Time{},
				pathToFile: map[string]*File{
					filepath.Join(root, "file.txt"): {
						path: filepath.Join(
							root, "file.txt"),
						mtime:    time.Unix(99, 0),
						size:     8,
						hashType: Hash{crypto.MD5},
						hash: []byte{
							0xff, 0x8e, 0x57, 0x7b, 0xdd, 0xb1, 0x7d, 0xb3,
							0x39, 0xea, 0xe0, 0xb3, 0xd9, 0xbc, 0xf1, 0x80,
						},
					},
				},
			},
			expectedProgress: []VerifyProgress{
				{
					Stage: VerifyPre,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 0,
						FileNumberTotal:     1,
						SizeProcessedBytes:  0,
						SizeTotalBytes:      8,
					},
					Done:   0,
					Total:  0,
					Result: 0,
				},
				{
					Stage: VerifyDuring,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 0,
						FileNumberTotal:     1,
						SizeProcessedBytes:  0,
						SizeTotalBytes:      8,
					},
					Done:   8,
					Total:  8,
					Result: 0,
				},
				{
					Stage: VerifyDuring,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 0,
						FileNumberTotal:     1,
						SizeProcessedBytes:  0,
						SizeTotalBytes:      8,
					},
					Done:   8,
					Total:  8,
					Result: 0,
				},
				{
					Stage: VerifyPost,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 1,
						FileNumberTotal:     1,
						SizeProcessedBytes:  8,
						SizeTotalBytes:      8,
					},
					Done:   0,
					Total:  0,
					Result: VerifyMismatchOutdatedHash,
				},
			},
			wantErr: false,
		},
		{
			name: "VerifyMismatchCorrupted",
			collection: &HashCollection{
				root:  root,
				name:  "foo.cshd",
				mtime: time.Time{},
				pathToFile: map[string]*File{
					filepath.Join(root, "file.txt"): {
						path: filepath.Join(
							root, "file.txt"),
						mtime:    time.Unix(100, 0),
						size:     8,
						hashType: Hash{crypto.MD5},
						hash: []byte{
							0xff, 0x8e, 0x57, 0x7b, 0xdd, 0xb1, 0x7d, 0xb3,
							0x39, 0xea, 0xe0, 0xb3, 0xd9, 0xbc, 0xf1, 0x80,
						},
					},
				},
			},
			expectedProgress: []VerifyProgress{
				{
					Stage: VerifyPre,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 0,
						FileNumberTotal:     1,
						SizeProcessedBytes:  0,
						SizeTotalBytes:      8,
					},
					Done:   0,
					Total:  0,
					Result: 0,
				},
				{
					Stage: VerifyDuring,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 0,
						FileNumberTotal:     1,
						SizeProcessedBytes:  0,
						SizeTotalBytes:      8,
					},
					Done:   8,
					Total:  8,
					Result: 0,
				},
				{
					Stage: VerifyDuring,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 0,
						FileNumberTotal:     1,
						SizeProcessedBytes:  0,
						SizeTotalBytes:      8,
					},
					Done:   8,
					Total:  8,
					Result: 0,
				},
				{
					Stage: VerifyPost,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 1,
						FileNumberTotal:     1,
						SizeProcessedBytes:  8,
						SizeTotalBytes:      8,
					},
					Done:   0,
					Total:  0,
					Result: VerifyMismatchCorrupted,
				},
			},
			wantErr: false,
		},
		{
			name: "mixed results across multiple files",
			collection: &HashCollection{
				root:  root,
				name:  "mixed.cshd",
				mtime: time.Time{},
				pathToFile: map[string]*File{
					// 4. FILE MISSING (not on disk)
					filepath.Join(root, "missing.txt"): {
						path:     filepath.Join(root, "missing.txt"),
						mtime:    time.Time{},
						size:     10,
						hashType: Hash{crypto.MD5},
						hash:     []byte{0xaa},
					},

					// 2. MISMATCH CORRUPTED (content differs)
					filepath.Join(root, "file.txt"): {
						path:     filepath.Join(root, "file.txt"),
						mtime:    time.Unix(100, 0),
						size:     8,
						hashType: Hash{crypto.MD5},
						hash:     []byte{0xff, 0xff, 0xff, 0xff}, // wrong hash
					},

					// 3. MISMATCH SIZE
					filepath.Join(root, "foo", "bar", "vid.mp4"): {
						path:     filepath.Join(root, "foo", "bar", "vid.mp4"),
						mtime:    time.Unix(200, 0),
						size:     9999, // wrong size
						hashType: Hash{crypto.MD5},
						hash:     []byte{0xbb},
					},

					// 1. OUTDATED HASH vs DISK MTIME (disk is newer)
					filepath.Join(root, "baz", "omg.doc"): {
						path:     filepath.Join(root, "baz", "omg.doc"),
						mtime:    time.Unix(1, 0), // older than disk (mtime=300)
						size:     11,
						hashType: Hash{crypto.MD5},
						hash: []byte{
							0x3d, 0x8e, 0x57, 0x7b, 0xdd, 0xb1, 0x7d, 0xb3,
						},
					},
				},
			},
			expectedProgress: []VerifyProgress{
				// 1. MISMATCH OUTDATED
				{
					Stage: VerifyPre,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("baz", "omg.doc"),
						FileNumberProcessed: 0,
						FileNumberTotal:     4,
						SizeProcessedBytes:  0,
						SizeTotalBytes:      10028,
					},
					Done:   0,
					Total:  0,
					Result: 0,
				},
				{
					Stage: VerifyDuring,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("baz", "omg.doc"),
						FileNumberProcessed: 0,
						FileNumberTotal:     4,
						SizeProcessedBytes:  0,
						SizeTotalBytes:      10028,
					},
					Done:   11,
					Total:  11,
					Result: 0,
				},
				{
					Stage: VerifyDuring,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("baz", "omg.doc"),
						FileNumberProcessed: 0,
						FileNumberTotal:     4,
						SizeProcessedBytes:  0,
						SizeTotalBytes:      10028,
					},
					Done:   11,
					Total:  11,
					Result: 0,
				},
				{
					Stage: VerifyPost,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("baz", "omg.doc"),
						FileNumberProcessed: 1,
						FileNumberTotal:     4,
						SizeProcessedBytes:  11,
						SizeTotalBytes:      10028,
					},
					Done:   0,
					Total:  0,
					Result: VerifyMismatchOutdatedHash,
				},
				// 2. MISMATCH CORRUPTED
				{
					Stage: VerifyPre,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 1,
						FileNumberTotal:     4,
						SizeProcessedBytes:  11,
						SizeTotalBytes:      10028,
					},
					Done:   0,
					Total:  0,
					Result: 0,
				},
				{
					Stage: VerifyDuring,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 1,
						FileNumberTotal:     4,
						SizeProcessedBytes:  11,
						SizeTotalBytes:      10028,
					},
					Done:   8,
					Total:  8,
					Result: 0,
				},
				{
					Stage: VerifyDuring,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 1,
						FileNumberTotal:     4,
						SizeProcessedBytes:  11,
						SizeTotalBytes:      10028,
					},
					Done:   8,
					Total:  8,
					Result: 0,
				},
				{
					Stage: VerifyPost,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 2,
						FileNumberTotal:     4,
						SizeProcessedBytes:  19,
						SizeTotalBytes:      10028,
					},
					Done:   0,
					Total:  0,
					Result: VerifyMismatchCorrupted,
				},
				// 3. MISMATCH SIZE
				{
					Stage: VerifyPre,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("foo", "bar", "vid.mp4"),
						FileNumberProcessed: 2,
						FileNumberTotal:     4,
						SizeProcessedBytes:  19,
						SizeTotalBytes:      10028,
					},
					Done:   0,
					Total:  0,
					Result: 0,
				},
				{
					Stage: VerifyPost,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("foo", "bar", "vid.mp4"),
						FileNumberProcessed: 3,
						FileNumberTotal:     4,
						SizeProcessedBytes:  10018,
						SizeTotalBytes:      10028,
					},
					Done:   0,
					Total:  0,
					Result: VerifyMismatchSize,
				},
				// 4. FILE MISSING
				{
					Stage: VerifyPre,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("missing.txt"),
						FileNumberProcessed: 3,
						FileNumberTotal:     4,
						SizeProcessedBytes:  10018,
						SizeTotalBytes:      10028,
					},
					Done:   0,
					Total:  0,
					Result: 0,
				},
				{
					Stage: VerifyPost,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("missing.txt"),
						FileNumberProcessed: 4,
						FileNumberTotal:     4,
						SizeProcessedBytes:  10028,
						SizeTotalBytes:      10028,
					},
					Done:   0,
					Total:  0,
					Result: VerifyFileMissing,
				},
			},
			wantErr: false,
		},
		{
			name: "include func",
			include: func(path string) bool {
				if path == "file.txt" {
					return true
				}

				return false
			},
			collection: &HashCollection{
				root:  root,
				name:  "mixed.cshd",
				mtime: time.Time{},
				pathToFile: map[string]*File{
					// 4. FILE MISSING (not on disk)
					filepath.Join(root, "missing.txt"): {
						path:     filepath.Join(root, "missing.txt"),
						mtime:    time.Time{},
						size:     10,
						hashType: Hash{crypto.MD5},
						hash:     []byte{0xaa},
					},

					// 2. MISMATCH CORRUPTED (content differs)
					filepath.Join(root, "file.txt"): {
						path:     filepath.Join(root, "file.txt"),
						mtime:    time.Unix(100, 0),
						size:     8,
						hashType: Hash{crypto.MD5},
						hash:     []byte{0xff, 0xff, 0xff, 0xff}, // wrong hash
					},

					// 3. MISMATCH SIZE
					filepath.Join(root, "foo", "bar", "vid.mp4"): {
						path:     filepath.Join(root, "foo", "bar", "vid.mp4"),
						mtime:    time.Unix(200, 0),
						size:     9999, // wrong size
						hashType: Hash{crypto.MD5},
						hash:     []byte{0xbb},
					},

					// 1. OUTDATED HASH vs DISK MTIME (disk is newer)
					filepath.Join(root, "baz", "omg.doc"): {
						path:     filepath.Join(root, "baz", "omg.doc"),
						mtime:    time.Unix(1, 0), // older than disk (mtime=300)
						size:     11,
						hashType: Hash{crypto.MD5},
						hash: []byte{
							0x3d, 0x8e, 0x57, 0x7b, 0xdd, 0xb1, 0x7d, 0xb3,
						},
					},
				},
			},
			expectedProgress: []VerifyProgress{
				// 1. MISMATCH OUTDATED
				// skipped
				// 2. MISMATCH CORRUPTED
				{
					Stage: VerifyPre,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 1,
						FileNumberTotal:     4,
						SizeProcessedBytes:  11,
						SizeTotalBytes:      10028,
					},
					Done:   0,
					Total:  0,
					Result: 0,
				},
				{
					Stage: VerifyDuring,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 1,
						FileNumberTotal:     4,
						SizeProcessedBytes:  11,
						SizeTotalBytes:      10028,
					},
					Done:   8,
					Total:  8,
					Result: 0,
				},
				{
					Stage: VerifyDuring,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 1,
						FileNumberTotal:     4,
						SizeProcessedBytes:  11,
						SizeTotalBytes:      10028,
					},
					Done:   8,
					Total:  8,
					Result: 0,
				},
				{
					Stage: VerifyPost,
					Common: VerifyProgressCommon{
						TreeRoot:            root,
						RelativePath:        filepath.Join("file.txt"),
						FileNumberProcessed: 2,
						FileNumberTotal:     4,
						SizeProcessedBytes:  19,
						SizeTotalBytes:      10028,
					},
					Done:   0,
					Total:  0,
					Result: VerifyMismatchCorrupted,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			progressReceived := []VerifyProgress{}
			err := tt.collection.Verify(
				tt.include,
				func(p VerifyProgress) bool {
					progressReceived = append(progressReceived, p)
					return true
				})

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}

				return
			} else {
				assertNoErr(t, err)
			}

			for _, p := range progressReceived {
				t.Logf("progress: stage %v result %v relpath %q",
					p.Stage, p.Result, p.Common.RelativePath)
			}
			assertEqual(t, len(progressReceived), len(tt.expectedProgress))

			for i, want := range tt.expectedProgress {
				got := &progressReceived[i]

				t.Logf("checking progress index %v", i)
				assertEqual(t, got.Stage, want.Stage)
				assertEqual(t, got.Result, want.Result)
				assertEqual(t, got.Total, want.Total)
				assertEqual(t, got.Done, want.Done)
				assertEqual(t, got.Common.RelativePath, want.Common.RelativePath)
				assertEqual(t, got.Common.SizeProcessedBytes, want.Common.SizeProcessedBytes)
				assertEqual(t, got.Common.SizeTotalBytes, want.Common.SizeTotalBytes)
				assertEqual(t, got.Common.FileNumberProcessed, want.Common.FileNumberProcessed)
				assertEqual(t, got.Common.FileNumberTotal, want.Common.FileNumberTotal)
				assertEqual(t, got.Common.TreeRoot, want.Common.TreeRoot)
			}
		})
	}
}

func TestCollectionVerifyStopsOnProgressFuncFalse(t *testing.T) {
	root := makeAbsOrFail(t, "foo")
	collection := newHashCollection(filepath.Join(root, "bar"))
	collection.pathToFile = map[string]*File{
		filepath.Join(root, "ours", "bar.txt"): {
			path:     filepath.Join(root, "ours", "bar.txt"),
			mtime:    time.Unix(12345, 0),
			size:     5678,
			hashType: Hash{crypto.SHA512},
			hash:     []byte{0xab, 0xab, 0xab, 0xab},
		},
		filepath.Join(root, "other", "xer.txt"): {
			path:     filepath.Join(root, "other", "xer.txt"),
			mtime:    time.Unix(898989, 111),
			size:     3344,
			hashType: Hash{crypto.SHA3_256},
			hash:     []byte{0xaa, 0xaa, 0xaa, 0xaa},
		},
	}

	done := 0
	progressReceived := []VerifyProgress{}
	err := collection.Verify(
		func(p string) bool { return true },
		func(p VerifyProgress) bool {
			if p.Stage == VerifyPre {
				done += 1
			}

			continueVerify := true
			if p.Stage == VerifyPost && done == 1 {
				continueVerify = false
			}

			progressReceived = append(progressReceived, p)
			return continueVerify
		})

	assertNoErr(t, err)
	// only 2 instead of 3, since no during due to file missing
	assertEqual(t, len(progressReceived), 2)
	assertEqual(t, done, 1)
}

func TestCollectionVerifyHandlesIncludeNil(t *testing.T) {
	root := makeAbsOrFail(t, "foo")
	collection := newHashCollection(filepath.Join(root, "bar"))
	collection.pathToFile = map[string]*File{
		filepath.Join(root, "foo", "bar.txt"): {
			path:     filepath.Join(root, "foo", "bar.txt"),
			mtime:    time.Unix(12345, 0),
			size:     5678,
			hashType: Hash{crypto.SHA512},
			hash:     []byte{0xab, 0xab, 0xab, 0xab},
		},
	}

	err := collection.Verify(nil, func(p VerifyProgress) bool { return true })

	assertNoErr(t, err)
}

func TestCollectionVerifyHandlesProgressNil(t *testing.T) {
	root := makeAbsOrFail(t, "foo")
	collection := newHashCollection(filepath.Join(root, "bar"))
	collection.pathToFile = map[string]*File{
		filepath.Join(root, "foo", "bar.txt"): {
			path:     filepath.Join(root, "foo", "bar.txt"),
			mtime:    time.Unix(12345, 0),
			size:     5678,
			hashType: Hash{crypto.SHA512},
			hash:     []byte{0xab, 0xab, 0xab, 0xab},
		},
	}

	err := collection.Verify(func(p string) bool { return true }, nil)

	assertNoErr(t, err)
}
