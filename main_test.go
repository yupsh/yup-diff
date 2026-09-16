package main

import (
	"context"
	"errors"
	"testing"

	clix "github.com/gloo-foo/cli"
	"github.com/spf13/afero"
	urf "github.com/urfave/cli/v3"
)

// parse runs args through a bare command carrying the wrapper's flags and
// returns the parsed accessor.
func parse(t *testing.T, args ...string) *urf.Command {
	t.Helper()
	var got *urf.Command
	app := &urf.Command{
		Name:   name,
		Flags:  spec.Flags,
		Action: func(_ context.Context, c *urf.Command) error { got = c; return nil },
	}
	if err := app.Run(context.Background(), args); err != nil {
		t.Fatalf("parse: %v", err)
	}
	return got
}

func seeded(t *testing.T) afero.Fs {
	t.Helper()
	fs := afero.NewMemMapFs()
	if err := afero.WriteFile(fs, "a.txt", []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatalf("seed a: %v", err)
	}
	if err := afero.WriteFile(fs, "b.txt", []byte("one\nthree\n"), 0o644); err != nil {
		t.Fatalf("seed b: %v", err)
	}
	return fs
}

func TestReadLines(t *testing.T) {
	lines, err := readLines(seeded(t), "a.txt")
	if err != nil {
		t.Fatalf("readLines: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("lines=%d, want 2", len(lines))
	}
}

func TestReadLines_MissingFile(t *testing.T) {
	if _, err := readLines(afero.NewMemMapFs(), "nope.txt"); err == nil {
		t.Fatal("readLines: want error for missing file")
	}
}

func TestOptions(t *testing.T) {
	if got := len(options(parse(t, name))); got != 0 {
		t.Fatalf("options len=%d, want 0", got)
	}
	if got := len(options(parse(t, name, "-u"))); got != 1 {
		t.Fatalf("options len=%d, want 1", got)
	}
}

func TestBuild_Filter(t *testing.T) {
	inv := clix.Invocation{Args: parse(t, name, "a.txt", "b.txt"), Fs: seeded(t)}
	src, filter, err := build(inv)
	if err != nil || src == nil || filter == nil {
		t.Fatalf("build: src=%v filter=%v err=%v", src, filter, err)
	}
}

func TestBuild_OperandCount(t *testing.T) {
	inv := clix.Invocation{Args: parse(t, name, "only.txt"), Fs: seeded(t)}
	src, filter, err := build(inv)
	if !errors.Is(err, ErrOperandCount) {
		t.Fatalf("err=%v, want ErrOperandCount", err)
	}
	if src != nil || filter != nil {
		t.Fatalf("src=%v filter=%v, want both nil on error", src, filter)
	}
	if err.Error() != string(ErrOperandCount) {
		t.Fatalf("message=%q, want %q", err.Error(), string(ErrOperandCount))
	}
}

func TestBuild_ReadError(t *testing.T) {
	inv := clix.Invocation{Args: parse(t, name, "a.txt", "missing.txt"), Fs: afero.NewMemMapFs()}
	src, filter, err := build(inv)
	if err == nil || src != nil || filter != nil {
		t.Fatalf("build: src=%v filter=%v err=%v, want read error", src, filter, err)
	}
}

func Test_main(t *testing.T) {
	orig := runMain
	t.Cleanup(func() { runMain = orig })
	var gotName clix.Name
	runMain = func(s clix.Spec, _ clix.Version) { gotName = s.Name }
	main()
	if gotName != name {
		t.Fatalf("main used spec %q, want %s", gotName, name)
	}
}
