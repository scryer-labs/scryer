package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

var ErrNotInstalled = errors.New("engine is not installed")

const installRecord = ".scryer-install.json"

type Installation struct {
	Engine      Identity `json:"engine"`
	Executable  string   `json:"executable"`
	PackageRoot string   `json:"packageRoot"`
	SourcePath  string   `json:"sourcePath"`
	InstalledAt string   `json:"installedAt"`
}

type record struct {
	SchemaVersion int      `json:"schemaVersion"`
	Manifest      Manifest `json:"manifest"`
	SourcePath    string   `json:"sourcePath"`
	InstalledAt   string   `json:"installedAt"`
}

type Store struct{ Home string }

func DefaultStore() (Store, error) {
	home := os.Getenv("SCRYER_HOME")
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return Store{}, err
		}
		home = filepath.Join(userHome, ".config", "scryer")
	}
	home, err := filepath.Abs(home)
	return Store{Home: home}, err
}

func (s Store) directory() string { return filepath.Join(s.Home, "engines") }

func (s Store) List(ctx context.Context) ([]Installation, error) {
	installed := []Installation{}
	if _, err := os.Stat(s.Home); errors.Is(err, os.ErrNotExist) {
		return installed, nil
	} else if err != nil {
		return nil, err
	}
	unlock, err := s.lock(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()
	entries, err := os.ReadDir(s.directory())
	if errors.Is(err, os.ErrNotExist) {
		return installed, nil
	}
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !engineID.MatchString(entry.Name()) {
			continue
		}
		installation, err := s.load(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("installed engine %s: %w", entry.Name(), err)
		}
		installed = append(installed, installation)
	}
	sort.Slice(installed, func(i, j int) bool { return installed[i].Engine.ID < installed[j].Engine.ID })
	return installed, nil
}

// Find returns only owned installed packages. Callers handle explicit development overrides separately.
func (s Store) Find(ctx context.Context, id string) (Installation, error) {
	if err := validateID(id); err != nil {
		return Installation{}, err
	}
	if _, err := os.Stat(s.Home); errors.Is(err, os.ErrNotExist) {
		return Installation{}, ErrNotInstalled
	} else if err != nil {
		return Installation{}, err
	}
	unlock, err := s.lock(ctx)
	if err != nil {
		return Installation{}, err
	}
	defer unlock()
	if _, err := os.Lstat(filepath.Join(s.directory(), id)); errors.Is(err, os.ErrNotExist) {
		return Installation{}, ErrNotInstalled
	} else if err != nil {
		return Installation{}, err
	}
	return s.load(id)
}

func (s Store) load(id string) (Installation, error) {
	directory := filepath.Join(s.directory(), id)
	if err := realDirectory(directory); err != nil {
		return Installation{}, err
	}
	var metadata record
	if err := readJSON(filepath.Join(directory, installRecord), &metadata); err != nil {
		return Installation{}, fmt.Errorf("not an owned installation: %w", err)
	}
	manifest, root, err := ReadPackage(directory)
	if err != nil {
		return Installation{}, err
	}
	if metadata.SchemaVersion != 1 || metadata.Manifest != manifest || manifest.Engine.ID != id {
		return Installation{}, fmt.Errorf("installation metadata does not match package")
	}
	return Installation{Engine: manifest.Engine, Executable: filepath.Join(root, filepath.FromSlash(manifest.Executable)), PackageRoot: root, SourcePath: metadata.SourcePath, InstalledAt: metadata.InstalledAt}, nil
}

func (s Store) Install(ctx context.Context, id, source string, diagnostics io.Writer) (Installation, error) {
	if err := validateID(id); err != nil {
		return Installation{}, err
	}
	manifest, source, err := ReadPackage(source)
	if err != nil {
		return Installation{}, err
	}
	if manifest.Engine.ID != id {
		return Installation{}, fmt.Errorf("package identifies as %q, not requested %q", manifest.Engine.ID, id)
	}
	unlock, err := s.lock(ctx)
	if err != nil {
		return Installation{}, err
	}
	defer unlock()
	// Compare canonical paths so aliases such as /tmp and /private/tmp cannot hide overlap.
	home, err := filepath.EvalSymlinks(s.Home)
	if err != nil {
		return Installation{}, err
	}
	if within(source, home) || within(home, source) {
		return Installation{}, fmt.Errorf("source package and Scryer home must be separate directories")
	}
	target := filepath.Join(s.directory(), id)
	if _, err := os.Lstat(target); err == nil {
		if _, err := s.load(id); err != nil {
			return Installation{}, err
		}
		return Installation{}, fmt.Errorf("engine %q is already installed; uninstall it before installing another package", id)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Installation{}, err
	}
	staging, err := os.MkdirTemp(s.directory(), ".install-")
	if err != nil {
		return Installation{}, err
	}
	defer os.RemoveAll(staging)
	if err := copyPackage(ctx, source, staging); err != nil {
		return Installation{}, err
	}
	if _, err := os.Lstat(filepath.Join(staging, installRecord)); !errors.Is(err, os.ErrNotExist) {
		return Installation{}, fmt.Errorf("source package must not contain installation metadata")
	}
	if err := verifyPackage(ctx, staging, manifest, diagnostics); err != nil {
		return Installation{}, err
	}
	metadata := record{SchemaVersion: 1, Manifest: manifest, SourcePath: source, InstalledAt: time.Now().UTC().Format(time.RFC3339Nano)}
	if err := writeJSON(filepath.Join(staging, installRecord), metadata); err != nil {
		return Installation{}, err
	}
	// Publish only the fully copied and verified package. Existing installations are never overwritten.
	if err := os.Rename(staging, target); err != nil {
		return Installation{}, err
	}

	return s.load(id)
}

func (s Store) Uninstall(ctx context.Context, id string) (Installation, error) {
	if err := validateID(id); err != nil {
		return Installation{}, err
	}
	if _, err := os.Stat(s.Home); err != nil {
		return Installation{}, err
	}
	unlock, err := s.lock(ctx)
	if err != nil {
		return Installation{}, err
	}
	defer unlock()
	installation, err := s.load(id)
	if err != nil {
		return Installation{}, err
	}
	// Only a validated, owned installation is removable; source and project config are untouched.
	if err := os.RemoveAll(installation.PackageRoot); err != nil {
		return Installation{}, err
	}
	return installation, nil
}

func realDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("expected a real directory: %s", path)
	}
	return nil
}
