package checksum

import (
	"crypto"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTimeConversion(t *testing.T) {
	tests := []struct {
		time        time.Time
		expectedF64 float64
	}{
		{time: time.Unix(1337, 1_330_000), expectedF64: 1337.00133},
		{time: time.Unix(123456, 7_890_123), expectedF64: 123456.007890123},
	}

	for _, tt := range tests {
		asFloat64 := timeToF64Time(tt.time)
		assertEqual(t, asFloat64, tt.expectedF64)
		asTime := mTimeF64ToTime(asFloat64)

		assertTimeApproxEqual(t, asTime, tt.time, time.Microsecond)
	}
}

func setupHashCollection(root string) *HashCollection {
	name := "test.cshd"

	hc := &HashCollection{
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
				mtime:    time.Unix(1337, 1_330_000),
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
	}

	return hc
}

func TestFlushOnlyWritesHeaderOnce(t *testing.T) {
	hc := setupHashCollection(".")
	var sb strings.Builder
	s := NewSerializer(&sb)
	err := s.Flush(hc)
	assertNoErr(t, err)
	err = s.Flush(hc)
	assertNoErr(t, err)
	got := sb.String()

	assertEqual(t, got,
		`# version 1
1673815645.7979772,1337,sha512,deadbeef bar foo/bar/baz xer/file.txt
1337.00133,,md5,ffffff foo/bar
,,sha256,ababab xer/foo.bin
1673815645.7979772,1337,sha512,deadbeef bar foo/bar/baz xer/file.txt
1337.00133,,md5,ffffff foo/bar
,,sha256,ababab xer/foo.bin
`)
}

func TestSerializer(t *testing.T) {
	root := t.TempDir()
	name := "test.cshd"

	tests := []struct {
		name     string
		input    *HashCollection
		expected string
		wantErr  bool
	}{
		{
			name: "stable order",
			input: &HashCollection{
				root:  root,
				name:  name,
				mtime: time.Time{},
				pathToFile: map[string]*File{
					filepath.Join(root, "xer", "foo.bin"): {
						path:     filepath.Join(root, "xer", "foo.bin"),
						mtime:    time.Time{},
						size:     0,
						hashType: Hash{crypto.SHA256},
						hash:     []byte{0xab, 0xab, 0xab},
					},
					filepath.Join(root, "bar foo", "bar", "baz xer", "file.txt"): {
						path:     filepath.Join(root, "bar foo", "bar", "baz xer", "file.txt"),
						mtime:    time.Unix(1673815645, 797977200),
						size:     1337,
						hashType: Hash{crypto.SHA512},
						hash:     []byte{0xde, 0xad, 0xbe, 0xef},
					},
					filepath.Join(root, "foo", "bar"): {
						path:     filepath.Join(root, "foo", "bar"),
						mtime:    time.Unix(1337, 1_330_000),
						size:     0,
						hashType: Hash{crypto.MD5},
						hash:     []byte{0xff, 0xff, 0xff},
					},
				},
			},
			expected: `# version 1
1673815645.7979772,1337,sha512,deadbeef bar foo/bar/baz xer/file.txt
1337.00133,,md5,ffffff foo/bar
,,sha256,ababab xer/foo.bin
`,
			wantErr: false,
		},
		{
			name: "empty collection",
			input: &HashCollection{
				root:       root,
				name:       name,
				mtime:      time.Time{},
				pathToFile: map[string]*File{},
			},
			expected: "",
			wantErr:  false,
		},
		{
			name: "unsupported hash type",
			input: &HashCollection{
				root:  root,
				name:  name,
				mtime: time.Time{},
				pathToFile: map[string]*File{
					filepath.Join(root, "foo"): {
						path:     filepath.Join(root, "foo"),
						mtime:    time.Time{},
						size:     0,
						hashType: Hash{crypto.BLAKE2b_256},
						hash:     []byte{},
					},
				},
			},
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sb strings.Builder
			s := NewSerializer(&sb)
			err := s.Flush(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			} else {
				assertNoErr(t, err)
			}
			got := sb.String()

			if !tt.wantErr {
				assertEqual(t, got, tt.expected)
			}
		})
	}
}

func TestFlushNoRelativePathPossible(t *testing.T) {
	serializeRelFunc = func(_ string, _ string) (string, error) {
		return "", errors.New("can't make path relative")
	}
	defer func() { serializeRelFunc = filepath.Rel }()

	hc := setupHashCollection(".")
	var sb strings.Builder
	s := NewSerializer(&sb)
	err := s.Flush(hc)
	assertErr(t, err)

	if !strings.Contains(err.Error(), "can't make path relative") {
		t.Fatalf("expected an error regarding relative paths, got '%s'", err)
	}
}
