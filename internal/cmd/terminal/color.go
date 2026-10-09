// Package terminal owns shared CLI color preferences, independent of report formats.
package terminal

import (
	"io"
	"os"
)

func UseColor(mode string, output io.Writer) bool {
	if mode == "always" {
		return true
	}
	if mode == "never" {
		return false
	}
	if _, disabled := os.LookupEnv("NO_COLOR"); disabled {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	file, ok := output.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
