package engine

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func copyPackage(ctx context.Context, source, target string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		destination := filepath.Join(target, relative)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.Mkdir(destination, 0755)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}
			if !within(source, resolved) {
				return fmt.Errorf("package symlink escapes its root: %s", relative)
			}
			inside, err := filepath.Rel(source, resolved)
			if err != nil {
				return err
			}
			link, err := filepath.Rel(filepath.Dir(destination), filepath.Join(target, inside))
			if err != nil {
				return err
			}
			return os.Symlink(link, destination)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported package entry: %s", relative)
		}
		return copyFile(ctx, path, destination, info.Mode().Perm())
	})
}

func copyFile(ctx context.Context, source, target string, mode fs.FileMode) (err error) {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := out.Close(); err == nil {
			err = closeErr
		}
	}()
	buffer := make([]byte, 256*1024)
	for {
		if err = ctx.Err(); err != nil {
			return err
		}
		count, readErr := in.Read(buffer)
		if count > 0 {
			if _, err = out.Write(buffer[:count]); err != nil {
				return err
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}
