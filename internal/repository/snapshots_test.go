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
