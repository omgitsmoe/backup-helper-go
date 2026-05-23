# backup-helper-go

A CLI tool for managing checksum (hash) files. It automates common
checksumming workflows: collecting the most up-to-date hashes from
multiple checksum files, incrementally updating hashes for changed files
only, and identifying files missing checksums.

## Features

- **Incremental hashing** — Walk a directory tree and produce a `.cshd`
  file containing hashes for every file. Unchanged files can be skipped
  entirely (by matching stored mtime) or included verbatim without
  recomputation.
- **Most-current aggregation** — Discover all existing checksum files
  (`.cshd`, `.md5`, `.sha256`, etc.) under a root, merge them, and keep
  the newest entry for each file based on collection mtime.
- **Missing checksum detection** — Report which files and directories
  don't have a recorded hash yet.
- **Verification** — Verify a single checksum file or verify all hashes
  found under a directory root. Detects missing files, size mismatches,
  hash mismatches, corruption (same mtime, different hash), and stale
  hashes.
- **Glob-based filtering** — Allow/block patterns for both hash file
  discovery and all-file operations (using `doublestar` patterns).
- **Multiple hash algorithms** — MD5, SHA1, SHA224, SHA256, SHA384,
  SHA512, SHA3-224/256/384/512.
- **Progress reporting** — Rich terminal output showing file discovery,
  hashing progress, merge status, and verification results.

## Installation

```bash
go install github.com/omgitsmoe/backup-helper-go/cmd/checksum@latest
```

## Usage

```
checksum [global flags] <command> [command flags] [arguments]
```

### Commands

| Command | Description |
|---|---|
| `build` | Create a single `.cshd` file containing the most current hashes from all existing checksum files under a root. |
| `incremental` | Walk all files under a root, compute hashes for changed/new files, and write an incremental `.cshd` file. |
| `missing` | List files and directories that don't have a known checksum. |
| `fill` | (TODO) Generate checksums for files that don't have one yet. |
| `move` | (TODO) Move a hash file and update relative paths inside it. |
| `verify file` | Verify the entries in a single checksum file against what's on disk. |
| `verify root` | Verify all hashes found under a directory root. |

### Common flags

| Flag | Description |
|---|---|
| `--hash-type` | Hash algorithm to use for new hashes (default: `sha512`) |
| `-s, --skip-unchanged` | Skip files whose recorded mtime matches disk (no hashing) |
| `-i, --include-unchanged` | Include unchanged files in the output (default: `true`) |
| `--discover-hash-files-depth` | Max directory depth for discovering existing checksum files |
| `--keep-deleted` | Keep checksum entries for files no longer on disk |
| `--hash-allow` / `--hash-block` | Glob patterns to allow/block which checksum files are used as sources |
| `--all-allow` / `--all-block` | Glob patterns to allow/block files in all discovery operations |

### Examples

```bash
# Build a most-current checksum file from all existing hash files
checksum build /path/to/data

# Incrementally hash, skipping files with unchanged mtime
checksum incremental -s /path/to/data

# Find files without checksums
checksum missing /path/to/data

# Verify a single checksum file
checksum verify file /path/to/data/backup_2026-01-01T120000.cshd

# Verify all hashes under a root
checksum verify root /path/to/data
```

## File format — `.cshd`

The custom `.cshd` format is a line-based text format:

```
# version 1
<epoch_mtime>,<size>,<hash_type>,<hex_hash> <relative/path>
```

Example:

```
# version 1
1735689600.123456789,1024,sha512,abcdef...123456 src/main.go
```

Standard single-hash-per-file formats (`.md5`, `.sha256`, etc.) are also supported for reading:

```
<hex_hash> <relative/path>
```

## Project structure

```
├── cmd/checksum/          # CLI entry point
│   ├── main.go            # Command definitions (build, incremental, missing, fill, move, verify)
│   ├── progress.go        # Terminal progress reporter
│   └── rndpath.go         # Test-path generator
└── pkg/checksum/          # Library
    ├── checksum.go        # Checker type, Options, high-level API
    ├── collection.go      # HashCollection (merge, verify, iteration)
    ├── file.go            # File metadata, hashing, verification
    ├── hash.go            # Hash type wrapping crypto.Hash
    ├── incremental.go     # Incremental checksum generation
    ├── build.go           # Most-current aggregation
    ├── discover.go        # File discovery with matcher filtering
    ├── matcher.go         # Glob-based allow/block matcher
    ├── parse.go           # .cshd format parser
    ├── parse_single.go    # Standard <hash> <path> parser
    ├── serialize.go       # .cshd serializer
    └── progress.go        # Progress event types
```
