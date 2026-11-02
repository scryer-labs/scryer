// Package engine owns local engine packages and their installed inventory.
package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

var engineID = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// Manifest describes a relocatable local package, independently of scan report data.
type Manifest struct {
	SchemaVersion int      `json:"schemaVersion"`
	Engine        Identity `json:"engine"`
	Executable    string   `json:"executable"`
	OS            string   `json:"os"`
	Arch          string   `json:"arch"`
}

func validateID(id string) error {
	if !engineID.MatchString(id) {
		return fmt.Errorf("invalid engine id %q", id)
	}
	return nil
}

func ReadPackage(directory string) (Manifest, string, error) {
	root, err := filepath.Abs(directory)
	if err != nil {
		return Manifest{}, "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return Manifest{}, "", err
	}
	var manifest Manifest
	if err := readJSON(filepath.Join(root, "engine.json"), &manifest); err != nil {
		return manifest, "", fmt.Errorf("engine package manifest: %w", err)
	}
	if err := validateID(manifest.Engine.ID); err != nil {
		return manifest, "", err
	}
	if manifest.SchemaVersion != 1 || manifest.Engine.ProtocolVersion != 1 || manifest.Engine.Mock {
		return manifest, "", fmt.Errorf("expected a real engine package with schemaVersion/protocolVersion 1")
	}
	if manifest.OS != runtime.GOOS || manifest.Arch != runtime.GOARCH {
		return manifest, "", fmt.Errorf("engine package targets %s/%s; host is %s/%s", manifest.OS, manifest.Arch, runtime.GOOS, runtime.GOARCH)
	}
	if manifest.Executable == "" || filepath.IsAbs(manifest.Executable) || !filepath.IsLocal(filepath.FromSlash(manifest.Executable)) {
		return manifest, "", fmt.Errorf("engine executable must be relative to its package")
	}
	executable := filepath.Join(root, filepath.FromSlash(manifest.Executable))
	resolved, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return manifest, "", err
	}
	if !within(root, resolved) {
		return manifest, "", fmt.Errorf("engine executable escapes its package")
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return manifest, "", err
	}
	if !info.Mode().IsRegular() {
		return manifest, "", fmt.Errorf("engine executable is not a regular file")
	}
	return manifest, root, nil
}

func verifyPackage(ctx context.Context, root string, manifest Manifest, diagnostics io.Writer) error {
	identity, _, err := Describe(ctx, filepath.Join(root, filepath.FromSlash(manifest.Executable)), diagnostics)
	if err != nil {
		return err
	}
	if identity != manifest.Engine {
		return fmt.Errorf("engine describe identity does not match package manifest")
	}
	return nil
}

func within(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && (relative == "." || filepath.IsLocal(relative))
}

func readJSON(path string, value any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 1024*1024+1))
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("expected exactly one JSON document")
	}
	return nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0600)
}

// PackageForExecutable preserves the development launcher flag without treating an arbitrary bin directory as a package.
func PackageForExecutable(executable string) (string, error) {
	absolute, err := filepath.Abs(executable)
	if err != nil {
		return "", err
	}
	absolute, err = filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	manifest, root, err := ReadPackage(filepath.Dir(filepath.Dir(absolute)))
	if err != nil {
		return "", err
	}
	expected, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(manifest.Executable)))
	if err != nil {
		return "", err
	}
	if expected != absolute {
		return "", fmt.Errorf("executable does not match the package launcher")
	}
	return root, nil
}
