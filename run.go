package main

import (
	"bufio"
	"context"
	"fmt"
	"io"

	command "github.com/gloo-foo/cmd-diff"
	gloo "github.com/gloo-foo/framework"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
)

const flagUnified = "unified"

// usageText is the command's multi-line usage synopsis, shown in --help.
// cli/v3 indents the whole block by 3 spaces, so these lines are flush-left to
// stay aligned in the rendered output.
const usageText = `diff [OPTIONS] FILE1 FILE2

Compare FILE1 and FILE2 line by line.

With no options, produce ed-style output: ` + "`< line`" + ` for lines unique
to FILE1 and ` + "`> line`" + ` for lines unique to FILE2.`

// Error is the sentinel error type for this package.
type Error string

func (e Error) Error() string { return string(e) }

// ErrOperandCount is raised when diff is not given exactly two file operands.
const ErrOperandCount Error = "diff takes exactly two FILE operands"

// init replaces urfave/cli's default --version/-v flag with a --version-only
// flag, freeing the single-letter -v for command flags while still exposing
// the injected build version.
func init() {
	cli.VersionFlag = &cli.BoolFlag{Name: "version", Usage: "print version information and exit"}
}

// run builds and executes the diff CLI against the injected version, I/O, and
// filesystem, returning the process exit code.
func run(version string, args []string, _ io.Reader, stdout, stderr io.Writer, fs afero.Fs) int {
	cmd := newApp(version, stdout, fs)
	cmd.Writer = stdout
	cmd.ErrWriter = stderr
	if err := cmd.Run(context.Background(), args); err != nil {
		_, _ = fmt.Fprintf(stderr, "diff: %v\n", err)
		return 1
	}
	return 0
}

func newApp(version string, stdout io.Writer, fs afero.Fs) *cli.Command {
	return &cli.Command{
		Name:            "diff",
		Version:         version,
		Usage:           "compare two files line by line",
		UsageText:       usageText,
		HideHelpCommand: true,
		// Keep exit handling in run() rather than letting urfave/cli call
		// os.Exit, so the exit code stays testable.
		ExitErrHandler: func(context.Context, *cli.Command, error) {},
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: flagUnified, Aliases: []string{"u"}, Usage: "output a unified diff"},
		},
		Action: action(stdout, fs),
	}
}

func action(stdout io.Writer, fs afero.Fs) cli.ActionFunc {
	return func(_ context.Context, c *cli.Command) error {
		if c.NArg() != 2 {
			return ErrOperandCount
		}
		input2, err := readLines(fs, c.Args().Get(1))
		if err != nil {
			return err
		}
		source := gloo.ByteFileSource(fs, []gloo.File{gloo.File(c.Args().Get(0))})
		opts := append([]any{input2}, options(c)...)
		_, err = gloo.Run(source, gloo.ByteWriteTo(stdout), command.Diff(opts...))
		return err
	}
}

// readLines reads a file from fs and returns its lines as raw bytes for the
// second diff input, so file inputs flow through the injected filesystem
// rather than cmd-diff's hardcoded OS filesystem for positionals.
func readLines(fs afero.Fs, name string) (command.DiffInput, error) {
	f, err := fs.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	scanner := bufio.NewScanner(f)
	var lines command.DiffInput
	for scanner.Scan() {
		lines = append(lines, append([]byte(nil), scanner.Bytes()...))
	}
	return lines, scanner.Err()
}

func options(c *cli.Command) []any {
	var opts []any
	if c.Bool(flagUnified) {
		opts = append(opts, command.DiffUnified)
	}
	return opts
}
