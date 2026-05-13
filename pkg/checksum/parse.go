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
)

var errSkipCommentOrEmpty = errors.New("skip comment or empty line")

func Parse(r io.Reader) (HashCollection, error) {
	scanner := bufio.NewScanner(r)

	seenHeader := false
	hc := NewHashCollection("")
	var version int
	for scanner.Scan() {
		line := scanner.Text()
		if !seenHeader {
			var err error
			version, err = parseHeader(line)
			if err != nil {
				return HashCollection{}, err
			}

			seenHeader = true
			continue
		}

		file, err := parseLine(line, version)
		if err == errSkipCommentOrEmpty {
			continue
		}
		if err != nil {
			return HashCollection{}, err
		}

		// TODO method
		hc.pathToFile[file.path] = file
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
	err = fmt.Errorf(
		"failed to parse version number from '%s': %w",
		version_str, err)
	return
}

func parseLine(line string, version int) (File, error) {
	if strings.HasPrefix(line, "#") {
		return File{}, errSkipCommentOrEmpty
	}

	numFields := 3
	if version == 1 {
		numFields = 4
	}

	allFields, path, found := strings.Cut(line, " ")
	if !found {
		return File{}, fmt.Errorf(
			"expected a space ' ' to separate the fields from the path, got: %v", line)
	}

	fields := strings.SplitN(allFields, ",", numFields)
	if len(fields) != numFields {
		return File{}, fmt.Errorf(
			"expected %v comma-separated fields, got: %v", numFields, allFields)
	}
	
	i := 0

	var mtime time.Time
	if len(fields[i]) > 0 {
		mtime_float, err := strconv.ParseFloat(fields[i], 64)
		if err != nil {
			return File{}, fmt.Errorf(
				"expected a floating-point unix time stamp, got '%v': %w", fields[i], err)
		}
		mtime = mTimeF64ToTime(mtime_float)
	}
	i += 1

	var size int64
	var err error
	if version == 1 {
		if len(fields[i]) > 0 {
			size, err = strconv.ParseInt(fields[i], 10, 64)
			if err != nil {
				return File{}, fmt.Errorf(
					"expected a size in bytes, got '%v': %w", fields[i], err)
			}
		}
		i += 1
	}

	var hash_type Hash
	if len(fields[i]) > 0 {
		hash_type, err = FromIdentifier(fields[i])
		if err != nil {
			return File{}, fmt.Errorf(
				"expected a known hash identifier, got '%v': %w", fields[i], err)
		}
	}
	i += 1

	var hash_bytes []byte
	if len(fields[i]) > 0 {
		hash_bytes, err = hex.DecodeString(fields[i])
		if err != nil {
			return File{}, fmt.Errorf(
				"expected a hex-encoded string, got '%v': %w", fields[i], err)
		}
	}
	i += 1

	file := NewFile(path, hash_type)
	file.mtime = mtime
	file.size = size
	file.hash = hash_bytes

	return file, nil
}

func mTimeF64ToTime(mtime float64) time.Time {
	const NS_PER_SEC = 1_000_000_000.0

	seconds := int64(math.Trunc(mtime))
	nanoseconds := int64(math.Mod(mtime, 1.0) * NS_PER_SEC)

	t := time.Unix(seconds, nanoseconds)

	return t
}
