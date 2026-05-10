package checksum

import (
	"os"
	"path/filepath"
	"testing"
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
