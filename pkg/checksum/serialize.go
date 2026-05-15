package checksum

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

// NOTE: apparently this is done in the stdlib to get test-time
//
//	dependency injection by overwriting it in tests
//	see https://github.com/golang/go/blob/d36353499f673c89a267a489beb80133a14a75f9/src/database/sql/sql.go#L50
//	https://github.com/golang/go/blob/d36353499f673c89a267a489beb80133a14a75f9/src/database/sql/sql_test.go#L2348-L2349
var serializeRelFunc = filepath.Rel

type Serializer struct {
	w             io.Writer
	headerWritten bool
}

func NewSerializer(w io.Writer) *Serializer {
	return &Serializer{w: w}
}

func (s *Serializer) Flush(c *HashCollection) error {
	if len(c.pathToFile) == 0 {
		return nil
	}
	if err := s.writeHeader(); err != nil {
		return err
	}

	root := c.Root()

	keys := make([]string, 0, len(c.pathToFile))
	for k := range c.pathToFile {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		f := c.pathToFile[k]
		line, err := serializeFile(root, f)
		if err != nil {
			return fmt.Errorf("failed to serialize file %v: %w", line, err)
		}

		if _, err := s.w.Write([]byte(line)); err != nil {
			return err
		}
	}

	return nil
}

func (s *Serializer) writeHeader() error {
	if s.headerWritten {
		return nil
	}

	if _, err := s.w.Write([]byte("# version 1\n")); err != nil {
		return err
	}
	s.headerWritten = true

	return nil
}

func serializeFile(root string, f *File) (string, error) {
	hashType, err := f.hashType.ToIdentifier()
	if err != nil {
		return "", err
	}

	var sizeStr string
	if f.size != 0 {
		sizeStr = strconv.FormatInt(f.size, 10)
	}

	// format with max precision
	var timeStr string
	if (f.mtime != time.Time{}) {
		timeStr = strconv.FormatFloat(timeToF64Time(f.mtime), 'f', -1, 64)
	}

	relativePath, err := serializeRelFunc(root, f.path)
	if err != nil {
		return "", err
	}

	path := filepath.ToSlash(filepath.Clean(relativePath))

	return fmt.Sprintf(
		"%s,%s,%s,%x %s\n",
		timeStr,
		sizeStr,
		hashType,
		f.hash,
		path,
	), nil
}

func timeToF64Time(t time.Time) float64 {
	sec := float64(t.Unix())
	nsec := float64(t.Nanosecond())

	return sec + nsec/1e9
}
