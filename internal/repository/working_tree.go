package repository

import (
	"bytes"
	"context"
	"fmt"
	"github.com/edwardnoaland/scryer/internal/process"
	"os"
	"path/filepath"
	"strings"
)

// Capture through the clone's private index: never stage or commit in the user's repository.
// Reading twice detects concurrent edits; the immutable private commit is the evidence identity.
func captureWorkingTree(ctx context.Context, root, clone, base string) (string, error) {
	entries, err := git(ctx, root, "ls-files", "--stage")
	if err != nil {
		return "", err
	}
	for _, entry := range strings.Split(entries, "\n") {
		if strings.HasPrefix(entry, "160000 ") {
			return "", fmt.Errorf("working-tree snapshots do not yet support submodules; use committed refs")
		}
	}
	if _, err = git(ctx, clone, "read-tree", base); err != nil {
		return "", err
	}
	capture := func() (string, error) {
		// Preserve tracked/staged files even when an ignore rule now matches them.
		listing, err := process.Command(ctx, root, "git", "ls-files", "--cached", "--others", "--exclude-standard", "-z").Output()
		if err != nil {
			return "", err
		}
		var tracked bytes.Buffer
		for _, path := range bytes.Split(listing, []byte{0}) {
			if len(path) == 0 {
				continue
			}
			if _, err := os.Lstat(filepath.Join(root, string(path))); err == nil {
				tracked.Write(path)
				tracked.WriteByte(0)
			} else if !os.IsNotExist(err) {
				return "", err
			}
		}

		if _, err := git(ctx, clone, "--work-tree="+root, "add", "--update", "--", "."); err != nil {
			return "", err
		}
		if tracked.Len() > 0 {
			command := process.Command(ctx, clone, "git", "--literal-pathspecs", "--work-tree="+root, "add", "--force", "--pathspec-from-file=-", "--pathspec-file-nul")
			command.Stdin = bytes.NewReader(tracked.Bytes())
			if output, err := command.CombinedOutput(); err != nil {
				return "", fmt.Errorf("capture tracked files: %w: %s", err, output)
			}
		}
		return git(ctx, clone, "write-tree")
	}
	tree, err := capture()
	if err != nil {
		return "", err
	}
	second, err := capture()
	if err != nil {
		return "", err
	}
	head, err := git(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	if tree != second || head != base {
		return "", fmt.Errorf("working tree changed during snapshot capture; retry analysis")
	}
	return git(ctx, clone, "-c", "user.name=Scryer", "-c", "user.email=snapshot@scryer.invalid", "commit-tree", tree, "-p", base, "-m", "Scryer isolated working-tree snapshot")
}
