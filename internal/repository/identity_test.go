package repository

import (
	"context"
	"path/filepath"
	"testing"
)

func TestParseRemoteNormalizesHostedRepositoryIdentity(t *testing.T) {
	for _, address := range []string{
		"git@GitHub.com:ScriptoriumLab/scriptorium-felt.git",
		"ssh://git@github.com:22/scriptoriumlab/scriptorium-felt.git",
		"https://github.com:443/scriptoriumlab/scriptorium-felt/",
	} {
		identity, ok := ParseRemote(address)
		if !ok || identity.ID != "github.com/scriptoriumlab/scriptorium-felt" || identity.Host != "github.com" || identity.Name != "scriptorium-felt" {
			t.Errorf("%s: %+v, %v", address, identity, ok)
		}
	}
	identity, ok := ParseRemote("https://forge.example:8443/team/group/Project.git")
	if !ok || identity.Host != "forge.example:8443" || identity.Namespace != "team/group" ||
		identity.Name != "Project" || identity.ID != "forge.example:8443/team/group/Project" {
		t.Fatalf("unexpected nested repository: %+v", identity)
	}
	for _, address := range []string{"../felt", "file:///tmp/felt", "C:/Projects/felt", "${REPO_URL}", "https://example.test/group%2Frepo", "https://example.test/group/.."} {
		if identity, ok := ParseRemote(address); ok {
			t.Errorf("accepted %q as %+v", address, identity)
		}
	}
}

func TestIdentityUsesLocalFallbackAndHostedRemote(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	assertLocal := func() {
		t.Helper()
		identity := Identity(ctx, root)
		if identity.Host != "local" || identity.Namespace != "local" || identity.Name != filepath.Base(root) || identity.ID != "" {
			t.Fatalf("unexpected local identity: %+v", identity)
		}
	}
	assertLocal()
	if _, err := git(ctx, root, "init"); err != nil {
		t.Fatal(err)
	}
	assertLocal()
	if _, err := git(ctx, root, "remote", "add", "upstream", "git@github.com:Example/Hosted.git"); err != nil {
		t.Fatal(err)
	}
	identity := Identity(ctx, root)
	if identity.Remote != "upstream" || identity.ID != "github.com/example/hosted" {
		t.Fatalf("unexpected hosted identity: %+v", identity)
	}
	if _, err := git(ctx, root, "remote", "add", "origin", "https://github.com/Fork/Hosted"); err != nil {
		t.Fatal(err)
	}
	identity = Identity(ctx, root)
	if identity.Remote != "origin" || identity.ID != "github.com/fork/hosted" {
		t.Fatalf("unexpected origin identity: %+v", identity)
	}
}
