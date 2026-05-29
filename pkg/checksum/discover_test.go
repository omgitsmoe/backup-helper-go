package checksum

import (
	"errors"
	"io/fs"
	"os"
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
		name             string
		matcher          Matcher
		discoverDepth    int
		expectedPaths    []string
		expectedProgress []ProgressEvent
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
			expectedProgress: []ProgressEvent{
				MostCurrentFoundFile{Path: filepath.Join("baz", "xer", "xer_bh_2026-01-13.cshd")},
				MostCurrentFoundFile{Path: "check.sha3_256"},
				MostCurrentFoundFile{Path: filepath.Join("foo", "2026-05-14.md5")},
				MostCurrentFoundFile{Path: filepath.Join("foo", "bar", "most_current_2025-10-23.sha512")},
				MostCurrentFoundFile{Path: "foo.cshd"},
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
			expectedProgress: []ProgressEvent{
				MostCurrentFoundFile{Path: filepath.Join("baz", "xer", "xer_bh_2026-01-13.cshd")},
				MostCurrentFoundFile{Path: "check.sha3_256"},
				MostCurrentFoundFile{Path: filepath.Join("foo", "2026-05-14.md5")},
				MostCurrentFoundFile{Path: filepath.Join("foo", "bar", "most_current_2025-10-23.sha512")},
				MostCurrentFoundFile{Path: "foo.cshd"},
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
			expectedProgress: []ProgressEvent{
				MostCurrentIgnoredPath{Path: "baz"},
				MostCurrentFoundFile{Path: "check.sha3_256"},
				MostCurrentIgnoredPath{Path: "foo"},
				MostCurrentFoundFile{Path: "foo.cshd"},
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
			expectedProgress: []ProgressEvent{
				MostCurrentIgnoredPath{Path: filepath.Join("baz", "xer")},
				MostCurrentFoundFile{Path: "check.sha3_256"},
				MostCurrentFoundFile{Path: filepath.Join("foo", "2026-05-14.md5")},
				MostCurrentIgnoredPath{Path: filepath.Join("foo", "bar")},
				MostCurrentFoundFile{Path: "foo.cshd"},
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
			expectedProgress: []ProgressEvent{
				MostCurrentFoundFile{Path: filepath.Join("baz", "xer", "xer_bh_2026-01-13.cshd")},
				MostCurrentFoundFile{Path: "check.sha3_256"},
				MostCurrentFoundFile{Path: filepath.Join("foo", "2026-05-14.md5")},
				MostCurrentFoundFile{Path: filepath.Join("foo", "bar", "most_current_2025-10-23.sha512")},
				MostCurrentFoundFile{Path: "foo.cshd"},
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
			expectedProgress: []ProgressEvent{
				MostCurrentFoundFile{Path: filepath.Join("baz", "xer", "xer_bh_2026-01-13.cshd")},
				MostCurrentIgnoredPath{Path: "check.sha3_256"},
				MostCurrentIgnoredPath{Path: filepath.Join("foo", "2026-05-14.md5")},
				MostCurrentIgnoredPath{Path: filepath.Join("foo", "bar", "most_current_2025-10-23.sha512")},
				MostCurrentFoundFile{Path: "foo.cshd"},
			},
		},
		{
			name: "only .cshd except baz/",
			matcher: mustMatcher(t, func() (Matcher, error) {
				return NewMatcher(
					WithAllow("**/*.cshd"),
					WithBlock("baz/"),
				)
			}),
			discoverDepth: -1,
			expectedPaths: []string{
				"foo.cshd",
			},
			expectedProgress: []ProgressEvent{
				MostCurrentIgnoredPath{Path: "baz"},
				MostCurrentIgnoredPath{Path: "check.sha3_256"},
				MostCurrentIgnoredPath{Path: filepath.Join("foo", "2026-05-14.md5")},
				MostCurrentIgnoredPath{Path: filepath.Join("foo", "bar", "most_current_2025-10-23.sha512")},
				MostCurrentFoundFile{Path: "foo.cshd"},
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
			expectedProgress: []ProgressEvent{
				MostCurrentIgnoredPath{Path: "baz"},
				MostCurrentIgnoredPath{Path: "check.sha3_256"},
				MostCurrentIgnoredPath{Path: "foo"},
				MostCurrentFoundFile{Path: "foo.cshd"},
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
			expectedProgress: []ProgressEvent{
				MostCurrentFoundFile{Path: filepath.Join("baz", "xer", "xer_bh_2026-01-13.cshd")},
				MostCurrentFoundFile{Path: "check.sha3_256"},
				MostCurrentIgnoredPath{Path: filepath.Join("foo", "2026-05-14.md5")},
				MostCurrentFoundFile{Path: filepath.Join("foo", "bar", "most_current_2025-10-23.sha512")},
				MostCurrentFoundFile{Path: "foo.cshd"},
			},
		},
		{
			name: "no .md5 and no foo/",
			matcher: mustMatcher(t, func() (Matcher, error) {
				return NewMatcher(
					WithBlock("**/*.md5"),
					WithBlock("foo/"),
				)
			}),
			discoverDepth: -1,
			expectedPaths: []string{
				"baz/xer/xer_bh_2026-01-13.cshd",
				"check.sha3_256",
				"foo.cshd",
			},
			expectedProgress: []ProgressEvent{
				MostCurrentFoundFile{Path: filepath.Join("baz", "xer", "xer_bh_2026-01-13.cshd")},
				MostCurrentFoundFile{Path: "check.sha3_256"},
				MostCurrentIgnoredPath{Path: "foo"},
				MostCurrentFoundFile{Path: "foo.cshd"},
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
			expectedProgress: []ProgressEvent{
				MostCurrentIgnoredPath{Path: "baz"},
				MostCurrentFoundFile{Path: "check.sha3_256"},
				MostCurrentIgnoredPath{Path: "foo"},
				MostCurrentIgnoredPath{Path: "foo.cshd"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := DefaultOptions()
			options.DiscoverHashFilesDepth = tt.discoverDepth
			options.HashFilesMatcher = tt.matcher

			receivedProgress := []ProgressEvent{}
			paths, err := discoverHashFiles(testdir, &options, func(p ProgressEvent) {
				receivedProgress = append(receivedProgress, p)
			})
			assertNoErr(t, err)

			expectedPathsNormalized := normalizeRelativeTestingPath(
				t, testdir, tt.expectedPaths)
			assertSliceEqual(t, paths, expectedPathsNormalized)
			assertSliceEqual(t, receivedProgress, tt.expectedProgress)
		})
	}
}

func TestDiscoverHashFilesOnlyFiles(t *testing.T) {
	testdir := t.TempDir()
	createFilesFromList(t, testdir, []string{
		"bar.cshd/foo.txt",
		"foo/bar/most_current_2025-10-23.sha512",
		"xer.sha512/baz.md5/vid.mp4",
	})

	opt := DefaultOptions()
	paths, err := discoverHashFiles(testdir, &opt, nil)
	assertNoErr(t, err)

	expectedPathsNormalized := normalizeRelativeTestingPath(
		t, testdir, []string{
			"foo/bar/most_current_2025-10-23.sha512",
		})
	assertSliceEqual(t, paths, expectedPathsNormalized)
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

func TestDiscoverFiles(t *testing.T) {
	testdir := t.TempDir()
	createFilesFromList(t, testdir, []string{
		"baz/xer/file.txt",
		"baz/xer/omg.docx",
		"baz/xer/xer_bh_2026-01-13.cshd",
		"check.sha3_256",
		"file.txt",
		"foo/2026-05-14.md5",
		"foo/bar/ex.sha512.bin",
		"foo/bar/file.txt",
		"foo/bar/most_current_2025-10-23.sha512",
		"foo/test.md",
		"foo/vid.mp4",
		"foo.cshd",
	})

	tests := []struct {
		name             string
		matcher          Matcher
		expectedPaths    []string
		expectedProgress []ProgressEvent
	}{
		{
			name:    "all files",
			matcher: Matcher{},
			expectedPaths: []string{
				"baz/xer/file.txt",
				"baz/xer/omg.docx",
				"baz/xer/xer_bh_2026-01-13.cshd",
				"check.sha3_256",
				"file.txt",
				"foo/2026-05-14.md5",
				"foo/bar/ex.sha512.bin",
				"foo/bar/file.txt",
				"foo/bar/most_current_2025-10-23.sha512",
				"foo/test.md",
				"foo/vid.mp4",
				"foo.cshd",
			},
			expectedProgress: []ProgressEvent{
				DiscoverFilesFound{Count: 1},
				DiscoverFilesFound{Count: 2},
				DiscoverFilesFound{Count: 3},
				DiscoverFilesFound{Count: 4},
				DiscoverFilesFound{Count: 5},
				DiscoverFilesFound{Count: 6},
				DiscoverFilesFound{Count: 7},
				DiscoverFilesFound{Count: 8},
				DiscoverFilesFound{Count: 9},
				DiscoverFilesFound{Count: 10},
				DiscoverFilesFound{Count: 11},
				DiscoverFilesFound{Count: 12},
				DiscoverFilesDone{
					Found:   12,
					Ignored: 0,
				},
			},
		},
		{
			name: "all files explicit",
			matcher: mustMatcher(t, func() (Matcher, error) {
				return NewMatcher(WithAllow("**/*"))
			}),
			expectedPaths: []string{
				"baz/xer/file.txt",
				"baz/xer/omg.docx",
				"baz/xer/xer_bh_2026-01-13.cshd",
				"check.sha3_256",
				"file.txt",
				"foo/2026-05-14.md5",
				"foo/bar/ex.sha512.bin",
				"foo/bar/file.txt",
				"foo/bar/most_current_2025-10-23.sha512",
				"foo/test.md",
				"foo/vid.mp4",
				"foo.cshd",
			},
			expectedProgress: []ProgressEvent{
				DiscoverFilesFound{Count: 1},
				DiscoverFilesFound{Count: 2},
				DiscoverFilesFound{Count: 3},
				DiscoverFilesFound{Count: 4},
				DiscoverFilesFound{Count: 5},
				DiscoverFilesFound{Count: 6},
				DiscoverFilesFound{Count: 7},
				DiscoverFilesFound{Count: 8},
				DiscoverFilesFound{Count: 9},
				DiscoverFilesFound{Count: 10},
				DiscoverFilesFound{Count: 11},
				DiscoverFilesFound{Count: 12},
				DiscoverFilesDone{
					Found:   12,
					Ignored: 0,
				},
			},
		},
		{
			name: "only *.txt",
			matcher: mustMatcher(t, func() (Matcher, error) {
				return NewMatcher(WithAllow("*.txt"))
			}),
			expectedPaths: []string{
				"file.txt",
			},
			expectedProgress: []ProgressEvent{
				DiscoverFilesIgnored{Path: filepath.Join("baz", "xer", "file.txt")},
				DiscoverFilesIgnored{Path: filepath.Join("baz", "xer", "omg.docx")},
				DiscoverFilesIgnored{Path: filepath.Join("baz", "xer", "xer_bh_2026-01-13.cshd")},
				DiscoverFilesIgnored{Path: filepath.Join("check.sha3_256")},
				DiscoverFilesFound{Count: 1},
				DiscoverFilesIgnored{Path: filepath.Join("foo", "2026-05-14.md5")},
				DiscoverFilesIgnored{Path: filepath.Join("foo", "bar", "ex.sha512.bin")},
				DiscoverFilesIgnored{Path: filepath.Join("foo", "bar", "file.txt")},
				DiscoverFilesIgnored{Path: filepath.Join("foo", "bar", "most_current_2025-10-23.sha512")},
				DiscoverFilesIgnored{Path: filepath.Join("foo", "test.md")},
				DiscoverFilesIgnored{Path: filepath.Join("foo", "vid.mp4")},
				DiscoverFilesIgnored{Path: filepath.Join("foo.cshd")},
				DiscoverFilesDone{
					Found:   1,
					Ignored: 11,
				},
			},
		},
		{
			name: "all *.txt files",
			matcher: mustMatcher(t, func() (Matcher, error) {
				return NewMatcher(WithAllow("**/*.txt"))
			}),
			expectedPaths: []string{
				"baz/xer/file.txt",
				"file.txt",
				"foo/bar/file.txt",
			},
			expectedProgress: []ProgressEvent{
				DiscoverFilesFound{Count: 1},
				DiscoverFilesIgnored{Path: filepath.Join("baz", "xer", "omg.docx")},
				DiscoverFilesIgnored{Path: filepath.Join("baz", "xer", "xer_bh_2026-01-13.cshd")},
				DiscoverFilesIgnored{Path: filepath.Join("check.sha3_256")},
				DiscoverFilesFound{Count: 2},
				DiscoverFilesIgnored{Path: filepath.Join("foo", "2026-05-14.md5")},
				DiscoverFilesIgnored{Path: filepath.Join("foo", "bar", "ex.sha512.bin")},
				DiscoverFilesFound{Count: 3},
				DiscoverFilesIgnored{Path: filepath.Join("foo", "bar", "most_current_2025-10-23.sha512")},
				DiscoverFilesIgnored{Path: filepath.Join("foo", "test.md")},
				DiscoverFilesIgnored{Path: filepath.Join("foo", "vid.mp4")},
				DiscoverFilesIgnored{Path: filepath.Join("foo.cshd")},
				DiscoverFilesDone{
					Found:   3,
					Ignored: 9,
				},
			},
		},
		{
			name: "all *.txt files, except foo/",
			matcher: mustMatcher(t, func() (Matcher, error) {
				return NewMatcher(
					WithAllow("**/*.txt"),
					WithBlock("foo/"),
				)
			}),
			expectedPaths: []string{
				"baz/xer/file.txt",
				"file.txt",
			},
			expectedProgress: []ProgressEvent{
				DiscoverFilesFound{Count: 1},
				DiscoverFilesIgnored{Path: filepath.Join("baz", "xer", "omg.docx")},
				DiscoverFilesIgnored{Path: filepath.Join("baz", "xer", "xer_bh_2026-01-13.cshd")},
				DiscoverFilesIgnored{Path: filepath.Join("check.sha3_256")},
				DiscoverFilesFound{Count: 2},
				// NOTE: files inside directories are not counted
				DiscoverFilesIgnored{Path: filepath.Join("foo")},
				DiscoverFilesIgnored{Path: filepath.Join("foo.cshd")},
				DiscoverFilesDone{
					Found:   2,
					Ignored: 5,
				},
			},
		},
		{
			name: "only baz/xer/ except *.txt",
			matcher: mustMatcher(t, func() (Matcher, error) {
				return NewMatcher(
					WithAllow("baz/xer/**/*"),
					WithBlock("**/*.txt"),
				)
			}),
			expectedPaths: []string{
				"baz/xer/omg.docx",
				"baz/xer/xer_bh_2026-01-13.cshd",
			},
			expectedProgress: []ProgressEvent{
				DiscoverFilesIgnored{Path: filepath.Join("baz", "xer", "file.txt")},
				DiscoverFilesFound{Count: 1},
				DiscoverFilesFound{Count: 2},
				DiscoverFilesIgnored{Path: filepath.Join("check.sha3_256")},
				DiscoverFilesIgnored{Path: filepath.Join("file.txt")},
				DiscoverFilesIgnored{Path: filepath.Join("foo", "2026-05-14.md5")},
				DiscoverFilesIgnored{Path: filepath.Join("foo", "bar", "ex.sha512.bin")},
				DiscoverFilesIgnored{Path: filepath.Join("foo", "bar", "file.txt")},
				DiscoverFilesIgnored{Path: filepath.Join("foo", "bar", "most_current_2025-10-23.sha512")},
				DiscoverFilesIgnored{Path: filepath.Join("foo", "test.md")},
				DiscoverFilesIgnored{Path: filepath.Join("foo", "vid.mp4")},
				DiscoverFilesIgnored{Path: filepath.Join("foo.cshd")},
				DiscoverFilesDone{
					Found:   2,
					Ignored: 10,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := DefaultOptions()
			options.AllFilesMatcher = tt.matcher

			receivedProgress := []ProgressEvent{}
			paths, err := discoverFiles(testdir, &options, func(p ProgressEvent) {
				receivedProgress = append(receivedProgress, p)
			})
			assertNoErr(t, err)

			expectedPathsNormalized := normalizeRelativeTestingPath(
				t, testdir, tt.expectedPaths)
			assertSliceEqual(t, paths, expectedPathsNormalized)
			assertSliceEqual(t, receivedProgress, tt.expectedProgress)
		})
	}
}

func TestDiscoverFilesOnlyFiles(t *testing.T) {
	testdir := t.TempDir()
	createFilesFromList(t, testdir, []string{
		"bar.bin/foo.txt",
		"foo/bar/most_current_2025-10-23.sha512",
		"xer.sha512/baz.doc/vid.mp4",
	})

	opt := DefaultOptions()
	paths, err := discoverFiles(testdir, &opt, nil)
	assertNoErr(t, err)

	expectedPathsNormalized := normalizeRelativeTestingPath(
		t, testdir, []string{
			"bar.bin/foo.txt",
			"foo/bar/most_current_2025-10-23.sha512",
			"xer.sha512/baz.doc/vid.mp4",
		})
	assertSliceEqual(t, paths, expectedPathsNormalized)
}

func TestDiscoverFilesProgressNil(t *testing.T) {
	testdir := t.TempDir()
	createFilesFromList(t, testdir, []string{
		"bar.bin/foo.txt",
		"foo/bar/most_current_2025-10-23.sha512",
		"xer.sha512/baz.doc/vid.mp4",
	})

	opt := DefaultOptions()
	_, err := discoverFiles(testdir, &opt, nil)
	assertNoErr(t, err)
}

func TestFilteredWalk_FiltersSymlinkedDirs(t *testing.T) {
	root := t.TempDir()

	regularDir := filepath.Join(root, "dir")
	if err := os.Mkdir(regularDir, 0o755); err != nil {
		t.Fatalf("mkdir regular dir: %v", err)
	}

	regularFile := filepath.Join(root, "file.txt")
	if err := os.WriteFile(regularFile, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write regular file: %v", err)
	}

	linkToFile := filepath.Join(root, "link-file")
	if err := os.Symlink(regularFile, linkToFile); err != nil {
		t.Skipf("symlinks not supported on this system: %v", err)
	}

	linkToDir := filepath.Join(root, "link-dir")
	if err := os.Symlink(regularDir, linkToDir); err != nil {
		t.Skipf("symlinks not supported on this system: %v", err)
	}

	type call struct {
		rel string
		err error
	}
	var got []call

	fn := func(path string, d fs.DirEntry, err error) error {
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			t.Fatalf("filepath.Rel(%q, %q): %v", root, path, relErr)
		}
		got = append(got, call{rel: rel, err: err})
		return nil
	}

	if err := FilteredWalk(root, Matcher{}, fn); err != nil {
		t.Fatalf("FilteredWalk returned error: %v", err)
	}

	want := map[string]error{
		".":         nil,                    // root dir
		"dir":       nil,                    // regular dir
		"file.txt":  nil,                    // regular file
		"link-file": ErrFilteredSpecialFile, // symlink to file
		"link-dir":  ErrFilteredSpecialFile, // symlink to dir
	}

	if len(got) != len(want) {
		t.Fatalf("got %d fn calls, want %d: %#v", len(got), len(want), got)
	}

	for _, c := range got {
		wantErr, ok := want[c.rel]
		if !ok {
			t.Fatalf("unexpected fn call for %q with err=%v", c.rel, c.err)
		}

		if wantErr == nil {
			if c.err != nil {
				t.Fatalf("fn(%q) err = %v, want nil", c.rel, c.err)
			}
			continue
		}

		if !errors.Is(c.err, wantErr) {
			t.Fatalf("fn(%q) err = %v, want %v", c.rel, c.err, wantErr)
		}
	}
}

func TestFilteredWalk_LinkNotExistIsNotAFailureCase(t *testing.T) {
	root := t.TempDir()

	linkToFileNotFound := filepath.Join(root, "link-file-not-found")
	if err := os.Symlink(
		filepath.Join(root, "does", "not", "exist123"), linkToFileNotFound); err != nil {
		t.Skipf("symlinks not supported on this system: %v", err)
	}

	type call struct {
		rel string
		err error
	}
	var got []call
	want := []call{
		{rel: ".", err: nil},
		{rel: "link-file-not-found", err: ErrFilteredSpecialFile},
	}

	fn := func(path string, d fs.DirEntry, err error) error {
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			t.Fatalf("filepath.Rel(%q, %q): %v", root, path, relErr)
		}
		got = append(got, call{rel: rel, err: err})
		return nil
	}

	err := FilteredWalk(root, Matcher{}, fn)
	assertNoErr(t, err)

	assertSliceEqual(t, got, want)
}
