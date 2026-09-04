//go:build !windows

package sysopen

import (
	"os/exec"
	"runtime"
)

func urlCommand(goos, target string) *exec.Cmd {
	if goos == "darwin" {
		return exec.Command("open", target)
	}
	return exec.Command("xdg-open", target)
}

func openURL(target string) error {
	cmd := urlCommand(runtime.GOOS, target)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait() //nolint:errcheck // reap while the detached browser owns its lifetime
	return nil
}
