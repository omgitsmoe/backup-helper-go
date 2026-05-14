package checksum

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

func ParseSingle(collectionPath string, hashType Hash, r io.Reader) (*HashCollection, error) {
	scanner := bufio.NewScanner(r)

	hc := NewHashCollection(collectionPath)
	collectionRoot := hc.Root()
	for scanner.Scan() {
		line := scanner.Text()

		hash, path, found := strings.Cut(line, " ")
		if !found {
			return &HashCollection{}, fmt.Errorf(
				"expected a space separating hash and path: %q", line)
		}

		hashBytes, err := parseHash(hash)
		if err != nil {
			return &HashCollection{}, fmt.Errorf("parse line %q: %w", line, err)
		}

		path = filepath.Join(collectionRoot, path)
		file := NewFile(path, hashType)
		file.hash = hashBytes

		err = hc.Insert(&file)
		if err != nil {
			return &HashCollection{}, fmt.Errorf("failed to insert file: %w", err)
		}
	}

	return hc, scanner.Err()
}
