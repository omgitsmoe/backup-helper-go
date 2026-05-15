package checksum

import (
	"crypto"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	root := t.TempDir()
	name := "test.cshd"

	tests := []struct {
		name                   string
		input                  string
		expectedHashCollection HashCollection
		wantErr                bool
	}{
		{
			name: "valid input version 0",
			input: `1673815645.7979772,sha512,deadbeef bar foo/bar/baz xer/file.txt
# comments
# supported
,md5,ffffff foo/bar
,sha256,ababab xer/foo.bin`,
			expectedHashCollection: HashCollection{
				root:  root,
				name:  name,
				mtime: time.Time{},
				pathToFile: map[string]*File{
					filepath.Join(root, "bar foo", "bar", "baz xer", "file.txt"): {
						path:     filepath.Join(root, "bar foo", "bar", "baz xer", "file.txt"),
						mtime:    time.Unix(1673815645, 797977200),
						size:     0,
						hashType: Hash{crypto.SHA512},
						hash:     []byte{0xde, 0xad, 0xbe, 0xef},
					},
					filepath.Join(root, "foo", "bar"): {
						path:     filepath.Join(root, "foo", "bar"),
						mtime:    time.Time{},
						size:     0,
						hashType: Hash{crypto.MD5},
						hash:     []byte{0xff, 0xff, 0xff},
					},
					filepath.Join(root, "xer", "foo.bin"): {
						path:     filepath.Join(root, "xer", "foo.bin"),
						mtime:    time.Time{},
						size:     0,
						hashType: Hash{crypto.SHA256},
						hash:     []byte{0xab, 0xab, 0xab},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid input version 0: duplicate path",
			input: `,md5,ffffff foo/bar
,sha256,ababab foo/bar`,
			expectedHashCollection: HashCollection{},
			wantErr:                true,
		},
		{
			name:                   "invalid input version 0: invalid line",
			input:                  `1673815645.7979772,sha512 bar foo/bar/baz xer/file.txt`,
			expectedHashCollection: HashCollection{},
			wantErr:                true,
		},
		{
			name: "valid input version 1",
			input: `# version 1
# comments
1673815645.7979772,1337,sha512,deadbeef bar foo/bar/baz xer/file.txt
# supported
,,md5,ffffff foo/bar
,42069,sha256,ababab xer/foo.bin`,
			expectedHashCollection: HashCollection{
				root:  root,
				name:  name,
				mtime: time.Time{},
				pathToFile: map[string]*File{
					filepath.Join(root, "bar foo", "bar", "baz xer", "file.txt"): {
						path:     filepath.Join(root, "bar foo", "bar", "baz xer", "file.txt"),
						mtime:    time.Unix(1673815645, 797977200),
						size:     1337,
						hashType: Hash{crypto.SHA512},
						hash:     []byte{0xde, 0xad, 0xbe, 0xef},
					},
					filepath.Join(root, "foo", "bar"): {
						path:     filepath.Join(root, "foo", "bar"),
						mtime:    time.Time{},
						size:     0,
						hashType: Hash{crypto.MD5},
						hash:     []byte{0xff, 0xff, 0xff},
					},
					filepath.Join(root, "xer", "foo.bin"): {
						path:     filepath.Join(root, "xer", "foo.bin"),
						mtime:    time.Time{},
						size:     42069,
						hashType: Hash{crypto.SHA256},
						hash:     []byte{0xab, 0xab, 0xab},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid input version 1: duplicate path",
			input: `# version 1
,,md5,ffffff foo/bar
,,sha256,ababab foo/bar`,
			expectedHashCollection: HashCollection{},
			wantErr:                true,
		},
		{
			name: "invalid input version 1: invalid line",
			input: `# version 1
1673815645.7979772,,sha512 bar foo/bar/baz xer/file.txt`,
			expectedHashCollection: HashCollection{},
			wantErr:                true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hc, err := Parse(filepath.Join(root, name), strings.NewReader(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			} else {
				assertNoErr(t, err)
			}

			assertHashCollectionsEqual(t, hc, &tt.expectedHashCollection)
		})
	}
}

func TestParseLine(t *testing.T) {
	root := t.TempDir()

	tests := []struct {
		name         string
		version      int
		line         string
		expectedFile File
		wantErr      bool
	}{
		{
			name:    "full valid line version 0",
			version: 0,
			line:    "1673815645.7979772,sha512,deadbeef foo/bar/baz xer/file.txt",
			expectedFile: File{
				path:     filepath.Join(root, "foo", "bar", "baz xer", "file.txt"),
				size:     0,
				mtime:    time.Unix(1673815645, 797977200),
				hashType: Hash{crypto.SHA512},
				hash:     []byte{0xde, 0xad, 0xbe, 0xef},
			},
			wantErr: false,
		},
		{
			name:    "valid line with empty fields version 0",
			version: 0,
			line:    ",sha512,deadbeef foo/bar/baz xer/file.txt",
			expectedFile: File{
				path:     filepath.Join(root, "foo", "bar", "baz xer", "file.txt"),
				size:     0,
				mtime:    time.Time{},
				hashType: Hash{crypto.SHA512},
				hash:     []byte{0xde, 0xad, 0xbe, 0xef},
			},
			wantErr: false,
		},
		{
			name:    "full valid line version 1",
			version: 1,
			line:    "1673815645.7979772,1337,sha512,deadbeef foo/bar/baz xer/file.txt",
			expectedFile: File{
				path:     filepath.Join(root, "foo", "bar", "baz xer", "file.txt"),
				size:     1337,
				mtime:    time.Unix(1673815645, 797977200),
				hashType: Hash{crypto.SHA512},
				hash:     []byte{0xde, 0xad, 0xbe, 0xef},
			},
			wantErr: false,
		},
		{
			name:    "valid line with empty fields version 1",
			version: 1,
			line:    ",,sha512,deadbeef foo/bar/baz xer/file.txt",
			expectedFile: File{
				path:     filepath.Join(root, "foo", "bar", "baz xer", "file.txt"),
				size:     0,
				mtime:    time.Time{},
				hashType: Hash{crypto.SHA512},
				hash:     []byte{0xde, 0xad, 0xbe, 0xef},
			},
			wantErr: false,
		},
		{
			name:         "invalid line version 1: missing hash type",
			version:      1,
			line:         "1673815645.7979772,1337,,deadbeef foo/bar/baz xer/file.txt",
			expectedFile: File{},
			wantErr:      true,
		},
		{
			name:         "invalid line version 1: missing hash",
			version:      1,
			line:         "1673815645.7979772,1337,sha512, foo/bar/baz xer/file.txt",
			expectedFile: File{},
			wantErr:      true,
		},
		{
			name:         "invalid line version 1: invalid missing space",
			version:      1,
			line:         "1673815645.7979772,1337,sha512,ffffffffabcd",
			expectedFile: File{},
			wantErr:      true,
		},
		{
			name:         "invalid line version 1: not enough fields",
			version:      1,
			line:         "1673815645.7979772,sha512,ffff foo/bar/baz xer/file.txt",
			expectedFile: File{},
			wantErr:      true,
		},
		{
			name:         "invalid line version 0: not enough fields",
			version:      0,
			line:         "sha512,ffff foo/bar/baz xer/file.txt",
			expectedFile: File{},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := parseLine(root, tt.line, tt.version)
			t.Log(err)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			} else {
				assertNoErr(t, err)
			}

			assertEqual(t, file.path, tt.expectedFile.path)
			assertEqual(t, file.size, tt.expectedFile.size)
			assertTimeApproxEqual(t, file.mtime, tt.expectedFile.mtime, time.Microsecond)
			assertEqual(t, file.hashType, tt.expectedFile.hashType)
			assertSliceEqual(t, file.hash, tt.expectedFile.hash)
		})
	}
}

func TestParseMTime(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedMTime time.Time
		wantErr       bool
	}{
		{
			name:          "empty string",
			input:         "",
			expectedMTime: time.Time{},
			wantErr:       false,
		},
		{
			name:          "valid float",
			input:         "1673815645.7979772",
			expectedMTime: mTimeF64ToTime(1673815645.7979772),
			wantErr:       false,
		},
		{
			name:    "invalid float",
			input:   "not-a-number",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMTime(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !got.Equal(tt.expectedMTime) {
				t.Fatalf("got %v, want %v", got, tt.expectedMTime)
			}
		})
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
		wantErr  bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "valid number",
			input:    "1673815645",
			expected: 1673815645,
			wantErr:  false,
		},
		{
			name:    "invalid number",
			input:   "not-a-number",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSize(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.expected {
				t.Fatalf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseHashType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Hash
		wantErr  bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: Hash{},
			wantErr:  true,
		},
		{
			name:     "valid hash type",
			input:    "sha512",
			expected: Hash{crypto.SHA512},
			wantErr:  false,
		},
		{
			name:    "invalid hash type",
			input:   "foo",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseHashType(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.expected {
				t.Fatalf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseHash(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []byte
		wantErr  bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "valid hash",
			input:    "deadbeef",
			expected: []byte{0xde, 0xad, 0xbe, 0xef},
			wantErr:  false,
		},
		{
			name:    "invalid hash",
			input:   "deadbeefzz",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseHash(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assertSliceEqual(t, got, tt.expected)
		})
	}
}

func TestParseHeader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		wantErr  bool
	}{
		{
			name:     "missing",
			input:    "",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "only whitespace",
			input:    "     \t  ",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "version 1",
			input:    "# version 1",
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "version 1 with extra whitespace",
			input:    "# version    \t 1",
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "version invalid",
			input:    "# version foo",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "comment",
			input:    "# foo bar",
			expected: 0,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseHeader(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assertEqual(t, got, tt.expected)
		})
	}
}
