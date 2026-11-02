package engine

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

var testIdentity = Identity{ProtocolVersion: 1, ID: "java", Name: "Test engine", Version: "test", Source: "example/java"}

func TestMain(m *testing.M) {
	if len(os.Args) == 2 && os.Args[1] == "describe" {
		identity := testIdentity
		if os.Getenv("SCRYER_TEST_WRONG_IDENTITY") == "true" {
			identity.Version = "different"
		}
		_ = json.NewEncoder(os.Stdout).Encode(identity)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func packageFixture(t *testing.T) string {
	t.Helper()
	source := t.TempDir()
	if err := os.MkdirAll(filepath.Join(source, "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if err := copyFile(context.Background(), executable, filepath.Join(source, "bin", "engine"), 0755); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{SchemaVersion: 1, Engine: testIdentity, Executable: "bin/engine", OS: runtime.GOOS, Arch: runtime.GOARCH}
	if err := writeJSON(filepath.Join(source, "engine.json"), manifest); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(source, "lib"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "lib", "facts.jar"), []byte("worker payload"), 0644); err != nil {
		t.Fatal(err)
	}
	return source
}

func Test_should_persist_complete_package_when_installing_and_keep_source_when_uninstalling(t *testing.T) {
	ctx := context.Background()
	source := packageFixture(t)
	store := Store{Home: filepath.Join(t.TempDir(), "home")}
	before, err := store.List(ctx)
	if err != nil || len(before) != 0 {
		t.Fatal(before, err)
	}
	if _, err := os.Stat(store.Home); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("empty inventory created a home")
	}
	installed, err := store.Install(ctx, "java", source, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if installed.Executable == filepath.Join(source, "bin", "engine") {
		t.Fatal("installation references source executable")
	}
	if data, err := os.ReadFile(filepath.Join(installed.PackageRoot, "lib", "facts.jar")); err != nil || string(data) != "worker payload" {
		t.Fatal("worker payload missing", err)
	}
	// A second Store represents a fresh CLI invocation reading persisted state.
	fresh := Store{Home: store.Home}
	entries, err := fresh.List(ctx)
	if err != nil || len(entries) != 1 || entries[0] != installed {
		t.Fatal(entries, err)
	}
	hidden := source + "-hidden"
	if err := os.Rename(source, hidden); err != nil {
		t.Fatal(err)
	}
	defer os.Rename(hidden, source)
	identity, _, err := Describe(ctx, installed.Executable, io.Discard)
	if err != nil || identity != testIdentity {
		t.Fatal("installed package depends on source", err)
	}
	if _, err := fresh.Uninstall(ctx, "java"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(hidden); err != nil {
		t.Fatal("uninstall touched source", err)
	}
	entries, err = fresh.List(ctx)
	if err != nil || len(entries) != 0 {
		t.Fatal(entries, err)
	}
	if _, err := fresh.Find(ctx, "java"); !errors.Is(err, ErrNotInstalled) {
		t.Fatal(err)
	}
}

func Test_should_preserve_existing_install_when_reinstallation_is_requested(t *testing.T) {
	ctx := context.Background()
	source := packageFixture(t)
	store := Store{Home: t.TempDir()}
	installed, err := store.Install(ctx, "java", source, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Install(ctx, "java", source, io.Discard); err == nil {
		t.Fatal("existing engine overwritten")
	}
	current, err := store.Find(ctx, "java")
	if err != nil || current != installed {
		t.Fatal("existing installation changed", err)
	}
}

func Test_should_leave_no_installation_when_package_identity_is_inconsistent(t *testing.T) {
	source := packageFixture(t)
	store := Store{Home: t.TempDir()}
	t.Setenv("SCRYER_TEST_WRONG_IDENTITY", "true")
	if _, err := store.Install(context.Background(), "java", source, io.Discard); err == nil {
		t.Fatal("mismatched identity accepted")
	}
	entries, err := store.List(context.Background())
	if err != nil || len(entries) != 0 {
		t.Fatal("partial install published", err)
	}
}

func Test_should_reject_unsafe_package_when_links_or_entrypoints_escape_root(t *testing.T) {
	source := packageFixture(t)
	outside := t.TempDir()
	store := Store{Home: t.TempDir()}
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("outside"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret"), filepath.Join(source, "escape")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Install(context.Background(), "java", source, io.Discard); err == nil {
		t.Fatal("escaping symlink accepted")
	}
	if _, err := store.Uninstall(context.Background(), "../outside"); err == nil {
		t.Fatal("traversal accepted")
	}
	if data, err := os.ReadFile(filepath.Join(outside, "secret")); err != nil || string(data) != "outside" {
		t.Fatal("outside data changed", err)
	}
}

func Test_should_preserve_internal_links_when_package_is_relocated(t *testing.T) {
	source := packageFixture(t)
	store := Store{Home: t.TempDir()}
	if err := os.Symlink(filepath.Join(source, "lib", "facts.jar"), filepath.Join(source, "lib", "alias.jar")); err != nil {
		t.Fatal(err)
	}
	installed, err := store.Install(context.Background(), "java", source, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	link, err := os.Readlink(filepath.Join(installed.PackageRoot, "lib", "alias.jar"))
	if err != nil || filepath.IsAbs(link) {
		t.Fatal("link was not relocated", link, err)
	}
}

func Test_should_refuse_uninstall_when_directory_is_not_owned(t *testing.T) {
	store := Store{Home: t.TempDir()}
	path := filepath.Join(store.Home, "engines", "java")
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "keep"), []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Uninstall(context.Background(), "java"); err == nil {
		t.Fatal("unowned directory removed")
	}
	if _, err := os.Stat(filepath.Join(path, "keep")); err != nil {
		t.Fatal(err)
	}
}

func Test_should_stop_waiting_when_inventory_lock_context_is_cancelled(t *testing.T) {
	store := Store{Home: t.TempDir()}
	unlock, err := store.lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := store.List(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("lock wait ignored cancellation", err)
	}
}

func Test_should_use_configured_home_when_default_store_is_requested(t *testing.T) {
	home := filepath.Join(t.TempDir(), "custom")
	t.Setenv("SCRYER_HOME", home)
	store, err := DefaultStore()
	if err != nil || store.Home != home {
		t.Fatal(store, err)
	}
}

func Test_should_reject_package_when_platform_or_entrypoint_is_incompatible(t *testing.T) {
	source := packageFixture(t)
	original, _, err := ReadPackage(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Manifest){
		func(m *Manifest) { m.Arch = "unsupported" },
		func(m *Manifest) { m.Executable = "../outside" },
		func(m *Manifest) { m.SchemaVersion = 2 },
		func(m *Manifest) { m.Engine.Mock = true },
	} {
		manifest := original
		change(&manifest)
		if err := writeJSON(filepath.Join(source, "engine.json"), manifest); err != nil {
			t.Fatal(err)
		}
		if _, _, err := ReadPackage(source); err == nil {
			t.Fatal("incompatible package accepted")
		}
	}
}

func Test_should_refuse_uninstall_when_engine_directory_is_a_symlink(t *testing.T) {
	store := Store{Home: t.TempDir()}
	outside := t.TempDir()
	if err := os.MkdirAll(store.directory(), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "keep"), []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(store.directory(), "java")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Uninstall(context.Background(), "java"); err == nil {
		t.Fatal("symlinked directory removed")
	}
	if _, err := os.Stat(filepath.Join(outside, "keep")); err != nil {
		t.Fatal(err)
	}
}

func Test_should_use_dot_config_when_no_scryer_home_override_is_set(t *testing.T) {
	t.Setenv("SCRYER_HOME", "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	store, err := DefaultStore()
	expected := filepath.Join(home, ".config", "scryer")
	if err != nil || store.Home != expected {
		t.Fatalf("expected %s, got %s: %v", expected, store.Home, err)
	}
}
