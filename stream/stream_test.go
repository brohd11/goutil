package stream

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// collect returns a Reporter that appends every reported line to lines. Each call carries
// one line as "%s", so formatting it back out recovers exactly what the streamer emitted.
func collect(lines *[]string) Reporter {
	return func(format string, args ...any) {
		*lines = append(*lines, fmt.Sprintf(format, args...))
	}
}

func TestCmdMultiLineOutput(t *testing.T) {
	var lines []string
	err := Cmd(context.Background(), "", nil, collect(&lines), "sh", "-c", "printf 'one\ntwo\nthree\n'")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"one", "two", "three"}
	if strings.Join(lines, ",") != strings.Join(want, ",") {
		t.Errorf("got %v, want %v", lines, want)
	}
}

func TestCmdCarriageReturnProgress(t *testing.T) {
	var lines []string
	// Pass printable escape sequences to the shell and let printf produce the
	// carriage-return bytes. Literal CRs in a Windows command line can be
	// consumed while Git's sh translates its argv.
	err := Cmd(context.Background(), "", nil, collect(&lines), "sh", "-c", "printf '10%%\\r50%%\\r100%%\\r'")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"10%", "50%", "100%"}
	if strings.Join(lines, ",") != strings.Join(want, ",") {
		t.Errorf("got %v, want %v", lines, want)
	}
}

func TestCmdDropsEmptyLinesAndTrims(t *testing.T) {
	var lines []string
	err := Cmd(context.Background(), "", nil, collect(&lines), "sh", "-c", "printf 'a\n\n   \nb  \t\n'")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a", "b"}
	if strings.Join(lines, ",") != strings.Join(want, ",") {
		t.Errorf("got %v, want %v", lines, want)
	}
}

func TestCmdFlushesUnterminatedLine(t *testing.T) {
	var lines []string
	err := Cmd(context.Background(), "", nil, collect(&lines), "sh", "-c", "printf 'done\nno newline at end'")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"done", "no newline at end"}
	if strings.Join(lines, ",") != strings.Join(want, ",") {
		t.Errorf("got %v, want %v", lines, want)
	}
}

func TestCmdFailureFoldsLastLine(t *testing.T) {
	var lines []string
	err := Cmd(context.Background(), "", nil, collect(&lines), "sh", "-c", "echo first; echo 'fatal: broke' >&2; exit 3")
	if err == nil {
		t.Fatal("want an error on non-zero exit")
	}
	if !strings.Contains(err.Error(), "exit status 3") || !strings.HasSuffix(err.Error(), ": fatal: broke") {
		t.Errorf("got %q, want exit status with the last output line folded in", err)
	}
}

func TestCmdEmptyArgv(t *testing.T) {
	err := Cmd(context.Background(), "", nil, func(string, ...any) {})
	if err == nil || err.Error() != "no command to run" {
		t.Errorf("got %v, want \"no command to run\"", err)
	}
}
