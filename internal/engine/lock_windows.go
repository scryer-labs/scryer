//go:build windows

package engine

import (
	"context"
	"fmt"
	"os"
)

func lockFile(ctx context.Context, file *os.File) error {
	return fmt.Errorf("persistent engine management on Windows is to be supported")
}
