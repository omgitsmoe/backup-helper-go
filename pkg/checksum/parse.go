package checksum

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"errors"
	"time"
	"math"
	"encoding/hex"
	"path/filepath"
)

var errSkipCommentOrEmpty = errors.New("skip comment or empty line")
var ErrMissingField = errors.New("missing or empty field")

func Parse(collectionPath string, r io.Reader) (*HashCollection, error) {
	scanner := bufio.NewScanner(r)

	seenHeader := false
	hc := NewHashCollection(collectionPath)
	collectionRoot := hc.Root()
	var version int
	for scanner.Scan() {
		line := scanner.Text()
		if !seenHeader && strings.HasPrefix(line, "#") {
			var err error
			version, err = parseHeader(line)
			if err != nil {
				return &HashCollection{}, err
			}

			seenHeader = true
			continue
		} else {
			seenHeader = true
		}

		file, err := parseLine(collectionRoot, line, version)
		if err == errSkipCommentOrEmpty {
			continue
		}
		if err != nil {
			return &HashCollection{}, err
		}

		err = hc.Insert(&file)
		if err != nil {
			return &HashCollection{}, fmt.Errorf("failed to insert file: %w", err)
		}
	}

	return hc, scanner.Err()
}

func parseHeader(line string) (version int, err error) {
	after, found := strings.CutPrefix(line, "# version ")
	if !found {
		return
	}

	version_str := strings.TrimSpace(after)
	version, err = strconv.Atoi(version_str)
	if err != nil {
		err = fmt.Errorf(
			"failed to parse version number from '%s': %w",
			version_str, err)
	}
	return
}

func parseLine(collectionRoot string, line string, version int) (File, error) {
	if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
		return File{}, errSkipCommentOrEmpty
	}

	numFields := 3
	if version == 1 {
		numFields = 4
	}

	allFields, path, found := strings.Cut(line, " ")
	if !found {
		return File{}, fmt.Errorf(
			"expected a space separating fields and path: %q", line)
	}

	fields := strings.SplitN(allFields, ",", numFields)
	if len(fields) != numFields {
		return File{}, fmt.Errorf(
			"%w: expected %d comma-separated fields, got %q",
			ErrMissingField, numFields, allFields)
	}

	var (
		mtime    time.Time
		size     int64
		hashType Hash
		hash     []byte
		err      error
	)

	mtime, err = parseMTime(fields[0])
	if err != nil {
		return File{}, fmt.Errorf("parse line %q: %w", line, err)
	}

	idx := 1

	if version == 1 {
		size, err = parseSize(fields[idx])
		if err != nil {
			return File{}, fmt.Errorf("parse line %q: %w", line, err)
		}
		idx++
	}

	// --- hash type ---
	hashType, err = parseHashType(fields[idx])
	if err != nil {
		return File{}, fmt.Errorf("parse line %q: %w", line, err)
	}
	idx++

	// --- hash ---
	hash, err = parseHash(fields[idx])
	if err != nil {
		return File{}, fmt.Errorf("parse line %q: %w", line, err)
	}

	path = filepath.Join(collectionRoot, path)
	file := NewFile(path, hashType)
	file.mtime = mtime
	file.size = size
	file.hash = hash

	return file, nil
}

func parseMTime(field string) (time.Time, error) {
	if field != "" {
		f, err := strconv.ParseFloat(field, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf(
				"invalid mtime %q: %w", field, err)
		}
		return mTimeF64ToTime(f), nil
	}

	return time.Time{}, nil
}

func mTimeF64ToTime(mtime float64) time.Time {
	const NS_PER_SEC = 1_000_000_000.0

	seconds := int64(math.Trunc(mtime))
	nanoseconds := int64(math.Mod(mtime, 1.0) * NS_PER_SEC)

	t := time.Unix(seconds, nanoseconds)

	return t
}

func parseSize(field string) (int64, error) {
	if field != "" {
		size, err := strconv.ParseInt(field, 10, 64)
		if err != nil {
			return 0, fmt.Errorf(
				"invalid size %q: %w", field, err)
		}

		return size, nil
	}

	return 0, nil
}

func parseHashType(field string) (Hash, error) {
	if field != "" {
		hashType, err := FromIdentifier(field)
		if err != nil {
			return Hash{}, fmt.Errorf(
				"invalid hash type %q: %w", field, err)
		}

		return hashType, nil
	}

	return Hash{}, fmt.Errorf("empty hash type: %w", ErrMissingField)
}

func parseHash(field  string) ([]byte, error) {
	if field != "" {
		hash, err := hex.DecodeString(field)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid hex hash %q: %w", field, err)
		}

		return hash, nil
	}

	return nil, fmt.Errorf("empty hash: %w", ErrMissingField)
}
