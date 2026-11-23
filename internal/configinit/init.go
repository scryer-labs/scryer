// Package configinit creates Core-owned scope configuration from shared templates.
// It does not inspect languages, engine packages or existing configuration contents.
package configinit

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Scope string

const (
	Project   Scope = "project"
	Workspace Scope = "workspace"
)

//go:embed templates/project.yaml
var projectTemplate string

//go:embed templates/workspace.yaml
var workspaceTemplate string

type Result struct {
	Operation string `json:"operation"`
	Scope     Scope  `json:"scope"`
	Status    string `json:"status"`
	Root      string `json:"root"`
	Path      string `json:"path"`
}

func template(scope Scope) (string, error) {
	switch scope {
	case Project:
		return projectTemplate, nil
	case Workspace:
		return workspaceTemplate, nil
	default:
		return "", fmt.Errorf("unknown configuration scope %q", scope)
	}
}

// Initialize creates only the requested configuration. Exclusive creation keeps
// an existing file intact, including concurrent initialization by another caller.
func Initialize(ctx context.Context, directory string, scope Scope) (Result, error) {
	result := Result{Operation: "init", Scope: scope}
	content, err := template(scope)
	if err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return result, err
	}
	absolute, err = filepath.EvalSymlinks(absolute)
	if err != nil {
		return result, err
	}
	root, err := os.OpenRoot(absolute)
	if err != nil {
		return result, err
	}
	defer root.Close()
	result.Root = absolute
	relative := filepath.Join(".scryer", string(scope)+".yaml")
	result.Path = filepath.Join(absolute, relative)
	if err = root.Mkdir(".scryer", 0755); err != nil && !errors.Is(err, os.ErrExist) {
		return result, err
	}
	file, err := root.OpenFile(relative, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if errors.Is(err, os.ErrExist) {
		result.Status = "exists"
		return result, nil
	}
	if err != nil {
		return result, fmt.Errorf("create %s: %w", result.Path, err)
	}
	_, writeErr := file.WriteString(content)
	closeErr := file.Close()
	if err = errors.Join(writeErr, closeErr); err != nil {
		_ = root.Remove(relative)
		return result, fmt.Errorf("write %s: %w", result.Path, err)
	}
	result.Status = "created"
	return result, nil
}
