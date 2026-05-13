package checksum

import (
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

	if time.Since(c.Mtime()) > time.Second*3 {
		t.Fatalf("mtime seems too old: %v", c.mtime)
	}
}
