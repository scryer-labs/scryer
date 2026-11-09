package projectfacts

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/edwardnoaland/scryer/internal/process"
)

func visitFiles(ctx context.Context, root *os.Root, visit func(string) error) error {
	inGit, err := hasGitAncestor(root.Name())
	if err != nil {
		return err
	}
	if inGit {
		command := process.Command(ctx, root.Name(), "git", "ls-files", "--cached", "--others", "--exclude-standard", "-z", "--", ".")
		output, err := command.Output()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			return fmt.Errorf("list project files: %w", err)
		}
		seen := make(map[string]bool)
		for _, entry := range bytes.Split(output, []byte{0}) {
			path := string(entry)
			if path == "" || seen[path] {
				continue
			}
			seen[path] = true
			if err := visit(path); err != nil {
				return err
			}
		}
		return nil
	}
	return fs.WalkDir(root.FS(), ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if entry.Name() == ".git" {
			return nil
		}
		return visit(path)
	})
}

func hasGitAncestor(path string) (bool, error) {
	for {
		_, err := os.Lstat(filepath.Join(path, ".git"))
		if err == nil {
			return true, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		parent := filepath.Dir(path)
		if parent == path {
			return false, nil
		}
		path = parent
	}
}
