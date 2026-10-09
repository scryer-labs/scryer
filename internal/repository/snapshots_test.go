package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotsResolveExpressionsPreserveDirtySourceAndCleanUp(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		out, err := git(ctx, root, args...)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	run("init")
	run("config", "user.name", "Fixture")
	run("config", "user.email", "fixture@example.test")
	file := filepath.Join(root, "source.java")
	if err := os.WriteFile(file, []byte("before"), 0600); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "before")
	if err := os.WriteFile(file, []byte("after"), 0600); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "after")
	if err := os.WriteFile(file, []byte("dirty"), 0600); err != nil {
		t.Fatal(err)
	}
	original := run("status", "--porcelain")
	trees := run("worktree", "list", "--porcelain")
	snapshots, err := Open(ctx, root, "HEAD~1", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	for path, expected := range map[string]string{snapshots.BeforeRoot: "before", snapshots.AfterRoot: "after"} {
		// Older embedded Git implementations require a normal .git directory.
		info, err := os.Stat(filepath.Join(path, ".git"))
		if err != nil || !info.IsDir() {
			t.Fatalf("snapshot is not a standalone checkout: %v", err)
		}
		if common, err := git(ctx, path, "rev-parse", "--git-common-dir"); err != nil || common != ".git" {
			t.Fatalf("shared worktree metadata: %s %v", common, err)
		}
		actual, err := os.ReadFile(filepath.Join(path, "source.java"))
		if err != nil || string(actual) != expected {
			t.Fatalf("checkout %s: %s %v", path, actual, err)
		}
	}
	if run("status", "--porcelain") != original || run("worktree", "list", "--porcelain") != trees {
		t.Fatal("modified user's checkout/worktree metadata")
	}
	temporary := snapshots.temporary
	if err := snapshots.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(temporary); !os.IsNotExist(err) {
		t.Fatal("snapshots remain")
	}
	if _, err := Open(ctx, root, "--bad", "HEAD"); err == nil {
		t.Fatal("invalid reference accepted")
	}
}

func TestWorkingTreeCapturesCurrentContentWithoutChangingOriginal(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		out, err := git(ctx, root, args...)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	write := func(name, value string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	run("init")
	run("config", "user.name", "Fixture")
	run("config", "user.email", "fixture@example.test")
	write(".gitignore", "ignored/\n")
	write("source.java", "baseline")
	write("deleted.java", "deleted")
	run("add", ".")
	run("commit", "-m", "baseline")
	base := run("rev-parse", "HEAD")
	write("source.java", "staged")
	run("add", "source.java")
	write("source.java", "unstaged")
	write("staged-ignored.java", "staged ignored")
	run("add", "staged-ignored.java")
	write(".gitignore", "ignored/\nstaged-ignored.java\n")
	write("new file.java", "untracked")
	if err := os.Remove(filepath.Join(root, "deleted.java")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "ignored"), 0700); err != nil {
		t.Fatal(err)
	}
	write("ignored/cache", "excluded")
	write("local-cache", "excluded by repository-local rule")
	if err := os.WriteFile(filepath.Join(root, ".git/info/exclude"), []byte("local-cache\n"), 0600); err != nil {
		t.Fatal(err)
	}
	status, index, trees := run("status", "--porcelain"), run("ls-files", "--stage"), run("worktree", "list", "--porcelain")
	snapshots, err := Open(ctx, root, "HEAD", ".")
	if err != nil {
		t.Fatal(err)
	}
	defer snapshots.Close()
	if snapshots.WorkingBase != base || snapshots.After == base {
		t.Fatal("missing snapshot identity")
	}
	for name, expected := range map[string]string{"source.java": "unstaged", "new file.java": "untracked", "staged-ignored.java": "staged ignored"} {
		content, err := os.ReadFile(filepath.Join(snapshots.AfterRoot, name))
		if err != nil || string(content) != expected {
			t.Fatalf("%s: %s %v", name, content, err)
		}
	}
	for _, name := range []string{"deleted.java", "ignored/cache", "local-cache"} {
		if _, err := os.Stat(filepath.Join(snapshots.AfterRoot, name)); !os.IsNotExist(err) {
			t.Fatalf("unexpected snapshot file %s: %v", name, err)
		}
	}
	if run("status", "--porcelain") != status || run("ls-files", "--stage") != index || run("rev-parse", "HEAD") != base || run("worktree", "list", "--porcelain") != trees {
		t.Fatal("original repository changed")
	}
	if _, err := git(ctx, root, "cat-file", "-e", snapshots.After); err == nil {
		t.Fatal("snapshot commit leaked into original repository")
	}
}
