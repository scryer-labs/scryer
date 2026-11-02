package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
)

func (s Store) lock(ctx context.Context) (func(), error) {
	for _, directory := range []string{s.Home, s.directory()} {
		if _, err := os.Lstat(directory); errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(directory, 0700); err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
		if err := realDirectory(directory); err != nil {
			return nil, err
		}
	}
	file, err := os.OpenFile(filepath.Join(s.Home, "engines.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := lockFile(ctx, file); err != nil {
		file.Close()
		return nil, err
	}
	return func() { file.Close() }, nil
}
