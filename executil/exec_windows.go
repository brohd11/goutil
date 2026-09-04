//go:build windows

package executil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func commandContext(ctx context.Context, argv []string) (*exec.Cmd, error) {
	path, err := exec.LookPath(argv[0])
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".cmd" && ext != ".bat" {
		return exec.CommandContext(ctx, path, argv[1:]...), nil
	}

	comspec := strings.TrimSpace(os.Getenv("COMSPEC"))
	if comspec == "" {
		comspec = "cmd.exe"
	}
	cmd := exec.CommandContext(ctx, comspec)
	// Supplying CmdLine bypasses os/exec's CommandLineToArgvW quoting. cmd.exe has
	// its own grammar; node_modules shims need a second metacharacter escape because
	// their forwarding layer parses the arguments again.
	doubleMeta := isNodeCmdShim(path)
	line, err := cmdJoin(append([]string{path}, argv[1:]...), doubleMeta)
	if err != nil {
		return nil, err
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: fmt.Sprintf(`%s /d /s /v:off /c "%s"`, syscall.EscapeArg(comspec), line)}
	return cmd, nil
}

func isNodeCmdShim(path string) bool {
	normalized := strings.ReplaceAll(strings.ToLower(filepath.Clean(path)), `\`, "/")
	return strings.Contains(normalized, "/node_modules/.bin/") && strings.HasSuffix(normalized, ".cmd")
}
