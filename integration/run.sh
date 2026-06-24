#!/bin/sh
# Integration checks for yup-diff, run inside a Debian (GNU diffutils) container.
#
# yup-diff's output format does NOT match GNU `diff`: it compares positionally
# (line N vs line N, no LCS edit script), its default is ed-style `< `/`> `
# rows, and its -u emits a single `@@ -1,N +1,M @@` hunk spanning both files.
# GNU diff has no mode producing those exact bytes, so every case is an `assert`
# against yup-diff's own documented contract (see cmd-diff COMPATIBILITY.md).
# The one place both agree — identical files produce no output and exit 0 — is
# checked with `parity`.
#
# parity ARGS  — yup-diff and GNU `diff` must produce byte-identical output.
# assert WANT  — yup-diff must produce WANT exactly (the documented divergences).
#
# diff's exit status is 1 when the files differ (not an error), so both helpers
# tolerate a non-zero exit and compare stdout only.
set -eu

export LC_ALL=C

fails=0
work=$(mktemp -d)
f1=$work/f1
f2=$work/f2

parity() {
	ours=$(yup-diff "$@" 2>/dev/null || true)
	gnu=$(diff "$@" 2>/dev/null || true)
	if [ "$ours" = "$gnu" ]; then
		printf 'ok    parity  diff %s\n' "$*"
	else
		printf 'FAIL  parity  diff %s\n        gnu:  %s\n        ours: %s\n' "$*" "$gnu" "$ours"
		fails=$((fails + 1))
	fi
}

assert() {
	want=$1
	shift
	got=$(yup-diff "$@" 2>/dev/null || true)
	if [ "$got" = "$want" ]; then
		printf 'ok    assert  diff %s\n' "$*"
	else
		printf 'FAIL  assert  diff %s\n        want: %s\n        got:  %s\n' "$*" "$want" "$got"
		fails=$((fails + 1))
	fi
}

# Identical files: no output, exit 0 — the one mode that matches GNU diff.
printf 'apple\nbanana\ncherry\n' >"$f1"
cp "$f1" "$f2"
parity "$f1" "$f2"

# Default ed-style output, positional comparison: line N of FILE1 vs line N of
# FILE2. Each differing position emits `< ` (FILE1) then `> ` (FILE2).
printf 'apple\nbanana\ncherry\n' >"$f1"
printf 'banana\ncherry\nkiwi\n' >"$f2"
assert "$(printf '< apple\n> banana\n< banana\n> cherry\n< cherry\n> kiwi')" "$f1" "$f2"

# A single changed line in the middle: only that position differs.
printf 'a\nb\nc\n' >"$f1"
printf 'a\nx\nc\n' >"$f2"
assert "$(printf '< b\n> x')" "$f1" "$f2"

# FILE1 longer: its trailing line has no counterpart, emits only `< `.
printf 'a\nb\n' >"$f1"
printf 'a\n' >"$f2"
assert '< b' "$f1" "$f2"

# FILE2 longer: its trailing line has no counterpart, emits only `> `.
printf 'a\n' >"$f1"
printf 'a\nb\n' >"$f2"
assert '> b' "$f1" "$f2"

# Unified (-u): `--- a` / `+++ b` headers, one `@@ -1,N +1,M @@` hunk, then
# ` `/`-`/`+` body rows for the whole span.
printf 'a\nb\nc\n' >"$f1"
printf 'a\nx\nc\n' >"$f2"
assert "$(printf -- '--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+x\n c')" -u "$f1" "$f2"

# Unified, long form (--unified) behaves identically to -u.
assert "$(printf -- '--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+x\n c')" --unified "$f1" "$f2"

# Unified, FILE1 longer: trailing FILE1 line shows as a `-` row.
printf 'a\nb\n' >"$f1"
printf 'a\n' >"$f2"
assert "$(printf -- '--- a\n+++ b\n@@ -1,2 +1,1 @@\n a\n-b')" -u "$f1" "$f2"

# Unified, identical files: no output.
cp "$f1" "$f1.same"
assert '' -u "$f1" "$f1"

if [ "$fails" -ne 0 ]; then
	printf '\n%s check(s) failed\n' "$fails"
	exit 1
fi
printf '\nall checks passed\n'
