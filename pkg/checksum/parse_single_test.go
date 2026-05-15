package checksum

import (
	"crypto"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseSingle(t *testing.T) {
	root := t.TempDir()
	name := "test.cshd"

	tests := []struct {
		name                   string
		input                  string
		hashType               Hash
		expectedHashCollection HashCollection
		wantErr                bool
	}{
		{
			name: "valid input",
			input: `deadbeef bar foo/bar/baz xer/file.txt
ffffff foo/bar
ababab xer/foo.bin`,
			hashType: Hash{crypto.SHA512},
			expectedHashCollection: HashCollection{
				root:  root,
				name:  name,
				mtime: time.Time{},
				pathToFile: map[string]*File{
					filepath.Join(root, "bar foo", "bar", "baz xer", "file.txt"): {
						path:     filepath.Join(root, "bar foo", "bar", "baz xer", "file.txt"),
						mtime:    time.Time{},
						size:     0,
						hashType: Hash{crypto.SHA512},
						hash:     []byte{0xde, 0xad, 0xbe, 0xef},
					},
					filepath.Join(root, "foo", "bar"): {
						path:     filepath.Join(root, "foo", "bar"),
						mtime:    time.Time{},
						size:     0,
						hashType: Hash{crypto.SHA512},
						hash:     []byte{0xff, 0xff, 0xff},
					},
					filepath.Join(root, "xer", "foo.bin"): {
						path:     filepath.Join(root, "xer", "foo.bin"),
						mtime:    time.Time{},
						size:     0,
						hashType: Hash{crypto.SHA512},
						hash:     []byte{0xab, 0xab, 0xab},
					},
				},
			},
			wantErr: false,
		},
		{
			name:                   "invalid input",
			input:                  `foo`,
			expectedHashCollection: HashCollection{},
			wantErr:                true,
		},
		{
			name: "invalid input: duplicate path",
			input: `ffffff foo/bar
ababab foo/bar`,
			expectedHashCollection: HashCollection{},
			wantErr:                true,
		},
		{
			name:                   "invalid input: invalid hash",
			input:                  `invalidhash foo/bar/baz xer/file.txt`,
			expectedHashCollection: HashCollection{},
			wantErr:                true,
		},
		{
			name:                   "invalid input: missing space",
			input:                  `invalidhashfoo/bar/baz/file.txt`,
			expectedHashCollection: HashCollection{},
			wantErr:                true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hc, err := ParseSingle(
				filepath.Join(root, name),
				tt.hashType,
				strings.NewReader(tt.input),
			)
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
