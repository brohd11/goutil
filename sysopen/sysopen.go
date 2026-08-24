// Package sysopen hands filesystem paths to the operating system's file manager.
// It contains no TUI or CLI policy, so both interactive apps and ordinary commands
// can share the same cross-platform launcher.
package sysopen

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

var buildPathCommand = func(path string, reveal bool) *exec.Cmd {
	return pathCommand(runtime.GOOS, path, reveal)
}

// OpenPath opens path in the OS file manager. When reveal is true, a file is
// highlighted in its containing directory where the platform supports that gesture.
// The launcher is detached: success means the OS command started successfully, not
// that the file manager remained open or displayed the path.
func OpenPath(path string, reveal bool) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	cmd := buildPathCommand(path, reveal)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	go cmd.Wait() //nolint:errcheck // reap while the detached file manager owns its lifetime
	return nil
}

// pathCommand is split from OpenPath so every platform's argv can be tested on one
// host. Linux has no portable "reveal" verb, so it opens the containing directory.
func pathCommand(goos, path string, reveal bool) *exec.Cmd {
	switch goos {
	case "darwin":
		if reveal {
			return exec.Command("open", "-R", path)
		}
		return exec.Command("open", path)
	case "windows":
		if reveal {
			return exec.Command("explorer", "/select,"+path)
		}
		return exec.Command("explorer", path)
	default:
		if reveal {
			path = filepath.Dir(path)
		}
		return exec.Command("xdg-open", path)
	}
}
