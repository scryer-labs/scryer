//go:build !cgo || (!darwin && !linux)

package engineabi

import "fmt"

type library struct{}

func open(string) (library, error) {
	return library{}, fmt.Errorf("shared-library engines require cgo on macOS or Linux")
}

func (library) execute([]byte) ([]byte, int32, error) {
	return nil, 0, fmt.Errorf("shared-library engine unavailable")
}

func (library) cancel(string) bool { return false }
