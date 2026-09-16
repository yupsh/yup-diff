// Command yup-diff is the CLI wrapper around github.com/gloo-foo/cmd-diff.
package main

import (
	"bufio"

	clix "github.com/gloo-foo/cli"
	command "github.com/gloo-foo/cmd-diff"
	"github.com/spf13/afero"
	urf "github.com/urfave/cli/v3"
)

// version is the build version. It defaults to "dev" for local builds and is
// overridden at release time via the linker: -ldflags "-X main.version=<v>".
var version = "dev"

const (
	name        = "diff"
	flagUnified = "unified"
)

// Error is the sentinel error type for this package.
type Error string

func (e Error) Error() string { return string(e) }

// ErrOperandCount is raised when diff is not given exactly two file operands.
const ErrOperandCount Error = "diff takes exactly two FILE operands"

// synopsis is the multi-line --help usage block; urfave/cli indents it three
// spaces, so the lines stay flush-left.
const synopsis = `diff [OPTIONS] FILE1 FILE2

Compare FILE1 and FILE2 line by line.

With no options, produce ed-style output: ` + "`< line`" + ` for lines unique
to FILE1 and ` + "`> line`" + ` for lines unique to FILE2.`

// spec declares the diff wrapper: FILE1 streams as the pipeline source and
// FILE2 is read into memory as diff's second input, so the command is an
// ordinary filter over FILE1.
var spec = clix.Spec{
	Name:     name,
	Summary:  "compare two files line by line",
	Synopsis: synopsis,
	Build:    build,
	Flags: []urf.Flag{
		&urf.BoolFlag{
			Name:    flagUnified,
			Aliases: []string{"u"},
			Usage:   "output a unified diff",
			Sources: urf.EnvVars("YUP_DIFF_UNIFIED"),
		},
	},
}

// build maps the invocation to diff's pipeline: FILE1 feeds the source, FILE2 is
// read into diff's second input, and the diff command compares them. Anything
// other than exactly two FILE operands is a usage error.
func build(inv clix.Invocation) (clix.Source, clix.Command, error) {
	files := inv.Args.Args().Slice()
	if len(files) != 2 {
		return nil, nil, ErrOperandCount
	}
	input2, err := readLines(inv.Fs, clix.Path(files[1]))
	if err != nil {
		return nil, nil, err
	}
	source := clix.Files(inv.Fs, clix.Path(files[0]))
	return source, command.Diff(append([]any{input2}, options(inv.Args)...)...), nil
}

// readLines reads a file from fs and returns its lines as raw bytes for diff's
// second input, so file inputs flow through the injected filesystem rather than
// cmd-diff's hardcoded OS filesystem for positionals.
func readLines(fs afero.Fs, path clix.Path) (command.DiffInput, error) {
	f, err := fs.Open(string(path))
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

// options folds the parsed flags into diff's option values.
func options(c *urf.Command) []any {
	var opts []any
	if c.Bool(flagUnified) {
		opts = append(opts, command.DiffUnified)
	}
	return opts
}

// runMain is an indirection seam so main's wiring is testable without spawning
// the process; a test swaps it and restores it.
var runMain = clix.Main

func main() { runMain(spec, version) }
