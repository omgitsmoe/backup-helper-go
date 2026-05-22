package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/omgitsmoe/backup-helper-go/pkg/checksum"
)

type HashType string

const (
	HashTypeMd5      HashType = "md5"
	HashTypeSha1     HashType = "sha1"
	HashTypeSha224   HashType = "sha224"
	HashTypeSha256   HashType = "sha256"
	HashTypeSha384   HashType = "sha384"
	HashTypeSha512   HashType = "sha512"
	HashTypeSha3_224 HashType = "sha3_224"
	HashTypeSha3_256 HashType = "sha3_256"
	HashTypeSha3_384 HashType = "sha3_384"
	HashTypeSha3_512 HashType = "sha3_512"
)

func (h HashType) ToLib() checksum.Hash {
	libHash, err := checksum.FromIdentifier(string(h))
	if err != nil {
		panic("bug: HashType to checksum.HashType must succeed")
	}

	return libHash
}

type MatcherArgs struct {
	Allow []string
	Block []string
}

type MostCurrentArgs struct {
	DiscoverHashFilesDepth *uint
	KeepDeleted            bool
	HashFilesMatcher       MatcherArgs
}

type IncrementalArgs struct {
	Root                         string
	HashType                     HashType
	IncludeUnchanged             bool
	SkipUnchanged                bool
	PeriodicWriteIntervalSeconds *time.Duration
	MostCurrent                  MostCurrentArgs
	AllFilesMatcher              MatcherArgs
}

// ---- entry point ----

func main() {
	app := &cli.Command{
		Name:                  "checksum",
		Version:               "0.1.0",
		Usage:                 "Create, build, fill, move, and verify checksum files.",
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			incrementalCommand(),
			buildCommand(),
			missingCommand(),
			fillCommand(),
			moveCommand(),
			verifyCommand(),
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

// ---- command builders ----

func incrementalCommand() *cli.Command {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:  "hash-type",
			Value: string(HashTypeSha512),
			Usage: "Which hash type will be used for generating new hashes.",
		},
		&cli.BoolFlag{
			Name:    "include-unchanged",
			Aliases: []string{"i"},
			Usage:   "Whether to include unchanged files in the incremental checksum output file.",
		},
		&cli.BoolFlag{
			Name:    "skip-unchanged",
			Aliases: []string{"s"},
			Usage:   "Whether to skip a file based on the recorded modification time if it matches on disk.",
		},
		&cli.UintFlag{
			Name:  "periodic-write-interval-seconds",
			Usage: "Write current checksum entries every N seconds instead of only at the end.",
		},
	}
	flags = append(flags, mostCurrentFlags()...)
	flags = append(flags, allFilesMatcherFlags()...)

	return &cli.Command{
		Name:      "incremental",
		Usage:     "Creates an incremental checksum file.",
		ArgsUsage: "ROOT",
		Flags:     flags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args, err := parseIncrementalArgs(cmd)
			if err != nil {
				return err
			}
			return runIncremental(ctx, args)
		},
	}
}

func fillCommand() *cli.Command {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:  "hash-type",
			Value: string(HashTypeSha512),
			Usage: "Which hash type will be used for generating new hashes.",
		},
		&cli.BoolFlag{
			Name:    "include-unchanged",
			Aliases: []string{"i"},
		},
		&cli.BoolFlag{
			Name:    "skip-unchanged",
			Aliases: []string{"s"},
		},
		&cli.UintFlag{
			Name: "periodic-write-interval-seconds",
		},
	}

	flags = append(flags, mostCurrentFlags()...)
	flags = append(flags, allFilesMatcherFlags()...)

	return &cli.Command{
		Name:      "fill",
		Usage:     "Generate checksums for files that don't have one yet.",
		ArgsUsage: "ROOT",
		Flags:     flags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args, err := parseIncrementalArgs(cmd)
			if err != nil {
				return err
			}
			return runFill(ctx, args)
		},
	}
}

func buildCommand() *cli.Command {
	return &cli.Command{
		Name:      "build",
		Usage:     "Create one checksum file for the given root directory.",
		ArgsUsage: "ROOT",
		Flags:     mostCurrentFlags(),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			root, err := requiredArg(cmd, 0, "ROOT")
			if err != nil {
				return err
			}
			args := MostCurrentArgs{
				DiscoverHashFilesDepth: optionalUintFlag(cmd, "discover-hash-files-depth"),
				KeepDeleted:            cmd.Bool("keep-deleted"),
				HashFilesMatcher:       parseHashMatcher(cmd, "hash-allow", "hash-block"),
			}
			return runBuild(ctx, root, args)
		},
	}
}

func missingCommand() *cli.Command {
	return &cli.Command{
		Name:      "missing",
		Usage:     "Check for files that don't have a checksum yet.",
		ArgsUsage: "ROOT",
		Flags:     mostCurrentFlags(),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			root, err := requiredArg(cmd, 0, "ROOT")
			if err != nil {
				return err
			}
			args := MostCurrentArgs{
				DiscoverHashFilesDepth: optionalUintFlag(cmd, "discover-hash-files-depth"),
				KeepDeleted:            cmd.Bool("keep-deleted"),
				HashFilesMatcher:       parseHashMatcher(cmd, "hash-allow", "hash-block"),
			}
			return runMissing(ctx, root, args)
		},
	}
}

func moveCommand() *cli.Command {
	return &cli.Command{
		Name:      "move",
		Usage:     "Move a hash file and update relative paths inside it.",
		ArgsUsage: "SRC DST",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			src, err := requiredArg(cmd, 0, "SRC")
			if err != nil {
				return err
			}
			dst, err := requiredArg(cmd, 1, "DST")
			if err != nil {
				return err
			}
			return runMove(ctx, src, dst)
		},
	}
}

func verifyCommand() *cli.Command {
	return &cli.Command{
		Name:  "verify",
		Usage: "Subcommands for all verify operations.",
		Commands: []*cli.Command{
			verifyFileCommand(),
			verifyRootCommand(),
		},
	}
}

func verifyFileCommand() *cli.Command {
	return &cli.Command{
		Name:      "file",
		Usage:     "Verify a single hash file.",
		ArgsUsage: "PATH",
		Flags:     verifyMatcherFlags(),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			path, err := requiredArg(cmd, 0, "PATH")
			if err != nil {
				return err
			}
			args := parseVerifyMatcher(cmd)
			return runVerifyFile(ctx, path, args)
		},
	}
}

func verifyRootCommand() *cli.Command {
	return &cli.Command{
		Name:      "root",
		Usage:     "Verify hashes in a directory.",
		ArgsUsage: "ROOT",
		Flags: append(
			mostCurrentFlags(),
			verifyMatcherFlags()...,
		),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			root, err := requiredArg(cmd, 0, "ROOT")
			if err != nil {
				return err
			}

			mostCurrent := MostCurrentArgs{
				DiscoverHashFilesDepth: optionalUintFlag(cmd, "discover-hash-files-depth"),
				KeepDeleted:            cmd.Bool("keep-deleted"),
				HashFilesMatcher:       parseHashMatcher(cmd, "hash-allow", "hash-block"),
			}
			verifyMatcher := parseVerifyMatcher(cmd)

			return runVerifyRoot(ctx, root, mostCurrent, verifyMatcher)
		},
	}
}

// ---- flag groups ----

func mostCurrentFlags() []cli.Flag {
	return []cli.Flag{
		&cli.UintFlag{
			Name:  "discover-hash-files-depth",
			Usage: "Maximum directory depth for discovering checksum files recursively.",
		},
		&cli.BoolFlag{
			Name:  "keep-deleted",
			Usage: "Keep checksum entries for files that no longer exist on disk.",
		},
		&cli.StringSliceFlag{
			Name:  "hash-allow",
			Usage: "Glob patterns for files allowed as checksum sources. Repeatable.",
		},
		&cli.StringSliceFlag{
			Name:  "hash-block",
			Usage: "Glob patterns for checksum files to exclude. Repeatable.",
		},
	}
}

func allFilesMatcherFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringSliceFlag{
			Name:  "all-allow",
			Usage: "Glob patterns for files included in all discovery operations. Repeatable.",
		},
		&cli.StringSliceFlag{
			Name:  "all-block",
			Usage: "Glob patterns for files excluded from all discovery operations. Repeatable.",
		},
	}
}

func verifyMatcherFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringSliceFlag{
			Name:  "verify-allow",
			Usage: "Glob patterns for files included in verify operations. Repeatable.",
		},
		&cli.StringSliceFlag{
			Name:  "verify-block",
			Usage: "Glob patterns for files excluded from verify operations. Repeatable.",
		},
	}
}

// ---- parsing helpers ----

func parseIncrementalArgs(cmd *cli.Command) (IncrementalArgs, error) {
	root, err := requiredArg(cmd, 0, "ROOT")
	if err != nil {
		return IncrementalArgs{}, err
	}

	hashType, err := parseHashType(cmd.String("hash-type"))
	if err != nil {
		return IncrementalArgs{}, err
	}

	var periodic *time.Duration
	if cmd.IsSet("periodic-write-interval-seconds") {
		d := time.Duration(cmd.Uint("periodic-write-interval-seconds")) * time.Second
		periodic = &d
	}

	return IncrementalArgs{
		Root:                         root,
		HashType:                     hashType,
		IncludeUnchanged:             cmd.Bool("include-unchanged"),
		SkipUnchanged:                cmd.Bool("skip-unchanged"),
		PeriodicWriteIntervalSeconds: periodic,
		MostCurrent: MostCurrentArgs{
			DiscoverHashFilesDepth: optionalUintFlag(cmd, "discover-hash-files-depth"),
			KeepDeleted:            cmd.Bool("keep-deleted"),
			HashFilesMatcher:       parseHashMatcher(cmd, "hash-allow", "hash-block"),
		},
		AllFilesMatcher: parseAllMatcher(cmd),
	}, nil
}

func parseHashType(v string) (HashType, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "md5":
		return HashTypeMd5, nil
	case "sha1":
		return HashTypeSha1, nil
	case "sha224":
		return HashTypeSha224, nil
	case "sha256":
		return HashTypeSha256, nil
	case "sha384":
		return HashTypeSha384, nil
	case "sha512":
		return HashTypeSha512, nil
	case "sha3_224", "sha3-224":
		return HashTypeSha3_224, nil
	case "sha3_256", "sha3-256":
		return HashTypeSha3_256, nil
	case "sha3_384", "sha3-384":
		return HashTypeSha3_384, nil
	case "sha3_512", "sha3-512":
		return HashTypeSha3_512, nil
	default:
		return "", fmt.Errorf("invalid hash type %q", v)
	}
}

func parseHashMatcher(cmd *cli.Command, allowName, blockName string) MatcherArgs {
	return MatcherArgs{
		Allow: cmd.StringSlice(allowName),
		Block: cmd.StringSlice(blockName),
	}
}

func parseAllMatcher(cmd *cli.Command) MatcherArgs {
	return MatcherArgs{
		Allow: cmd.StringSlice("all-allow"),
		Block: cmd.StringSlice("all-block"),
	}
}

func parseVerifyMatcher(cmd *cli.Command) MatcherArgs {
	return MatcherArgs{
		Allow: cmd.StringSlice("verify-allow"),
		Block: cmd.StringSlice("verify-block"),
	}
}

func optionalUintFlag(cmd *cli.Command, name string) *uint {
	if !cmd.IsSet(name) {
		return nil
	}
	v := cmd.Uint(name)
	return &v
}

func requiredArg(cmd *cli.Command, idx int, label string) (string, error) {
	arg := cmd.Args().Get(idx)
	if strings.TrimSpace(arg) == "" {
		return "", fmt.Errorf("missing required argument: %s", label)
	}
	return arg, nil
}

func makeMatcher(args MatcherArgs) checksum.Matcher {
	matcherOptions := []checksum.MatcherOption{}
	for _, p := range args.Allow {
		matcherOptions = append(matcherOptions, checksum.WithAllow(p))
	}
	for _, p := range args.Block {
		matcherOptions = append(matcherOptions, checksum.WithBlock(p))
	}
	matcher, err := checksum.NewMatcher(matcherOptions...)
	if err != nil {
		log.Fatalf("failed to create all files matcher: %s", err)
	}

	return matcher
}

func applyIncrementalArgs(options *checksum.Options, args IncrementalArgs) {
	options.HashType = args.HashType.ToLib()
	options.IncrementalIncludeUnchangedFiles = args.IncludeUnchanged
	options.IncrementalSkipUnchanged = args.SkipUnchanged
	if args.PeriodicWriteIntervalSeconds != nil {
		// why ptr?
		options.IncrementalPeriodicWriteInterval = *args.PeriodicWriteIntervalSeconds
	}
	options.AllFilesMatcher = makeMatcher(args.AllFilesMatcher)
	applyMostCurrentArgs(options, args.MostCurrent)
}

func applyMostCurrentArgs(options *checksum.Options, args MostCurrentArgs) {
	if args.DiscoverHashFilesDepth != nil {
		// why ptr?
		options.DiscoverHashFilesDepth = int(*args.DiscoverHashFilesDepth)
	}
	options.MostCurrentFilterDeleted = !args.KeepDeleted
	options.HashFilesMatcher = makeMatcher(args.HashFilesMatcher)
}

func runIncremental(_ context.Context, args IncrementalArgs) error {
	options := checksum.DefaultOptions()
	applyIncrementalArgs(&options, args)

	checker, err := checksum.NewCheckerWithOptions(args.Root, options)
	if err != nil {
		log.Fatalf("failed to create checker: %s", err)
	}

	reporter := NewProgressReporter()
	inc, err := checker.Incremental(func(p checksum.ProgressEvent) {
		reporter.Report(p)
	})
	if err != nil {
		fmt.Printf("incremental failed: %s\n", err)
		os.Exit(1)
	}

	path, err := inc.Path()
	if err != nil {
		fmt.Printf("incremental failed: %s\n", err)
		os.Exit(1)
	}

	f, err := os.Create(path)
	if err != nil {
		fmt.Printf("incremental failed to create file: %s\n", err)
		os.Exit(1)
	}
	defer f.Close()

	ser := checksum.NewSerializer(f)
	err = ser.Flush(inc)
	if err != nil {
		fmt.Printf("incremental failed to write file: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote file at %q", path)

	return nil
}

func runFill(_ context.Context, _ IncrementalArgs) error {
	return fmt.Errorf("not implemented TODO")
}

func runBuild(_ context.Context, root string, args MostCurrentArgs) error {
	options := checksum.DefaultOptions()
	applyMostCurrentArgs(&options, args)

	checker, err := checksum.NewCheckerWithOptions(root, options)
	if err != nil {
		log.Fatalf("failed to create checker: %s", err)
	}

	reporter := NewProgressReporter()
	mostCurrent, err := checker.BuildMostCurrent(func(p checksum.ProgressEvent) {
		reporter.Report(p)
	})
	if err != nil {
		fmt.Printf("buildMostCurrent failed: %s\n", err)
		os.Exit(1)
	}

	path, err := mostCurrent.Path()
	if err != nil {
		fmt.Printf("buildMostCurrent failed: %s\n", err)
		os.Exit(1)
	}

	f, err := os.Create(path)
	if err != nil {
		fmt.Printf("buildMostCurrent failed to create file: %s\n", err)
		os.Exit(1)
	}
	defer f.Close()

	ser := checksum.NewSerializer(f)
	err = ser.Flush(mostCurrent)
	if err != nil {
		fmt.Printf("buildMostCurrent failed to write file: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote file at %q", path)

	return nil
}

func runMissing(_ context.Context, root string, args MostCurrentArgs) error {
	options := checksum.DefaultOptions()
	applyMostCurrentArgs(&options, args)

	checker, err := checksum.NewCheckerWithOptions(root, options)
	if err != nil {
		log.Fatalf("failed to create checker: %s", err)
	}

	reporter := NewProgressReporter()

	missing, err := checker.CheckMissing(func(p checksum.ProgressEvent) {
		reporter.Report(p)
	})
	if err != nil {
		fmt.Printf("incremental failed: %s\n", err)
		os.Exit(1)
	}

	if len(missing.Directories) > 0 {
		fmt.Println("Directories that are completely missing:")
		for _, d := range missing.Directories {
			fmt.Printf("\t%q\n", d)
		}
	}
	if len(missing.Files) > 0 {
		fmt.Println("Files that are missing:")
		for _, f := range missing.Files {
			fmt.Printf("\t%q\n", f)
		}
	}

	if len(missing.Files) == 0 && len(missing.Directories) == 0 {
		fmt.Println("Success! All files have a known hash! (No mtime check was made!)")
	}

	return nil
}

func runMove(_ context.Context, _ string, _ string) error {
	return fmt.Errorf("not implemented TODO")
}

func runVerifyFile(_ context.Context, path string, _ MatcherArgs) error {
	// TODO use MatcherArgs in verify include
	options := checksum.DefaultOptions()
	root := filepath.Dir(path)
	checker, err := checksum.NewCheckerWithOptions(root, options)
	if err != nil {
		log.Fatalf("failed to create checker: %s", err)
	}

	collection, err := checker.Read(path)
	if err != nil {
		fmt.Printf("Failed to read collection: %s\n", err)
		os.Exit(1)
	}

	reporter := NewProgressReporter()

	err = checker.Verify(collection, nil, func(p checksum.VerifyProgress) bool {
		reporter.ReportVerify(&p)

		return true
	})

	if err != nil {
		fmt.Printf("Verify failed: %s\n", err)
		os.Exit(1)
	}
	return nil
}

func runVerifyRoot(_ context.Context, root string, args MostCurrentArgs, _ MatcherArgs) error {
	// TODO use MatcherArgs in verify include
	options := checksum.DefaultOptions()
	applyMostCurrentArgs(&options, args)

	checker, err := checksum.NewCheckerWithOptions(root, options)
	if err != nil {
		log.Fatalf("failed to create checker: %s", err)
	}

	reporter := NewProgressReporter()

	err = checker.VerifyRoot(nil, func(p checksum.ProgressEvent) {
		reporter.Report(p)
	})

	return err
}
