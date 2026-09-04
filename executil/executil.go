// Package executil builds subprocesses with the same argv-oriented interface as
// os/exec while also handling Windows batch-file launchers.
package executil

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Command builds a subprocess for argv. Unlike exec.Command, construction can
// fail: Windows has to resolve the program before it can decide whether a .cmd
// or .bat wrapper must be run through the command processor.
func Command(argv ...string) (*exec.Cmd, error) {
	return CommandContext(context.Background(), argv...)
}

// CommandContext is Command with cancellation. Native executables are launched
// directly. On Windows, .cmd and .bat programs are routed through COMSPEC because
// CreateProcess cannot execute batch files itself.
func CommandContext(ctx context.Context, argv ...string) (*exec.Cmd, error) {
	if len(argv) == 0 || strings.TrimSpace(argv[0]) == "" {
		return nil, fmt.Errorf("no command to run")
	}
	return commandContext(ctx, argv)
}
