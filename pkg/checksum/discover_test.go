package checksum

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"
)

func setupWalkDir(t *testing.T, root string) {
	t.Helper()

	createFilesFromList(t, root, []string{
		"file.txt",
		"foo/vid.mp4",
		"foo/test.md",
		"foo/bar/file.txt",
		"foo/bar/ex.bin",
		"baz/xer/file.txt",
		"baz/xer/omg.docx",
	})
}

func TestFilteredWalkSkipsMatchedDirectoryOnSkipDir(t *testing.T) {
	testdir := t.TempDir()
	setupWalkDir(t, testdir)

	allPaths := []string{}
	err := FilteredWalk(testdir, Matcher{}, func(path string, d fs.DirEntry, err error) error {
		// NOTE: root is also visited
		if path == testdir {
			return nil
		}

		if d.IsDir() {
			return fs.SkipDir
		} else {
			allPaths = append(allPaths, path)
		}

		return nil
	})
	assertNoErr(t, err)

	expectedPath := filepath.Join(testdir, "file.txt")
	if len(allPaths) != 1 || allPaths[0] != expectedPath {
		t.Fatalf("Expected all directories to be skipped, got: %v", allPaths)
	}
}

func TestFilteredWalkCanBeAborted(t *testing.T) {
	testdir := t.TempDir()
	setupWalkDir(t, testdir)

	expectedErr := errors.New("abort")
	allPaths := []string{}
	err := FilteredWalk(testdir, Matcher{}, func(path string, d fs.DirEntry, err error) error {
		allPaths = append(allPaths, path)
		return expectedErr
	})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected err %s, got %s", expectedErr, err)
	}

	if len(allPaths) != 1 || allPaths[0] != testdir {
		t.Fatalf("expected only the root, got: %v", allPaths)
	}
}

func mustMatcher(t *testing.T, fn func() (Matcher, error)) Matcher {
	t.Helper()

	m, err := fn()
	if err != nil {
		t.Fatalf("failed to create matcher: %v", err)
	}
	return m
}

func TestFilteredWalk(t *testing.T) {
	testdir := t.TempDir()
	setupWalkDir(t, testdir)

	tests := []struct {
		name                  string
		matcher               Matcher
		expectedPaths         []string
		expectedFilteredPaths []string
	}{
		{
			name:    "default matcher",
			matcher: Matcher{},
			expectedPaths: []string{
				"baz/xer/file.txt",
				"baz/xer/omg.docx",
				"file.txt",
				"foo/bar/ex.bin",
				"foo/bar/file.txt",
				"foo/test.md",
				"foo/vid.mp4",
			},
			expectedFilteredPaths: []string{},
		},
		{
			name: "all matcher",
			matcher: mustMatcher(t, func() (Matcher, error) {
				return NewMatcher(WithAllow("**/*"))
			}),
			expectedPaths: []string{
				"baz/xer/file.txt",
				"baz/xer/omg.docx",
				"file.txt",
				"foo/bar/ex.bin",
				"foo/bar/file.txt",
				"foo/test.md",
				"foo/vid.mp4",
			},
			expectedFilteredPaths: []string{},
		},
		{
			name: "only *.txt",
			matcher: mustMatcher(t, func() (Matcher, error) {
				return NewMatcher(WithAllow("**/*.txt"))
			}),
			expectedPaths: []string{
				"baz/xer/file.txt",
				"file.txt",
				"foo/bar/file.txt",
			},
			expectedFilteredPaths: []string{
				"baz/xer/omg.docx",
				"foo/bar/ex.bin",
				"foo/test.md",
				"foo/vid.mp4",
			},
		},
		{
			name: "block foo/",
			matcher: mustMatcher(t, func() (Matcher, error) {
				return NewMatcher(WithBlock("foo/**"))
			}),
			expectedPaths: []string{
				"baz/xer/file.txt",
				"baz/xer/omg.docx",
				"file.txt",
			},
			expectedFilteredPaths: []string{
				"foo/",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allPaths := []string{}
			filteredPaths := []string{}
			err := FilteredWalk(testdir, tt.matcher, func(path string, d fs.DirEntry, err error) error {
				if err == ErrFiltered {
					filteredPaths = append(filteredPaths, path)
					return nil
				}

				if !d.IsDir() {
					allPaths = append(allPaths, path)
				}
				return nil
			})
			assertNoErr(t, err)

			t.Logf("visited paths: %v", allPaths)
			expectedPathsNormalized := normalizeRelativeTestingPath(
				t, testdir, tt.expectedPaths)
			assertSliceEqual(t, allPaths, expectedPathsNormalized)

			t.Logf("filtered paths: %v", filteredPaths)
			expectedFilteredPathsNormalized := normalizeRelativeTestingPath(
				t, testdir, tt.expectedFilteredPaths)
			assertSliceEqual(t, filteredPaths, expectedFilteredPathsNormalized)
		})
	}
}

func TestDiscoverHashFiles(t *testing.T) {
	testdir := t.TempDir()
	createFilesFromList(t, testdir, []string{
		"file.txt",
		"foo.cshd",
		"foo/vid.mp4",
		"foo/2026-05-14.md5",
		"foo/test.md",
		"foo/bar/file.txt",
		"foo/bar/ex.sha512.bin",
		"foo/bar/most_current_2025-10-23.sha512",
		"baz/xer/file.txt",
		"baz/xer/omg.docx",
		"baz/xer/xer_bh_2026-01-13.cshd",
		"check.sha3_256",
	})

	tests := []struct {
		name          string
		matcher       Matcher
		discoverDepth int
		expectedPaths []string
	}{
		{
			name:          "all hash files",
			matcher:       Matcher{},
			discoverDepth: -1,
			expectedPaths: []string{
				"baz/xer/xer_bh_2026-01-13.cshd",
				"check.sha3_256",
				"foo/2026-05-14.md5",
				"foo/bar/most_current_2025-10-23.sha512",
				"foo.cshd",
			},
		},
		{
			name: "all hash files explicit",
			matcher: mustMatcher(t, func() (Matcher, error) {
				return NewMatcher(WithAllow("**/*"))
			}),
			discoverDepth: -1,
			expectedPaths: []string{
				"baz/xer/xer_bh_2026-01-13.cshd",
				"check.sha3_256",
				"foo/2026-05-14.md5",
				"foo/bar/most_current_2025-10-23.sha512",
				"foo.cshd",
			},
		},
		{
			name:          "only root depth",
			matcher:       Matcher{},
			discoverDepth: 0,
			expectedPaths: []string{
				"check.sha3_256",
				"foo.cshd",
			},
		},
		{
			name:          "up to child dirs of root",
			matcher:       Matcher{},
			discoverDepth: 1,
			expectedPaths: []string{
				"check.sha3_256",
				"foo/2026-05-14.md5",
				"foo.cshd",
			},
		},
		{
			name:          "all depth",
			matcher:       Matcher{},
			discoverDepth: 2,
			expectedPaths: []string{
				"baz/xer/xer_bh_2026-01-13.cshd",
				"check.sha3_256",
				"foo/2026-05-14.md5",
				"foo/bar/most_current_2025-10-23.sha512",
				"foo.cshd",
			},
		},
		{
			name: "only .cshd",
			matcher: mustMatcher(t, func() (Matcher, error) {
				return NewMatcher(WithAllow("**/*.cshd"))
			}),
			discoverDepth: -1,
			expectedPaths: []string{
				"baz/xer/xer_bh_2026-01-13.cshd",
				"foo.cshd",
			},
		},
		{
			name: "only .cshd in root",
			matcher: mustMatcher(t, func() (Matcher, error) {
				return NewMatcher(WithAllow("**/*.cshd"))
			}),
			discoverDepth: 0,
			expectedPaths: []string{
				"foo.cshd",
			},
		},
		{
			name: "no .md5",
			matcher: mustMatcher(t, func() (Matcher, error) {
				return NewMatcher(WithBlock("**/*.md5"))
			}),
			discoverDepth: -1,
			expectedPaths: []string{
				"baz/xer/xer_bh_2026-01-13.cshd",
				"check.sha3_256",
				"foo/bar/most_current_2025-10-23.sha512",
				"foo.cshd",
			},
		},
		{
			name: "only root, no .cshd",
			matcher: mustMatcher(t, func() (Matcher, error) {
				return NewMatcher(WithBlock("**/*.cshd"))
			}),
			discoverDepth: 0,
			expectedPaths: []string{
				"check.sha3_256",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := DefaultOptions()
			options.DiscoverHashFilesDepth = tt.discoverDepth
			options.HashFilesMatcher = tt.matcher

			paths, err := discoverHashFiles(testdir, &options, nil)
			assertNoErr(t, err)

			expectedPathsNormalized := normalizeRelativeTestingPath(
				t, testdir, tt.expectedPaths)
			assertSliceEqual(t, paths, expectedPathsNormalized)
		})
	}
}

func TestDirectoryDepth(t *testing.T) {
	tests := []struct {
		base     string
		target   string
		expected int
		wantErr  bool
	}{
		{
			base:     filepath.Join("foo"),
			target:   filepath.Join("foo", "bar"),
			expected: 1,
			wantErr:  false,
		},
		{
			base:     filepath.Join("foo"),
			target:   filepath.Join("foo", "bar", "baz", "xer"),
			expected: 3,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		depth, err := directoryDepth(tt.base, tt.target)

		if tt.wantErr {
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			return
		} else {
			assertNoErr(t, err)
		}

		assertEqual(t, depth, tt.expected)
	}
}
