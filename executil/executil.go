// Package executil builds subprocesses with the same argv-oriented interface as
// os/exec while also handling Windows batch-file launchers.
package executil

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Command builds a subprocess for argv. Unlike exec.Command it can fail: on Windows the
// program must be resolved to decide whether a .cmd/.bat needs the command processor.
func Command(argv ...string) (*exec.Cmd, error) {
	return CommandContext(context.Background(), argv...)
}

// CommandContext is Command with cancellation. On Windows, .cmd and .bat programs run
// through COMSPEC because CreateProcess cannot execute batch files.
func CommandContext(ctx context.Context, argv ...string) (*exec.Cmd, error) {
	if len(argv) == 0 || strings.TrimSpace(argv[0]) == "" {
		return nil, fmt.Errorf("no command to run")
	}
	return commandContext(ctx, argv)
}
