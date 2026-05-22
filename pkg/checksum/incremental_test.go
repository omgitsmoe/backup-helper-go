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
		expectedProgress      []ProgressEvent
		wantErr               bool
	}{
		{
			name:             "empty dir: FileRemoved",
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
			expectedProgress: []ProgressEvent{
				DiscoverFilesDone{
					Found:   0,
					Ignored: 0,
				},
				FileRemoved{Path: filepath.Join("foo", "bar", "linux.iso")},
				Finished{},
			},
			wantErr: false,
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
			expectedProgress: []ProgressEvent{
				DiscoverFilesFound{Count: 1},
				DiscoverFilesFound{Count: 2},
				DiscoverFilesFound{Count: 3},
				DiscoverFilesFound{Count: 4},
				DiscoverFilesFound{Count: 5},
				DiscoverFilesFound{Count: 6},
				DiscoverFilesDone{
					Found:   6,
					Ignored: 0,
				},
				PreRead{Path: "abc.txt"},
				ReadProgress{Read: 7, Total: 7},
				ReadProgress{Read: 7, Total: 7},
				FileNew{Path: "abc.txt"},
				PreRead{Path: filepath.Join("foo", "bar", "file.bin")},
				ReadProgress{Read: 16, Total: 16},
				ReadProgress{Read: 16, Total: 16},
				FileNew{Path: filepath.Join("foo", "bar", "file.bin")},
				PreRead{Path: filepath.Join("foo", "bar", "vid.mp4")},
				ReadProgress{Read: 15, Total: 15},
				ReadProgress{Read: 15, Total: 15},
				FileNew{Path: filepath.Join("foo", "bar", "vid.mp4")},
				PreRead{Path: filepath.Join("foo.cshd")},
				ReadProgress{Read: 8, Total: 8},
				ReadProgress{Read: 8, Total: 8},
				FileNew{Path: filepath.Join("foo.cshd")},
				PreRead{Path: filepath.Join("nested", "dir", "a.txt")},
				ReadProgress{Read: 16, Total: 16},
				ReadProgress{Read: 16, Total: 16},
				FileNew{Path: filepath.Join("nested", "dir", "a.txt")},
				PreRead{Path: filepath.Join("nested", "dir", "sub", "foo.doc")},
				ReadProgress{Read: 22, Total: 22},
				ReadProgress{Read: 22, Total: 22},
				FileNew{Path: filepath.Join("nested", "dir", "sub", "foo.doc")},
				FileRemoved{Path: filepath.Join("foo", "bar", "linux.iso")},
				Finished{},
			},
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
			expectedProgress: []ProgressEvent{
				DiscoverFilesFound{Count: 1},
				DiscoverFilesFound{Count: 2},
				DiscoverFilesFound{Count: 3},
				DiscoverFilesFound{Count: 4},
				DiscoverFilesDone{
					Found:   4,
					Ignored: 0,
				},
				PreRead{Path: "abc.txt"},
				ReadProgress{Read: 7, Total: 7},
				ReadProgress{Read: 7, Total: 7},
				FileMatch{Path: "abc.txt"},
				PreRead{Path: "file.txt"},
				// read skipped due to skipUnchanged
				FileUnchangedSkipped{Path: "file.txt"},
				PreRead{Path: filepath.Join("foo", "bar", "vid.mp4")},
				ReadProgress{Read: 23, Total: 23},
				ReadProgress{Read: 23, Total: 23},
				FileChangedOlder{Path: filepath.Join("foo", "bar", "vid.mp4")},
				PreRead{Path: filepath.Join("nested", "dir", "sub", "foo.doc")},
				ReadProgress{Read: 22, Total: 22},
				ReadProgress{Read: 22, Total: 22},
				FileNew{Path: filepath.Join("nested", "dir", "sub", "foo.doc")},
				Finished{},
			},
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
			expectedProgress: []ProgressEvent{
				DiscoverFilesFound{Count: 1},
				DiscoverFilesDone{
					Found:   1,
					Ignored: 0,
				},
				PreRead{Path: "file.txt"},
				ReadProgress{Read: 8, Total: 8},
				ReadProgress{Read: 8, Total: 8},
				FileMatch{Path: "file.txt"},
				Finished{},
			},
			wantErr: false,
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
			expectedProgress: []ProgressEvent{
				DiscoverFilesFound{Count: 1},
				DiscoverFilesDone{
					Found:   1,
					Ignored: 0,
				},
				PreRead{Path: "file.txt"},
				ReadProgress{Read: 8, Total: 8},
				ReadProgress{Read: 8, Total: 8},
				FileMatch{Path: "file.txt"},
				Finished{},
			},
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
			expectedProgress: []ProgressEvent{
				DiscoverFilesFound{Count: 1},
				DiscoverFilesDone{
					Found:   1,
					Ignored: 0,
				},
				PreRead{Path: "file.txt"},
				ReadProgress{Read: 16, Total: 16},
				ReadProgress{Read: 16, Total: 16},
				FileChangedOlder{Path: "file.txt"},
				Finished{},
			},
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
			expectedProgress: []ProgressEvent{
				DiscoverFilesFound{Count: 1},
				DiscoverFilesDone{
					Found:   1,
					Ignored: 0,
				},
				PreRead{Path: "file.txt"},
				FileUnchangedSkipped{Path: "file.txt"},
				Finished{},
			},
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
			expectedProgress: []ProgressEvent{
				DiscoverFilesIgnored{Path: filepath.Join("baz", "omg.doc")},
				DiscoverFilesIgnored{Path: filepath.Join("file.txt")},
				DiscoverFilesFound{Count: 1},
				DiscoverFilesDone{
					Found:   1,
					Ignored: 2,
				},
				PreRead{Path: filepath.Join("foo", "bar", "vid.mp4")},
				ReadProgress{Read: 15, Total: 15},
				ReadProgress{Read: 15, Total: 15},
				FileNew{Path: filepath.Join("foo", "bar", "vid.mp4")},
				Finished{},
			},
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

			receivedProgress := []ProgressEvent{}
			got, err := incremental(
				root, mostCurrent, &options,
				func(p ProgressEvent) {
					receivedProgress = append(receivedProgress, p)
				},
			)

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
			assertSliceEqual(t, receivedProgress, tt.expectedProgress)
		})
	}
}

func TestIncrementalProgressNil(t *testing.T) {
	root := t.TempDir()
	createFromTestFiles(
		t,
		root,
		[]testFile{
			{relativePath: filepath.Join("foo", "bar", "file.txt")},
		},
	)

	options := DefaultOptions()
	_, err := incremental(root,
		&HashCollection{
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
		},
		&options,
		nil)

	assertNoErr(t, err)
}

func TestIncrementalInclude(t *testing.T) {
	root := t.TempDir()
	createFromTestFiles(t, root, []testFile{
		{
			relativePath: "foo/bar/baz.txt",
			// md5 2c24aaea72e6bdca2403068ccdf8515c
			// sha256 87a9745e78c3c8d613a3201ea6298872bb813dfa0f3d4c3ec57c169c0ea3b869
			contents: []byte("heyho"),
		},
	})

	tests := []struct {
		name             string
		relativePath     string
		onDisk           *File
		previous         *File
		includeUnchaged  bool
		expected         bool
		expectedProgress []ProgressEvent
		wantErr          bool
	}{
		{
			name:         "same hash and type + include unchanged",
			relativePath: "testingpath123",
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
			expectedProgress: []ProgressEvent{
				FileMatch{Path: "testingpath123"},
			},
			wantErr: false,
		},
		{
			name:         "same hash and other type + include unchanged",
			relativePath: "testingpath123",
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
			expectedProgress: []ProgressEvent{
				ReadProgress{
					Read:  5,
					Total: 5,
				},
				ReadProgress{
					Read:  5,
					Total: 5,
				},
				FileMatch{Path: "testingpath123"},
			},
			wantErr: false,
		},
		{
			name:         "same hash and type + no include unchanged",
			relativePath: "testingpath123",
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
			expectedProgress: []ProgressEvent{
				FileMatch{Path: "testingpath123"},
			},
			wantErr: false,
		},
		{
			name:         "same hash and other type + no include unchanged",
			relativePath: "testingpath123",
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
			expectedProgress: []ProgressEvent{
				ReadProgress{
					Read:  5,
					Total: 5,
				},
				ReadProgress{
					Read:  5,
					Total: 5,
				},
				FileMatch{Path: "testingpath123"},
			},
			wantErr: false,
		},
		{
			name:         "different hash and same type",
			relativePath: "testingpath123",
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
			expectedProgress: []ProgressEvent{
				FileChanged{Path: "testingpath123"},
			},
			wantErr: false,
		},
		{
			name:         "FileChangedCorrupted",
			relativePath: "testingpath123",
			onDisk: &File{
				path:     filepath.Join(root, "foo", "bar", "baz.txt"),
				mtime:    time.Unix(100, 0),
				size:     0,
				hashType: Hash{crypto.MD5},
				hash: []byte{
					0xff, 0x24, 0xaa, 0xea, 0x72, 0xe6, 0xbd, 0xca,
					0x24, 0x03, 0x06, 0x8c, 0xcd, 0xf8, 0x51, 0x5c,
				},
			},
			previous: &File{
				path:     filepath.Join(root, "foo", "bar", "baz.txt"),
				mtime:    time.Unix(100, 0),
				size:     0,
				hashType: Hash{crypto.MD5},
				hash: []byte{
					0x2c, 0x24, 0xaa, 0xea, 0x72, 0xe6, 0xbd, 0xca,
					0x24, 0x03, 0x06, 0x8c, 0xcd, 0xf8, 0x51, 0x5c,
				},
			},
			includeUnchaged: false,
			expected:        true,
			expectedProgress: []ProgressEvent{
				FileChangedCorrupted{Path: "testingpath123"},
			},
			wantErr: false,
		},
		{
			name:         "FileChangedOlder",
			relativePath: "testingpath123",
			onDisk: &File{
				path:     filepath.Join(root, "foo", "bar", "baz.txt"),
				mtime:    time.Unix(200, 0),
				size:     0,
				hashType: Hash{crypto.MD5},
				hash: []byte{
					0xff, 0x24, 0xaa, 0xea, 0x72, 0xe6, 0xbd, 0xca,
					0x24, 0x03, 0x06, 0x8c, 0xcd, 0xf8, 0x51, 0x5c,
				},
			},
			previous: &File{
				path:     filepath.Join(root, "foo", "bar", "baz.txt"),
				mtime:    time.Unix(100, 0),
				size:     0,
				hashType: Hash{crypto.MD5},
				hash: []byte{
					0x2c, 0x24, 0xaa, 0xea, 0x72, 0xe6, 0xbd, 0xca,
					0x24, 0x03, 0x06, 0x8c, 0xcd, 0xf8, 0x51, 0x5c,
				},
			},
			includeUnchaged: false,
			expected:        true,
			expectedProgress: []ProgressEvent{
				FileChangedOlder{Path: "testingpath123"},
			},
			wantErr: false,
		},
		{
			name:         "hash newer than on disk => FileChanged",
			relativePath: "testingpath123",
			onDisk: &File{
				path:     filepath.Join(root, "foo", "bar", "baz.txt"),
				mtime:    time.Unix(100, 0),
				size:     0,
				hashType: Hash{crypto.MD5},
				hash: []byte{
					0xff, 0x24, 0xaa, 0xea, 0x72, 0xe6, 0xbd, 0xca,
					0x24, 0x03, 0x06, 0x8c, 0xcd, 0xf8, 0x51, 0x5c,
				},
			},
			previous: &File{
				path:     filepath.Join(root, "foo", "bar", "baz.txt"),
				mtime:    time.Unix(200, 0),
				size:     0,
				hashType: Hash{crypto.MD5},
				hash: []byte{
					0x2c, 0x24, 0xaa, 0xea, 0x72, 0xe6, 0xbd, 0xca,
					0x24, 0x03, 0x06, 0x8c, 0xcd, 0xf8, 0x51, 0x5c,
				},
			},
			includeUnchaged: false,
			expected:        true,
			expectedProgress: []ProgressEvent{
				FileChanged{Path: "testingpath123"},
			},
			wantErr: false,
		},
		{
			name:         "different hash and other type",
			relativePath: "testingpath123",
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
			expectedProgress: []ProgressEvent{
				ReadProgress{
					Read:  5,
					Total: 5,
				},
				ReadProgress{
					Read:  5,
					Total: 5,
				},
				FileChanged{Path: "testingpath123"},
			},
			wantErr: false,
		},
		{
			name:         "different hash and other type => err missing file",
			relativePath: "testingpath123",
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
			includeUnchaged:  false,
			expected:         false,
			expectedProgress: []ProgressEvent{},
			wantErr:          true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := DefaultOptions()
			options.IncrementalIncludeUnchangedFiles = tt.includeUnchaged

			receivedProgress := []ProgressEvent{}
			got, err := incrementalInclude(
				&tt.relativePath,
				tt.onDisk, tt.previous, &options,
				func(p ProgressEvent) {
					receivedProgress = append(receivedProgress, p)
				},
			)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			} else {
				assertNoErr(t, err)
			}

			assertEqual(t, got, tt.expected)
			assertSliceEqual(t, receivedProgress, tt.expectedProgress)
		})
	}
}
