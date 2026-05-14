package checksum

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func assertNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertErr(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func assertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func assertTimeApproxEqual(t *testing.T, got, want time.Time, tolerance time.Duration) {
	t.Helper()

	diff := got.Sub(want)
	if diff < 0 {
		diff = -diff
	}

	if diff > tolerance {
		t.Fatalf(
			"time mismatch: got %v, want %v (diff %v > %v)",
			got, want, diff, tolerance,
		)
	}
}

func createFilesFromList(t *testing.T, root string, relativePaths []string) {
	t.Helper()

	for _, p := range relativePaths {
		fullPath := filepath.Join(root, p)
		dirPath := filepath.Dir(fullPath)
		os.MkdirAll(dirPath, 0755)

		err := os.WriteFile(fullPath, []byte(p), 0644)
		if err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}
}

func normalizeRelativeTestingPath(t *testing.T, root string, relativePaths []string) []string {
	t.Helper()

	pathsNormalized := []string{}
	for _, p := range relativePaths {
		normalized := filepath.Join(
			root, filepath.FromSlash(p))
		pathsNormalized = append(pathsNormalized, normalized)
	}

	return pathsNormalized
}

func assertSliceEqual[T comparable](t *testing.T, actual []T, expected []T) {
	t.Helper()

	if len(actual) != len(expected) {
		t.Fatalf(
			"expected len %d, got %d",
			len(expected), len(actual),
		)
	}

	for i := range expected {
		if expected[i] != actual[i] {
			t.Fatalf("at index %d: expected %v, got %v", i, expected[i], actual[i])
		}
	}
}

func assertHashCollectionsEqual(t *testing.T, got *HashCollection, want *HashCollection) {
	t.Helper()

	if want == nil {
		if got != nil {
			t.Fatalf("wanted a nil HashCollection, got %v", got)
		}
	}
	if got == nil {
		t.Fatalf("wanted a non-nil HashCollection, got %v", got)
	}
	assertEqual(t, got.name, want.name)
	assertEqual(t, got.root, want.root)
	assertTimeApproxEqual(t, got.mtime, want.mtime, time.Microsecond)
	assertEqual(t, len(got.pathToFile), len(want.pathToFile))

	for p, expectedFile := range want.pathToFile {
		actualFile, found := got.pathToFile[p]
		if !found {
			t.Fatalf("expected file at '%q' was not found", p)
		}

		assertEqual(t, actualFile.path, expectedFile.path)
		assertEqual(t, actualFile.size, expectedFile.size)
		assertTimeApproxEqual(t, actualFile.mtime, expectedFile.mtime, time.Microsecond)
		assertEqual(t, actualFile.hashType, expectedFile.hashType)
		assertSliceEqual(t, actualFile.hash, expectedFile.hash)
	}
}
