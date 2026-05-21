package checksum

import (
	"crypto"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TODO test IncrementalPeriodicWriteInterval in separate test func
func TestIncremental(t *testing.T) {
	tests := []struct {
		name                  string
		skipUnchanged         bool
		includeUnchanged      bool
		hashType              Hash
		mostCurrent           func(root string) *HashCollection
		allFilesMatcher       Matcher
		testFiles             []testFile
		expectedSerialization string
		wantErr               bool
	}{
		{
			name:             "empty dir",
			skipUnchanged:    false,
			includeUnchanged: false,
			hashType:         Hash{crypto.MD5},
			mostCurrent: func(root string) *HashCollection {
				return &HashCollection{
					root: root,
					name: "most_current.cshd",
					pathToFile: map[string]*File{
						filepath.Join(root, "foo", "bar", "linux.iso"): {
							path:     filepath.Join(root, "foo", "bar", "linux.iso"),
							mtime:    time.Time{},
							hashType: Hash{crypto.MD5},
							hash:     []byte{0xde, 0xad, 0xbe, 0xef},
						},
					},
				}
			},
			allFilesMatcher:       Matcher{},
			testFiles:             []testFile{},
			expectedSerialization: ``,
			wantErr:               false,
		},
		{
			name:             "all",
			skipUnchanged:    false,
			includeUnchanged: false,
			hashType:         Hash{crypto.MD5},
			mostCurrent: func(root string) *HashCollection {
				return &HashCollection{
					root: root,
					name: "most_current.cshd",
					pathToFile: map[string]*File{
						filepath.Join(root, "foo", "bar", "linux.iso"): {
							path:     filepath.Join(root, "foo", "bar", "linux.iso"),
							mtime:    time.Time{},
							hashType: Hash{crypto.MD5},
							hash:     []byte{0xde, 0xad, 0xbe, 0xef},
						},
					},
				}
			},
			allFilesMatcher: Matcher{},
			testFiles: []testFile{
				{
					// md5 56b6f09c50bfb4706563cdf3463a6cc3
					// sha256 6c7015743716459d7fa1fc359664de136ca215afa0bea28efa1f744b38dff164
					relativePath: "abc.txt",
					mtime:        time.Unix(100, 1_330_000),
				},
				{
					// md5 4874627865d0464347cbaca03fdbe0f5
					relativePath: "foo.cshd",
					mtime:        time.Unix(200, 0),
				},
				{
					// md5 e52e909b8f3a42f43244843ec29e15da
					relativePath: "foo/bar/file.bin",
					mtime:        time.Unix(300, 0),
				},
				{
					// md5 87ae905d8f1fe92704f8e41cac4b81e2
					relativePath: "foo/bar/vid.mp4",
					mtime:        time.Unix(400, 0),
				},
				{
					// md5 2f95b10ff8bbc6367edd718cc8eba062
					relativePath: "nested/dir/a.txt",
					mtime:        time.Unix(500, 0),
				},
				{
					// md5 b05ea47eeb9a9aa7a6a7c751ed34bccc
					relativePath: "nested/dir/sub/foo.doc",
					mtime:        time.Unix(600, 0),
				},
			},
			expectedSerialization: `# version 1
100.00133,7,md5,56b6f09c50bfb4706563cdf3463a6cc3 abc.txt
200,8,md5,4874627865d0464347cbaca03fdbe0f5 foo.cshd
300,16,md5,e52e909b8f3a42f43244843ec29e15da foo/bar/file.bin
400,15,md5,87ae905d8f1fe92704f8e41cac4b81e2 foo/bar/vid.mp4
500,16,md5,2f95b10ff8bbc6367edd718cc8eba062 nested/dir/a.txt
600,22,md5,b05ea47eeb9a9aa7a6a7c751ed34bccc nested/dir/sub/foo.doc
`,
			wantErr: false,
		},
		{
			name:             "mixed unchanged / skipped / changed / new",
			skipUnchanged:    true,
			includeUnchanged: false,
			hashType:         Hash{crypto.MD5},
			mostCurrent: func(root string) *HashCollection {
				return &HashCollection{
					root: root,
					name: "most_current.cshd",
					pathToFile: map[string]*File{
						// unchanged content, but mtime differs -> hash is recomputed,
						// includeUnchanged=false => dropped
						filepath.Join(root, "abc.txt"): {
							path:     filepath.Join(root, "abc.txt"),
							mtime:    time.Unix(100, 0),
							size:     7,
							hashType: Hash{crypto.MD5},
							hash:     []byte{0x56, 0xb6, 0xf0, 0x9c, 0x50, 0xbf, 0xb4, 0x70, 0x65, 0x63, 0xcd, 0xf3, 0x46, 0x3a, 0x6c, 0xc3},
						},

						// same mtime -> skipUnchanged clones previous without hashing
						filepath.Join(root, "file.txt"): {
							path:     filepath.Join(root, "file.txt"),
							mtime:    time.Unix(100, 0),
							size:     8,
							hashType: Hash{crypto.MD5},
							hash:     []byte{0x3d, 0x8e, 0x57, 0x7b, 0xdd, 0xb1, 0x7d, 0xb3, 0x39, 0xea, 0xe0, 0xb3, 0xd9, 0xbc, 0xf1, 0x80},
						},

						// changed content -> hash differs -> included
						filepath.Join(root, "foo", "bar", "vid.mp4"): {
							path:     filepath.Join(root, "foo", "bar", "vid.mp4"),
							mtime:    time.Unix(400, 0),
							size:     15,
							hashType: Hash{crypto.MD5},
							hash:     []byte{0x87, 0xae, 0x90, 0x5d, 0x8f, 0x1f, 0xe9, 0x27, 0x04, 0xf8, 0xe4, 0x1c, 0xac, 0x4b, 0x81, 0xe2},
						},
					},
				}
			},
			allFilesMatcher: Matcher{},
			testFiles: []testFile{
				{
					relativePath: "abc.txt",
					mtime:        time.Unix(101, 0),
					contents:     []byte("abc.txt"), // same hash as previous
				},
				{
					relativePath: "file.txt",
					mtime:        time.Unix(100, 0), // same mtime -> skip hashing, reuse previous
					contents:     []byte("file.txt changed"),
				},
				{
					relativePath: filepath.Join("foo", "bar", "vid.mp4"),
					mtime:        time.Unix(401, 0),
					contents:     []byte("foo/bar/vid.mp4 changed"), // different hash
				},
				{
					relativePath: filepath.Join("nested", "dir", "sub", "foo.doc"),
					mtime:        time.Unix(600, 0),
					contents:     []byte("nested/dir/sub/foo.doc"), // new file
				},
			},
			expectedSerialization: `# version 1
100,8,md5,3d8e577bddb17db339eae0b3d9bcf180 file.txt
401,23,md5,8785a5fc676e75cd98062644c8ecd2ec foo/bar/vid.mp4
600,22,md5,b05ea47eeb9a9aa7a6a7c751ed34bccc nested/dir/sub/foo.doc
`,
			wantErr: false,
		},
		{
			name:             "includeUnchanged=false drops unchanged file",
			skipUnchanged:    true,
			hashType:         Hash{crypto.MD5},
			includeUnchanged: false,
			mostCurrent: func(root string) *HashCollection {
				return &HashCollection{
					root: root,
					name: "most_current.cshd",
					pathToFile: map[string]*File{
						// md5 3d8e577bddb17db339eae0b3d9bcf180
						filepath.Join(root, "file.txt"): {
							path:     filepath.Join(root, "file.txt"),
							mtime:    time.Unix(100, 0),
							size:     8,
							hashType: Hash{crypto.MD5},
							hash: []byte{
								0x3d, 0x8e, 0x57, 0x7b, 0xdd, 0xb1, 0x7d, 0xb3,
								0x39, 0xea, 0xe0, 0xb3, 0xd9, 0xbc, 0xf1, 0x80,
							},
						},
					},
				}
			},
			testFiles: []testFile{
				{
					relativePath: "file.txt",
					mtime:        time.Unix(101, 0),  // different mtime so hashing happens
					contents:     []byte("file.txt"), // same hash as previous
				},
			},
			expectedSerialization: "",
			wantErr:               false,
		},
		{
			name:             "includeUnchanged=true keeps unchanged file",
			skipUnchanged:    false,
			hashType:         Hash{crypto.MD5},
			includeUnchanged: true,
			mostCurrent: func(root string) *HashCollection {
				return &HashCollection{
					root: root,
					name: "most_current.cshd",
					pathToFile: map[string]*File{
						filepath.Join(root, "file.txt"): {
							path:     filepath.Join(root, "file.txt"),
							mtime:    time.Unix(100, 0),
							size:     8,
							hashType: Hash{crypto.MD5},
							hash: []byte{
								0x3d, 0x8e, 0x57, 0x7b, 0xdd, 0xb1, 0x7d, 0xb3,
								0x39, 0xea, 0xe0, 0xb3, 0xd9, 0xbc, 0xf1, 0x80,
							},
						},
					},
				}
			},
			testFiles: []testFile{
				{
					relativePath: "file.txt",
					mtime:        time.Unix(101, 0),
					contents:     []byte("file.txt"),
				},
			},
			expectedSerialization: `# version 1
101,8,md5,3d8e577bddb17db339eae0b3d9bcf180 file.txt
`,
			wantErr: false,
		},
		{
			name:             "changed file is included even when includeUnchanged=false",
			skipUnchanged:    false,
			hashType:         Hash{crypto.MD5},
			includeUnchanged: false,
			mostCurrent: func(root string) *HashCollection {
				return &HashCollection{
					root: root,
					name: "most_current.cshd",
					pathToFile: map[string]*File{
						filepath.Join(root, "file.txt"): {
							path:     filepath.Join(root, "file.txt"),
							mtime:    time.Unix(100, 0),
							size:     8,
							hashType: Hash{crypto.MD5},
							hash: []byte{
								0x3d, 0x8e, 0x57, 0x7b, 0xdd, 0xb1, 0x7d, 0xb3,
								0x39, 0xea, 0xe0, 0xb3, 0xd9, 0xbc, 0xf1, 0x80,
							},
						},
					},
				}
			},
			testFiles: []testFile{
				{
					relativePath: "file.txt",
					mtime:        time.Unix(101, 0),
					contents:     []byte("file.txt changed"),
				},
			},
			expectedSerialization: `# version 1
101,16,md5,984d5fc81394a6c1236876296699dafc file.txt
`,
			wantErr: false,
		},
		{
			name:             "skipUnchanged reuses previous hash when mtime is unchanged",
			skipUnchanged:    true,
			hashType:         Hash{crypto.MD5},
			includeUnchanged: false,
			mostCurrent: func(root string) *HashCollection {
				return &HashCollection{
					root: root,
					name: "most_current.cshd",
					pathToFile: map[string]*File{
						filepath.Join(root, "file.txt"): {
							path:     filepath.Join(root, "file.txt"),
							mtime:    time.Unix(100, 0),
							size:     8,
							hashType: Hash{crypto.MD5},
							hash: []byte{
								0x3d, 0x8e, 0x57, 0x7b, 0xdd, 0xb1, 0x7d, 0xb3,
								0x39, 0xea, 0xe0, 0xb3, 0xd9, 0xbc, 0xf1, 0x80,
							},
						},
					},
				}
			},
			testFiles: []testFile{
				{
					relativePath: "file.txt",
					mtime:        time.Unix(100, 0),          // same mtime => skip hash
					contents:     []byte("file.txt changed"), // would differ if hashed
				},
			},
			expectedSerialization: `# version 1
100,8,md5,3d8e577bddb17db339eae0b3d9bcf180 file.txt
`,
			wantErr: false,
		},
		{
			name:             "matcher filters discovered files",
			skipUnchanged:    false,
			hashType:         Hash{crypto.MD5},
			includeUnchanged: true,
			mostCurrent: func(root string) *HashCollection {
				return &HashCollection{
					root:       root,
					name:       "most_current.cshd",
					pathToFile: map[string]*File{},
				}
			},
			allFilesMatcher: mustMatcher(t, func() (Matcher, error) {
				return NewMatcher(WithAllow("foo/**/*"))
			}),
			testFiles: []testFile{
				{
					relativePath: "file.txt",
					mtime:        time.Unix(100, 0),
				},
				{
					relativePath: filepath.Join("foo", "bar", "vid.mp4"),
					mtime:        time.Unix(200, 0),
				},
				{
					relativePath: filepath.Join("baz", "omg.doc"),
					mtime:        time.Unix(300, 0),
				},
			},
			expectedSerialization: `# version 1
200,15,md5,87ae905d8f1fe92704f8e41cac4b81e2 foo/bar/vid.mp4
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			createFromTestFiles(t, root, tt.testFiles)

			mostCurrent := tt.mostCurrent(root)

			options := DefaultOptions()
			options.HashType = tt.hashType
			options.IncrementalSkipUnchanged = tt.skipUnchanged
			options.IncrementalIncludeUnchangedFiles = tt.includeUnchanged
			options.AllFilesMatcher = tt.allFilesMatcher

			got, err := incremental(root, mostCurrent, &options, nil)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			} else {
				assertNoErr(t, err)
			}

			var b strings.Builder
			ser := NewSerializer(&b)
			ser.Flush(got)
			gotSerialization := b.String()

			assertEqual(t, gotSerialization, tt.expectedSerialization)
		})
	}
}

func TestIncrementalInclude(t *testing.T) {
	root := t.TempDir()
	createFromTestFiles(t, root, []testFile{
		{
			relativePath: "foo/bar/baz.txt",
			mtime:        time.Time{},
			// md5 2c24aaea72e6bdca2403068ccdf8515c
			// sha256 87a9745e78c3c8d613a3201ea6298872bb813dfa0f3d4c3ec57c169c0ea3b869
			contents: []byte("heyho"),
		},
	})

	tests := []struct {
		name            string
		onDisk          *File
		previous        *File
		includeUnchaged bool
		expected        bool
		wantErr         bool
	}{
		{
			name: "same hash and type + include unchanged",
			onDisk: &File{
				path:     filepath.Join(root, "foo", "bar", "baz.txt"),
				mtime:    time.Time{},
				size:     0,
				hashType: Hash{crypto.MD5},
				hash: []byte{
					0x2c, 0x24, 0xaa, 0xea, 0x72, 0xe6, 0xbd, 0xca,
					0x24, 0x03, 0x06, 0x8c, 0xcd, 0xf8, 0x51, 0x5c,
				},
			},
			previous: &File{
				path:     filepath.Join(root, "foo", "bar", "baz.txt"),
				mtime:    time.Time{},
				size:     0,
				hashType: Hash{crypto.MD5},
				hash: []byte{
					0x2c, 0x24, 0xaa, 0xea, 0x72, 0xe6, 0xbd, 0xca,
					0x24, 0x03, 0x06, 0x8c, 0xcd, 0xf8, 0x51, 0x5c,
				},
			},
			includeUnchaged: true,
			expected:        true,
			wantErr:         false,
		},
		{
			name: "same hash and other type + include unchanged",
			onDisk: &File{
				path:     filepath.Join(root, "foo", "bar", "baz.txt"),
				mtime:    time.Time{},
				size:     0,
				hashType: Hash{crypto.SHA256},
				hash: []byte{
					0xde, 0xad, 0xbe, 0xef,
				},
			},
			previous: &File{
				path:     filepath.Join(root, "foo", "bar", "baz.txt"),
				mtime:    time.Time{},
				size:     0,
				hashType: Hash{crypto.MD5},
				hash: []byte{
					0x2c, 0x24, 0xaa, 0xea, 0x72, 0xe6, 0xbd, 0xca,
					0x24, 0x03, 0x06, 0x8c, 0xcd, 0xf8, 0x51, 0x5c,
				},
			},
			includeUnchaged: true,
			expected:        true,
			wantErr:         false,
		},
		{
			name: "same hash and type + no include unchanged",
			onDisk: &File{
				path:     filepath.Join(root, "foo", "bar", "baz.txt"),
				mtime:    time.Time{},
				size:     0,
				hashType: Hash{crypto.MD5},
				hash: []byte{
					0x2c, 0x24, 0xaa, 0xea, 0x72, 0xe6, 0xbd, 0xca,
					0x24, 0x03, 0x06, 0x8c, 0xcd, 0xf8, 0x51, 0x5c,
				},
			},
			previous: &File{
				path:     filepath.Join(root, "foo", "bar", "baz.txt"),
				mtime:    time.Time{},
				size:     0,
				hashType: Hash{crypto.MD5},
				hash: []byte{
					0x2c, 0x24, 0xaa, 0xea, 0x72, 0xe6, 0xbd, 0xca,
					0x24, 0x03, 0x06, 0x8c, 0xcd, 0xf8, 0x51, 0x5c,
				},
			},
			includeUnchaged: false,
			expected:        false,
			wantErr:         false,
		},
		{
			name: "same hash and other type + no include unchanged",
			onDisk: &File{
				path:     filepath.Join(root, "foo", "bar", "baz.txt"),
				mtime:    time.Time{},
				size:     0,
				hashType: Hash{crypto.SHA256},
				hash: []byte{
					0xde, 0xad, 0xbe, 0xef,
				},
			},
			previous: &File{
				path:     filepath.Join(root, "foo", "bar", "baz.txt"),
				mtime:    time.Time{},
				size:     0,
				hashType: Hash{crypto.MD5},
				hash: []byte{
					0x2c, 0x24, 0xaa, 0xea, 0x72, 0xe6, 0xbd, 0xca,
					0x24, 0x03, 0x06, 0x8c, 0xcd, 0xf8, 0x51, 0x5c,
				},
			},
			includeUnchaged: false,
			expected:        false,
			wantErr:         false,
		},
		{
			name: "different hash and type",
			onDisk: &File{
				path:     filepath.Join(root, "foo", "bar", "baz.txt"),
				mtime:    time.Time{},
				size:     0,
				hashType: Hash{crypto.MD5},
				hash: []byte{
					0xff, 0x24, 0xaa, 0xea, 0x72, 0xe6, 0xbd, 0xca,
					0x24, 0x03, 0x06, 0x8c, 0xcd, 0xf8, 0x51, 0x5c,
				},
			},
			previous: &File{
				path:     filepath.Join(root, "foo", "bar", "baz.txt"),
				mtime:    time.Time{},
				size:     0,
				hashType: Hash{crypto.MD5},
				hash: []byte{
					0x2c, 0x24, 0xaa, 0xea, 0x72, 0xe6, 0xbd, 0xca,
					0x24, 0x03, 0x06, 0x8c, 0xcd, 0xf8, 0x51, 0x5c,
				},
			},
			includeUnchaged: false,
			expected:        true,
			wantErr:         false,
		},
		{
			name: "different hash and other type",
			onDisk: &File{
				path:     filepath.Join(root, "foo", "bar", "baz.txt"),
				mtime:    time.Time{},
				size:     0,
				hashType: Hash{crypto.SHA256},
				hash: []byte{
					0xde, 0xad, 0xbe, 0xef,
				},
			},
			previous: &File{
				path:     filepath.Join(root, "foo", "bar", "baz.txt"),
				mtime:    time.Time{},
				size:     0,
				hashType: Hash{crypto.MD5},
				hash: []byte{
					0xff, 0x24, 0xaa, 0xea, 0x72, 0xe6, 0xbd, 0xca,
					0x24, 0x03, 0x06, 0x8c, 0xcd, 0xf8, 0x51, 0x5c,
				},
			},
			includeUnchaged: false,
			expected:        true,
			wantErr:         false,
		},
		{
			name: "different hash and other type => err missing file",
			onDisk: &File{
				path:     filepath.Join(root, "does", "not", "exist.txt"),
				mtime:    time.Time{},
				size:     0,
				hashType: Hash{crypto.SHA256},
				hash: []byte{
					0xde, 0xad, 0xbe, 0xef,
				},
			},
			previous: &File{
				path:     filepath.Join(root, "foo", "bar", "baz.txt"),
				mtime:    time.Time{},
				size:     0,
				hashType: Hash{crypto.MD5},
				hash: []byte{
					0xff, 0x24, 0xaa, 0xea, 0x72, 0xe6, 0xbd, 0xca,
					0x24, 0x03, 0x06, 0x8c, 0xcd, 0xf8, 0x51, 0x5c,
				},
			},
			includeUnchaged: false,
			expected:        false,
			wantErr:         true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := DefaultOptions()
			options.IncrementalIncludeUnchangedFiles = tt.includeUnchaged

			got, err := incrementalInclude(tt.onDisk, tt.previous, &options, nil)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			} else {
				assertNoErr(t, err)
			}

			assertEqual(t, got, tt.expected)
		})
	}
}
