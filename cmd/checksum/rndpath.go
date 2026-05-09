package main

import (
	"math/rand"
	"strings"
	"time"
)

var commonDirs = []string{
	"src", "pkg", "internal", "cmd", "lib",
	"build", "dist", "node_modules", "vendor",
	"test", "tests", "data", "assets",
}

var commonFiles = []string{
	"main.go", "index.js", "app.ts", "utils.go",
	"README.md", "config.json", "test.go",
	"handler.go", "server.go", "client.go",
}

const letters = "abcdefghijklmnopqrstuvwxyz"

func randName(r *rand.Rand, min, max int) string {
	n := r.Intn(max-min+1) + min
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return string(b)
}

// Zipf distribution for realistic skew
func newZipf(r *rand.Rand, n uint64) *rand.Zipf {
	return rand.NewZipf(r, 1.2, 1.0, n-1)
}

func generatePaths(n int) []string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	dirZipf := newZipf(r, uint64(len(commonDirs)))
	fileZipf := newZipf(r, uint64(len(commonFiles)))

	paths := make([]string, 0, n)

	for i := 0; i < n; i++ {
		// depth: mostly shallow, sometimes deep
		depth := 2 + r.Intn(4)
		if r.Float64() < 0.1 {
			depth += r.Intn(10) // rare deep paths
		}

		parts := make([]string, 0, depth+1)
		// from root
		parts = append(parts, "")

		// build directories
		for j := 0; j < depth; j++ {
			if r.Float64() < 0.7 {
				parts = append(parts, commonDirs[dirZipf.Uint64()])
			} else {
				parts = append(parts, randName(r, 3, 10))
			}
		}

		// file
		if r.Float64() < 0.7 {
			parts = append(parts, commonFiles[fileZipf.Uint64()])
		} else {
			parts = append(parts, randName(r, 5, 12)+".go")
		}

		paths = append(paths, strings.Join(parts, "/"))
	}

	return paths
}

func mutate(p string, r *rand.Rand) string {
	if r.Float64() < 0.3 {
		return p + "/extra/" + randName(r, 4, 8)
	}
	if r.Float64() < 0.3 {
		return "copy/" + p
	}
	return p
}
