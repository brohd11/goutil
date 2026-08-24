// Package sysopen hands filesystem paths to the operating system's file manager, and
// URLs to its browser. It contains no TUI or CLI policy, so both interactive apps and
// ordinary commands can share the same cross-platform launchers.
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

// URLCommand builds the command that opens target in the default browser. goos is a
// parameter so every platform's argv can be tested on one host, the same reason
// pathCommand takes one.
//
// Note this is NOT pathCommand with a different argument, though the darwin and linux
// arms match: Windows opens a path with explorer and a URL through the shell's start
// verb, so the two switches deliberately part company there. bubblestack had its own
// copy of the darwin/linux arms before this existed.
func URLCommand(goos, target string) *exec.Cmd {
	switch goos {
	case "darwin":
		return exec.Command("open", target)
	case "windows":
		// The empty first argument is start's title parameter. Without it a quoted
		// target would be consumed as the window title and never opened.
		return exec.Command("cmd", "/c", "start", "", target)
	default:
		return exec.Command("xdg-open", target)
	}
}

// OpenURL opens target in the default browser. Like OpenPath the launcher is detached:
// success means the OS command started, not that a browser displayed the page.
//
// target is used as-is — any scheme or host normalization is the caller's job, since
// this package names no domain type.
func OpenURL(target string) error {
	if target == "" {
		return fmt.Errorf("open url: empty target")
	}
	cmd := URLCommand(runtime.GOOS, target)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open %s: %w", target, err)
	}
	go cmd.Wait() //nolint:errcheck // reap while the detached browser owns its lifetime
	return nil
}
