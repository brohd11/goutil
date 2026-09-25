// Package stream runs a subprocess and relays its output line by line to a Reporter.
package stream

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/brohd11/goutil/executil"
)

// Reporter is a sink for human-readable progress lines. A CLI prints them to stdout; a
// TUI funnels them into its log — the streamer only calls it, never assumes where they go.
type Reporter func(format string, args ...any)

// Cmd runs argv in dir (""=cwd) with env (nil=inherited), relaying interleaved
// stdout/stderr to report line by line. A non-zero exit returns an error carrying the
// last output line. Cancelling ctx kills the process.
func Cmd(ctx context.Context, dir string, env []string, report Reporter, argv ...string) error {
	if len(argv) == 0 {
		return fmt.Errorf("no command to run")
	}
	w := &lineWriter{report: report}
	cmd, err := executil.CommandContext(ctx, argv...)
	if err != nil {
		return err
	}
	if dir != "" {
		cmd.Dir = dir
	}
	if env != nil {
		cmd.Env = env
	}
	cmd.Stdout = w
	cmd.Stderr = w

	err = cmd.Run()
	w.flush() // a final line with no trailing newline (an error message often has none)
	if err != nil {
		if w.last != "" {
			return fmt.Errorf("%w: %s", err, w.last)
		}
		return err
	}
	return nil
}

// lineWriter splits a byte stream into lines on \r as well as \n (progress meters are
// \r-delimited), remembering the last line for error messages.
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
