//go:build !windows

package executil

import (
	"context"
	"os/exec"
)

func commandContext(ctx context.Context, argv []string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	if cmd.Err != nil {
		return nil, cmd.Err
	}
	return cmd, nil
}
