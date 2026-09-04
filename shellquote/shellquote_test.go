package shellquote

import (
	"os/exec"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// The nasty cases: an embedded single quote (the one thing a single-quoted literal
// cannot hold directly), an empty argument (which must survive as an argument), and the
// metacharacters that would otherwise expand or split.
var cases = []string{
	"plain",
	"",
	"with space",
	"it's",
	"'leading",
	"trailing'",
	"''",
	`a"b`,
	`back\slash`,
	"$HOME",
	"$(rm -rf /)",
	"`whoami`",
	"a*b?c[d]",
	"semi;colon",
	"pipe|amp&",
	"new\nline",
	"tab\there",
	"~/path",
	"key=value",
	"a,b@c%d+e",
}

// The real test of a quoter is that a real shell hands the words back unchanged. Both
// spellings must satisfy this -- QuoteMinimal is an readability optimization, never a
// correctness exception.
func TestRoundTripsThroughRealShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell quoting is exercised on Unix runners")
	}
	for _, quote := range []struct {
		name string
		fn   func(string) string
	}{{"Quote", Quote}, {"QuoteMinimal", QuoteMinimal}} {
		for _, in := range cases {
			// printf %s each arg with a NUL terminator so the boundaries are
			// unambiguous even when an argument contains a newline.
			line := "printf '%s\\0' " + quote.fn(in)
			out, err := exec.Command("sh", "-c", line).Output()
			if err != nil {
				t.Fatalf("%s(%q): sh failed: %v", quote.name, in, err)
			}
			got := strings.TrimSuffix(string(out), "\x00")
			if got != in {
				t.Errorf("%s(%q) -> sh gave back %q", quote.name, in, got)
			}
		}
	}
}

// Join must preserve argument boundaries, not just contents: the count is the half that
// silently breaks when an empty or space-bearing argument is mishandled.
func TestJoinPreservesArgumentBoundaries(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell quoting is exercised on Unix runners")
	}
	for _, join := range []struct {
		name string
		fn   func([]string) string
	}{{"Join", Join}, {"JoinMinimal", JoinMinimal}} {
		line := "printf '%s\\0' " + join.fn(cases)
		out, err := exec.Command("sh", "-c", line).Output()
		if err != nil {
			t.Fatalf("%s: sh failed: %v", join.name, err)
		}
		got := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
		if len(got) != len(cases) {
			t.Fatalf("%s produced %d args, want %d: %q", join.name, len(got), len(cases), got)
		}
		for i := range cases {
			if got[i] != cases[i] {
				t.Errorf("%s arg %d = %q, want %q", join.name, i, got[i], cases[i])
			}
		}
	}
}

// The point of the Minimal spelling: ordinary words stay readable.
func TestQuoteMinimalLeavesSafeWordsBare(t *testing.T) {
	for _, s := range []string{"plain", "key=value", "a,b@c%d+e", "-flag", "/usr/bin/env"} {
		if got := QuoteMinimal(s); got != s {
			t.Errorf("QuoteMinimal(%q) = %q, want it left bare", s, got)
		}
	}
	// "~" is deliberately NOT in the safe set: bare, the shell would expand it to the
	// home directory, so an argv that literally contains "~/path" needs the quotes.
	if got := QuoteMinimal("~/path"); got != "'~/path'" {
		t.Errorf("QuoteMinimal(%q) = %q, want it quoted against tilde expansion", "~/path", got)
	}
	// ...but an empty argument must still be quoted, or it disappears entirely.
	if got := QuoteMinimal(""); got != "''" {
		t.Errorf(`QuoteMinimal("") = %q, want "''"`, got)
	}
}

// TestSplit is Join's inverse read back: the $EDITOR forms a user actually writes,
// and the one malformed input the parser is allowed to refuse.
func TestSplit(t *testing.T) {
	tests := map[string][]string{
		`code --wait`:                        {"code", "--wait"},
		`"/path with spaces/editor" --wait`:  {"/path with spaces/editor", "--wait"},
		`editor 'two words' plain`:           {"editor", "two words", "plain"},
		`"C:\Program Files\Editor\edit.exe"`: {`C:\Program Files\Editor\edit.exe`},
		`editor\ command`:                    {"editor command"},
	}
	for raw, want := range tests {
		got, err := Split(raw)
		if err != nil {
			t.Fatalf("Split(%q): %v", raw, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Split(%q) = %#v, want %#v", raw, got, want)
		}
	}
	if _, err := Split(`editor "unfinished`); err == nil {
		t.Fatal("unterminated quote should fail")
	}
}
