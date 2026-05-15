package checksum

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSortPathsByAscendingMTime(t *testing.T) {
	tests := []struct {
		name       string
		inputPaths []string
		expected   []pathWithMTime
	}{
		{
			name: "order",
			inputPaths: []string{
				filepath.Join("foo", "xer"),
				filepath.Join("xer"),
				filepath.Join("foo", "bar"),
				filepath.Join("bar"),
			},
			expected: []pathWithMTime{
				{Path: filepath.Join("foo", "bar"), MTime: time.Unix(1234, 3_456_000)},
				{Path: filepath.Join("bar"), MTime: time.Unix(1234, 3_459_000)},
				{Path: filepath.Join("foo", "xer"), MTime: time.Unix(42069, 1337)},
				{Path: filepath.Join("xer"), MTime: time.Unix(163378, 0)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			for i, p := range tt.inputPaths {
				tt.inputPaths[i] = filepath.Join(root, p)
			}
			for i, p := range tt.expected {
				path := filepath.Join(root, p.Path)
				tt.expected[i].Path = path

				if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
					t.Fatalf("failed to create parent dirs for test file: %v", err)
				}
				if err := os.WriteFile(path, []byte("dummy"), 0644); err != nil {
					t.Fatalf("failed to write test file at '%q': %s", p.Path, err)
				}

				err := os.Chtimes(path, p.MTime, p.MTime)
				if err != nil {
					t.Fatalf("failed to write mtime of test file: %v", err)
				}
			}

			got, err := sortPathsByAscendingMTime(tt.inputPaths)
			assertNoErr(t, err)

			if len(got) != len(tt.expected) {
				t.Fatalf(
					"length mismatch:\nwant:%v\ngot:%v\n",
					tt.expected, got)
			}

			for i, got := range got {
				want := tt.expected[i]

				assertEqual(t, got.Path, want.Path)
				assertTimeApproxEqual(t, got.MTime, want.MTime, time.Microsecond)
			}
		})
	}
}

func TestSortPathsByAscendingMTimeFileNotFound(t *testing.T) {
	paths, err := sortPathsByAscendingMTime([]string{
		filepath.Join("this", "file", "does", "not", "exist.txt"),
		filepath.Join("neither", "does", "this", "file"),
	})

	assertSliceEqual(t, paths, nil)
	assertErr(t, err)
}

func TestBuildMostCurrent(t *testing.T) {
	tests := []struct {
		name                   string
		discoverHashFilesDepth int
		filterDeleted          bool
		hashFilesMatcher       Matcher
		testFiles              []testFile
		expectedSerialization  string
		wantErr                bool
	}{
		{
			name:                   "empty dir",
			discoverHashFilesDepth: -1,
			filterDeleted:          false,
			hashFilesMatcher:       Matcher{},
			testFiles:              []testFile{},
			expectedSerialization:  ``,
			wantErr:                false,
		},
		{
			name:                   "all",
			discoverHashFilesDepth: -1,
			filterDeleted:          false,
			hashFilesMatcher:       Matcher{},
			testFiles: []testFile{
				{
					relativePath: "file.cshd",
					mtime:        time.Unix(100, 0),
					contents: []byte(`# version 1
1337.00133,42069,sha512,deadbeef abc.txt
33779,2233,md5,abababab foo/bar/file.bin
3500.25,888,sha256,5577 foo/data/vid.mp4
15999.50001,0,sha256,1111 empty.dat
60000.6,2048,sha256,8888 nested/dir/sub/deep.bin
6666.6,4096,sha256,9999 nested/dir/sub/foo.doc
10000.0,64,md5,3333 root.txt
`),
				},
				{
					relativePath: "file.md5",
					mtime:        time.Unix(200, 0),
					contents: []byte(`5577 foo/data/vid.mp4
1111 empty.dat
3344 root.txt
6666 tiny.flag
4444 deep/inside/file.log
`),
				},
				{
					relativePath: "foo/file.cshd",
					mtime:        time.Unix(300, 0),
					contents: []byte(`# version 1
1133779,112233,md5,abababab bar/file.bin
112500.25,11777,sha256,5555 data/blob.bin
113500.25,11888,sha256,5577 data/vid.mp4
`),
				},
				{
					relativePath: "nested/dir/file.sha256",
					mtime:        time.Unix(400, 0),
					contents: []byte(`2222 a.txt
8877 sub/deep.bin
`),
				},
			},
			expectedSerialization: `# version 1
1337.00133,42069,sha512,deadbeef abc.txt
,,md5,4444 deep/inside/file.log
,,md5,1111 empty.dat
1133779,112233,md5,abababab foo/bar/file.bin
112500.25,11777,sha256,5555 foo/data/blob.bin
113500.25,11888,sha256,5577 foo/data/vid.mp4
,,sha256,2222 nested/dir/a.txt
,,sha256,8877 nested/dir/sub/deep.bin
6666.6,4096,sha256,9999 nested/dir/sub/foo.doc
,,md5,3344 root.txt
,,md5,6666 tiny.flag
`,
			wantErr: false,
		},
		{
			name:                   "filter deleted",
			discoverHashFilesDepth: -1,
			filterDeleted:          true,
			hashFilesMatcher:       Matcher{},
			testFiles: []testFile{
				{
					relativePath: "file.cshd",
					mtime:        time.Unix(100, 0),
					contents: []byte(`# version 1
1337.00133,42069,sha512,deadbeef abc.txt
33779,2233,md5,abababab foo/bar/file.bin
3500.25,888,sha256,5577 foo/data/vid.mp4
15999.50001,0,sha256,1111 empty.dat
60000.6,2048,sha256,8888 nested/dir/sub/deep.bin
6666.6,4096,sha256,9999 nested/dir/sub/foo.doc
10000.0,64,md5,3333 root.txt
`),
				},
				{
					relativePath: "nested/dir/file.sha256",
					mtime:        time.Unix(400, 0),
					contents: []byte(`2222 a.txt
8877 sub/deep.bin
`),
				},
				{relativePath: "abc.txt"},
				{relativePath: "foo/bar/file.bin"},
				{relativePath: "nested/dir/a.txt"},
				{relativePath: "nested/dir/sub/foo.doc"},
			},
			expectedSerialization: `# version 1
1337.00133,42069,sha512,deadbeef abc.txt
33779,2233,md5,abababab foo/bar/file.bin
,,sha256,2222 nested/dir/a.txt
6666.6,4096,sha256,9999 nested/dir/sub/foo.doc
`,
			wantErr: false,
		},
		{
			name:                   "discover depth",
			discoverHashFilesDepth: 1,
			filterDeleted:          false,
			hashFilesMatcher:       Matcher{},
			testFiles: []testFile{
				{
					relativePath: "file.cshd",
					mtime:        time.Unix(100, 0),
					contents: []byte(`# version 1
1337.00133,42069,sha512,deadbeef abc.txt
33779,2233,md5,abababab foo/bar/file.bin
3500.25,888,sha256,5577 foo/data/vid.mp4
15999.50001,0,sha256,1111 empty.dat
60000,2048,sha256,8888 nested/dir/sub/deep.bin
6666.6,4096,sha256,9999 nested/dir/sub/foo.doc
10000.0,64,md5,3333 root.txt
`),
				},
				{
					relativePath: "file.md5",
					mtime:        time.Unix(200, 0),
					contents: []byte(`5577 foo/data/vid.mp4
1111 empty.dat
3344 root.txt
6666 tiny.flag
4444 deep/inside/file.log
`),
				},
				{
					relativePath: "foo/file.cshd",
					mtime:        time.Unix(300, 0),
					contents: []byte(`# version 1
1133779,112233,md5,abababab bar/file.bin
112500.25,11777,sha256,5555 data/blob.bin
113500.25,11888,sha256,5577 data/vid.mp4
`),
				},
				{
					relativePath: "nested/dir/file.sha256",
					mtime:        time.Unix(400, 0),
					contents: []byte(`2222 a.txt
8877 sub/deep.bin
`),
				},
			},
			expectedSerialization: `# version 1
1337.00133,42069,sha512,deadbeef abc.txt
,,md5,4444 deep/inside/file.log
,,md5,1111 empty.dat
1133779,112233,md5,abababab foo/bar/file.bin
112500.25,11777,sha256,5555 foo/data/blob.bin
113500.25,11888,sha256,5577 foo/data/vid.mp4
60000,2048,sha256,8888 nested/dir/sub/deep.bin
6666.6,4096,sha256,9999 nested/dir/sub/foo.doc
,,md5,3344 root.txt
,,md5,6666 tiny.flag
`,
			wantErr: false,
		},
		{
			name:                   "discover depth + only .cshd",
			discoverHashFilesDepth: 1,
			filterDeleted:          false,
			hashFilesMatcher: mustMatcher(t, func() (Matcher, error) {
				return NewMatcher(WithAllow("**/*.cshd"))
			}),
			testFiles: []testFile{
				{
					relativePath: "file.cshd",
					mtime:        time.Unix(100, 0),
					contents: []byte(`# version 1
1337.00133,42069,sha512,deadbeef abc.txt
33779,2233,md5,abababab foo/bar/file.bin
3500.25,888,sha256,5577 foo/data/vid.mp4
15999.5,,sha256,1111 empty.dat
60000,2048,sha256,8888 nested/dir/sub/deep.bin
6666.6,4096,sha256,9999 nested/dir/sub/foo.doc
10000,64,md5,3333 root.txt
`),
				},
				{
					relativePath: "file.md5",
					mtime:        time.Unix(200, 0),
					contents: []byte(`5577 foo/data/vid.mp4
1111 empty.dat
3344 root.txt
6666 tiny.flag
4444 deep/inside/file.log
`),
				},
				{
					relativePath: "foo/file.cshd",
					mtime:        time.Unix(300, 0),
					contents: []byte(`# version 1
1133779,112233,md5,abababab bar/file.bin
112500.25,11777,sha256,5555 data/blob.bin
113500.25,11888,sha256,5577 data/vid.mp4
`),
				},
				{
					relativePath: "nested/dir/file.sha256",
					mtime:        time.Unix(400, 0),
					contents: []byte(`2222 a.txt
8877 sub/deep.bin
`),
				},
			},
			expectedSerialization: `# version 1
1337.00133,42069,sha512,deadbeef abc.txt
15999.5,,sha256,1111 empty.dat
1133779,112233,md5,abababab foo/bar/file.bin
112500.25,11777,sha256,5555 foo/data/blob.bin
113500.25,11888,sha256,5577 foo/data/vid.mp4
60000,2048,sha256,8888 nested/dir/sub/deep.bin
6666.6,4096,sha256,9999 nested/dir/sub/foo.doc
10000,64,md5,3333 root.txt
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			createFromTestFiles(t, root, tt.testFiles)

			options := DefaultOptions()
			options.DiscoverHashFilesDepth = tt.discoverHashFilesDepth
			options.MostCurrentFilterDeleted = tt.filterDeleted
			options.HashFilesMatcher = tt.hashFilesMatcher

			got, err := buildMostCurrent(root, &options, nil)

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
