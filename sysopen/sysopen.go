// Package sysopen opens paths in the OS file manager and URLs in the browser.
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

// OpenPath opens path in the file manager, highlighting a file in its folder when reveal
// is set and the platform supports it. Success means the launcher started.
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

// OpenURL opens target, as given, in the default browser. Windows uses ShellExecuteW so
// query strings never pass through cmd.exe.
func OpenURL(target string) error {
	if target == "" {
		return fmt.Errorf("open url: empty target")
	}
	if err := openURL(target); err != nil {
		return fmt.Errorf("open %s: %w", target, err)
	}
	return nil
}
