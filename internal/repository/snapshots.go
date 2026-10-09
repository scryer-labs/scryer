// Package repository owns Git refs and isolated checkout lifetimes, independent of language analyzers.
package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/edwardnoaland/scryer/internal/process"
)

type Snapshots struct{ Repository, Before, After, BeforeRoot, AfterRoot, ComparisonRepository, WorkingBase, temporary string }

func (s *Snapshots) Close() error { return os.RemoveAll(s.temporary) }
func git(ctx context.Context, directory string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	output, err := process.Command(ctx, directory, "git", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}
func Open(ctx context.Context, directory, before, after string) (*Snapshots, error) {
	root, err := git(ctx, directory, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	resolve := func(ref string) (string, error) {
		return git(ctx, root, "rev-parse", "--verify", "--end-of-options", ref+"^{commit}")
	}
	beforeSHA, err := resolve(before)
	if err != nil {
		return nil, err
	}
	afterRef := after
	if after == "." {
		afterRef = "HEAD"
	}
	afterSHA, err := resolve(afterRef)
	if err != nil {
		return nil, err
	}
	temporary, err := os.MkdirTemp("", "scryer-snapshots-")
	if err != nil {
		return nil, err
	}
	snapshot := &Snapshots{Repository: root, Before: beforeSHA, After: afterSHA, BeforeRoot: filepath.Join(temporary, "before"), AfterRoot: filepath.Join(temporary, "after"), temporary: temporary}
	complete := false
	defer func() {
		if !complete {
			_ = snapshot.Close()
		}
	}()
	clone := filepath.Join(temporary, "repository")
	if _, err = git(ctx, temporary, "clone", "--shared", "--no-checkout", "--", root, clone); err != nil {
		return nil, err
	}
	snapshot.ComparisonRepository = clone
	if after == "." {
		snapshot.WorkingBase = afterSHA
		snapshot.After, err = captureWorkingTree(ctx, root, clone, afterSHA)
		if err != nil {
			return nil, err
		}
		afterSHA = snapshot.After
	}
	// Each snapshot has a normal .git directory. Older embedded Git clients (for
	// example JGit in legacy Maven plugins) cannot resolve linked-worktree metadata.
	// Objects are shared only inside this temporary workspace.
	for _, checkout := range []struct{ path, sha string }{{snapshot.BeforeRoot, beforeSHA}, {snapshot.AfterRoot, afterSHA}} {
		if _, err = git(ctx, temporary, "clone", "--shared", "--no-checkout", "--", clone, checkout.path); err != nil {
			return nil, err
		}
		if _, err = git(ctx, checkout.path, "checkout", "--detach", checkout.sha); err != nil {
			return nil, err
		}
	}
	complete = true
	return snapshot, nil
}
