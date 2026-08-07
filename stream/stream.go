// Package stream runs a subprocess and relays its output to a caller-supplied reporter one
// line at a time as it arrives. It's a domain-neutral port of what was gitstack's GitStream
// (and its near-verbatim twin in golaunch): the streaming machinery knows nothing about the
// program being run — git, a build script, any command — so it can back any tool that
// executes a process and wants its progress surfaced live in a log pane or on stdout.
package stream

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Reporter is a sink for human-readable progress lines. A CLI prints them to stdout; a
// TUI funnels them into its log — the streamer only calls it, never assumes where they go.
type Reporter func(format string, args ...any)

// Cmd runs argv (argv[0] is the program) in dir, relaying its stdout+stderr to report one
// line at a time as it arrives, and returns a non-nil error when the program exits non-zero
// (with its last line of output folded in, the part usually worth reading). stdout and
// stderr are interleaved the way a terminal would show them — a program often writes
// progress and errors to stderr, and a caller streaming to a log wants both.
//
// dir "" means the current working directory; env nil means the process's own environment
// (both are just exec's defaults, left alone). ctx cancellation kills the subprocess, which
// is how a TUI's task-abort works.
func Cmd(ctx context.Context, dir string, env []string, report Reporter, argv ...string) error {
	if len(argv) == 0 {
		return fmt.Errorf("no command to run")
	}
	w := &lineWriter{report: report}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	if dir != "" {
		cmd.Dir = dir
	}
	if env != nil {
		cmd.Env = env
	}
	cmd.Stdout = w
	cmd.Stderr = w

	err := cmd.Run()
	w.flush() // a final line with no trailing newline (an error message often has none)
	if err != nil {
		if w.last != "" {
			return fmt.Errorf("%w: %s", err, w.last)
		}
		return err
	}
	return nil
}

// lineWriter turns a subprocess's byte stream into whole lines for a Reporter, breaking on
// \r as well as \n: progress bars ("Receiving objects:  47%…") are carriage-return
// delimited, so otherwise a whole progress meter would arrive as one unreadable line. It
// keeps the last line reported so a failing command can quote the program's parting words
// in its error.
type lineWriter struct {
	report Reporter
	buf    []byte
	last   string
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexAny(w.buf, "\r\n")
		if i < 0 {
			break
		}
		w.emit(string(w.buf[:i]))
		w.buf = w.buf[i+1:]
	}
	return len(p), nil
}

// flush emits whatever partial line is left once the command has exited.
func (w *lineWriter) flush() {
	if len(w.buf) > 0 {
		w.emit(string(w.buf))
		w.buf = nil
	}
}

func (w *lineWriter) emit(line string) {
	line = strings.TrimRight(line, " \t")
	if line == "" {
		return
	}
	w.last = line
	// "%s" rather than the line as a format string: output can contain a literal %.
	w.report("%s", line)
}
