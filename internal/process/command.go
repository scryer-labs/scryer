// Package process owns bounded subprocess execution and cancellation.
package process

import (
	"context"
	"os/exec"
	"time"
)

func Command(ctx context.Context, directory, executable string, arguments ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, executable, arguments...)
	cmd.Dir = directory
	cmd.WaitDelay = 3 * time.Second
	configure(cmd)
	return cmd
}
