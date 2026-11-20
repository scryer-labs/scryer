// Package projectscope owns project configuration and the shared file inventory.
package projectscope

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/edwardnoaland/scryer/internal/contract"
	"gopkg.in/yaml.v2"
)

const ConfigPath = ".scryer/project.yaml"

type Config struct {
	SchemaVersion int `yaml:"schemaVersion"`
	Scan          struct {
		Include []string `yaml:"include"`
		Exclude []string `yaml:"exclude"`
	} `yaml:"scan"`
	Digest string `yaml:"-"`
}
type Inventory struct {
	Files []string
	Scope *contract.ScanScope
}

func ReadConfig(root string) (*Config, error) {
	directory, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	file, err := directory.Open(filepath.FromSlash(ConfigPath))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", ConfigPath, err)
	}
	data, err := io.ReadAll(io.LimitReader(file, 1<<20+1))
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if len(data) > 1<<20 {
		return nil, fmt.Errorf("%s exceeds 1 MiB", ConfigPath)
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", ConfigPath, err)
	}
	var config Config
	if err = yaml.UnmarshalStrict(data, &config); err != nil {
		return nil, fmt.Errorf("parse %s: %w", ConfigPath, err)
	}
	if config.SchemaVersion != 1 {
		return nil, fmt.Errorf("%s requires schemaVersion: 1", ConfigPath)
	}
	if config.Scan.Include == nil {
		config.Scan.Include = []string{"**/*"}
	}
	if config.Scan.Exclude == nil {
		config.Scan.Exclude = []string{}
	}
	for _, pattern := range append(append([]string{}, config.Scan.Include...), config.Scan.Exclude...) {
		if err = validatePattern(pattern); err != nil {
			return nil, fmt.Errorf("%s: %w", ConfigPath, err)
		}
	}
	config.Digest = fmt.Sprintf("%x", sha256.Sum256(data))
	return &config, nil
}

func validatePattern(pattern string) error {
	if pattern == "" || path.IsAbs(pattern) || strings.Contains(pattern, "\\") {
		return fmt.Errorf("invalid project-relative glob %q", pattern)
	}
	for _, segment := range strings.Split(pattern, "/") {
		if segment == "" || segment == "." || segment == ".." || (strings.Contains(segment, "**") && segment != "**") {
			return fmt.Errorf("invalid glob segment in %q", pattern)
		}
		if segment != "**" {
			if _, err := path.Match(segment, ""); err != nil {
				return fmt.Errorf("invalid glob %q: %w", pattern, err)
			}
		}
	}
	return nil
}

// ** matches complete path segments; other segments use Go path.Match syntax.
func match(pattern, file string) bool {
	p, f := strings.Split(pattern, "/"), strings.Split(file, "/")
	type state struct{ i, j int }
	seen, values := map[state]bool{}, map[state]bool{}
	var visit func(int, int) bool
	visit = func(i, j int) bool {
		key := state{i, j}
		if seen[key] {
			return values[key]
		}
		seen[key] = true
		result := false
		if i == len(p) {
			result = j == len(f)
		} else if p[i] == "**" {
			result = visit(i+1, j) || (j < len(f) && visit(i, j+1))
		} else if j < len(f) {
			ok, _ := path.Match(p[i], f[j])
			result = ok && visit(i+1, j+1)
		}
		values[key] = result
		return result
	}
	return visit(0, 0)
}
func matches(patterns []string, file string) bool {
	for _, pattern := range patterns {
		if match(pattern, file) {
			return true
		}
	}
	return false
}

// A nil config explicitly selects the existing default inventory. Callers decide
// which configuration to use, including pinning one policy across impact snapshots.
func Build(ctx context.Context, root string, config *Config) (Inventory, error) {
	directory, err := os.OpenRoot(root)
	if err != nil {
		return Inventory{}, err
	}
	defer directory.Close()
	result := Inventory{Files: []string{}}
	if config != nil {
		base := "filesystem"
		git, err := hasGitAncestor(root)
		if err != nil {
			return result, err
		}
		if git {
			base = "git-tracked-and-nonignored"
		}
		result.Scope = &contract.ScanScope{SchemaVersion: 1, Config: ConfigPath, ConfigSHA256: config.Digest, Base: base, Include: config.Scan.Include, Exclude: config.Scan.Exclude}
	}
	err = VisitFiles(ctx, directory, func(file string) error {
		info, err := directory.Lstat(file)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		file = filepath.ToSlash(file)
		if result.Scope != nil {
			result.Scope.Considered++
			if !matches(config.Scan.Include, file) || matches(config.Scan.Exclude, file) {
				result.Scope.Excluded++
				return nil
			}
		}
		result.Files = append(result.Files, file)
		return nil
	})
	if err != nil {
		return result, err
	}
	sort.Strings(result.Files)
	if result.Scope != nil {
		result.Scope.Included = len(result.Files)
		_, digest, err := manifest(root, result.Files)
		if err != nil {
			return result, err
		}
		result.Scope.SelectionSHA256 = digest
	}
	return result, nil
}
func Load(ctx context.Context, root string) (Inventory, error) {
	config, err := ReadConfig(root)
	if err != nil {
		return Inventory{}, err
	}
	return Build(ctx, root, config)
}
func manifest(root string, files []string) ([]byte, string, error) {
	data, err := json.Marshal(struct {
		SchemaVersion int      `json:"schemaVersion"`
		Root          string   `json:"root"`
		Files         []string `json:"files"`
	}{1, root, files})
	if err != nil {
		return nil, "", err
	}
	return data, fmt.Sprintf("%x", sha256.Sum256(data)), nil
}
func CreateManifest(root string, files []string) (*contract.FileSelectionRef, func(), error) {
	data, digest, err := manifest(root, files)
	if err != nil {
		return nil, nil, err
	}
	if len(data) > 64<<20 {
		return nil, nil, fmt.Errorf("file selection exceeds 64 MiB")
	}
	file, err := os.CreateTemp("", "scryer-file-selection-*.json")
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = os.Remove(file.Name()) }
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if err = errors.Join(writeErr, closeErr); err != nil {
		cleanup()
		return nil, nil, err
	}
	return &contract.FileSelectionRef{Path: file.Name(), SHA256: digest, FileCount: len(files)}, cleanup, nil
}
