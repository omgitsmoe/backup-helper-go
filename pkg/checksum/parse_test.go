package checksum

import (
	"crypto"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"encoding/hex"
)

func TestParse(t *testing.T) {
	s := `# version 1
1673815645.7979772,1337,sha512,90b834a83748223190dd1cce445bb1e7582e55948234e962aba9a3004cc558ce061c865a4fae255e048768e7d7011f958dad463243bb3560ee49335ec4c9e8a0
bar foo/bar/baz xer/file.txt`

	hc, err := Parse(strings.NewReader(s))
	_ = hc
	t.Fatalf("foo: %v", err)
}

func TestParseLine(t *testing.T) {
	s := `1673815645.7979772,1337,sha512,90b834a83748223190dd1cce445bb1e7582e55948234e962aba9a3004cc558ce061c865a4fae255e048768e7d7011f958dad463243bb3560ee49335ec4c9e8a0 foo/bar/baz xer/file.txt`
	expectedBytes, err := hex.DecodeString("90b834a83748223190dd1cce445bb1e7582e55948234e962aba9a3004cc558ce061c865a4fae255e048768e7d7011f958dad463243bb3560ee49335ec4c9e8a0")
	assertNoErr(t, err)

	file, err := parseLine(s, 1)
	assertNoErr(t, err)

	assertEqual(t, file.path, filepath.Join("foo", "bar", "baz xer", "file.txt"))
	assertEqual(t, file.size, 1337)
	assertTimeApproxEqual(t, file.mtime, time.Unix(1673815645, 797977200), time.Microsecond)
	assertEqual(t, file.hashType, Hash{crypto.SHA512})
	assertSliceEqual(t, file.hash, expectedBytes)
}
