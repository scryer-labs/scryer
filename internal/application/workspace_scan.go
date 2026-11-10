package application

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/edwardnoaland/scryer/internal/contract"
	"gopkg.in/yaml.v2"
)

type WorkspaceScan struct {
	SchemaVersion int                    `json:"schemaVersion"`
	Root          string                 `json:"root"`
	Projects      []WorkspaceProject     `json:"projects"`
	Dependencies  *WorkspaceDependencies `json:"dependencies,omitempty"`
}

type WorkspaceProject struct {
	Path  string                 `json:"path"`
	Facts *contract.ProjectFacts `json:"facts,omitempty"`
	Error string                 `json:"error,omitempty"`
}

type WorkspaceDependencies struct {
	Scope       string               `json:"scope"`
	Unsupported []string             `json:"unsupported"`
	References  []WorkspaceReference `json:"references"`
}

type WorkspaceReference struct {
	From        string                      `json:"from"`
	To          string                      `json:"to,omitempty"`
	Status      string                      `json:"status"`
	Candidates  []string                    `json:"candidates,omitempty"`
	Repository  contract.RepositoryIdentity `json:"repository"`
	Declaration json.RawMessage             `json:"declaration"`
}

func (s *Service) ScanWorkspace(ctx context.Context, directory string, selection contract.ScanSelection) (WorkspaceScan, error) {
	root, err := filepath.Abs(directory)
	if err != nil {
		return WorkspaceScan{}, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return WorkspaceScan{}, err
	}
	members, err := workspaceMembers(root)
	if err != nil {
		return WorkspaceScan{}, err
	}
	result := WorkspaceScan{SchemaVersion: contract.ReportVersion, Root: root, Projects: make([]WorkspaceProject, 0, len(members))}
	for _, path := range members {
		project := WorkspaceProject{Path: filepath.ToSlash(path)}
		document, err := s.ScanFacts(ctx, filepath.Join(root, path), selection.Stack, selection.LOC, selection.EngineFacts, false)
		if ctx.Err() != nil {
			return WorkspaceScan{}, ctx.Err()
		}
		if err == nil {
			err = json.Unmarshal(document.Data, &project.Facts)
		}
		if err != nil {
			project.Error = err.Error()
		}
		result.Projects = append(result.Projects, project)
	}
	if slices.Contains(selection.EngineFacts, "dependencies") {
		result.Dependencies = &WorkspaceDependencies{
			Scope:       "Only library references declared by build configuration are used for cross-project matching (currently CMake Git references).",
			Unsupported: []string{"IPC", "REST APIs"},
			References:  matchWorkspaceReferences(result.Projects),
		}
	}
	return result, nil
}

func workspaceMembers(root string) ([]string, error) {
	const configFile = ".scryer/workspace.yaml"
	data, err := os.ReadFile(filepath.Join(root, configFile))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", configFile, err)
	}
	var config struct {
		Projects []string `yaml:"projects"`
	}
	if err := yaml.UnmarshalStrict(data, &config); err != nil {
		return nil, fmt.Errorf("parse %s: %w", configFile, err)
	}
	if len(config.Projects) == 0 {
		return nil, fmt.Errorf("%s must list at least one project", configFile)
	}
	members := make([]string, 0, len(config.Projects))
	seen := map[string]bool{}
	for _, path := range config.Projects {
		if strings.TrimSpace(path) == "" || !filepath.IsLocal(path) {
			return nil, fmt.Errorf("project path %q must be relative to the workspace", path)
		}
		// Resolve existing paths to reject links outside the workspace and duplicate checkouts.
		clean := filepath.Clean(path)
		resolved, err := filepath.EvalSymlinks(filepath.Join(root, clean))
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("resolve project %q: %w", path, err)
		}
		if err == nil {
			clean, err = filepath.Rel(root, resolved)
			if err != nil || !filepath.IsLocal(clean) {
				return nil, fmt.Errorf("project %q resolves outside the workspace", path)
			}
		}
		if seen[clean] {
			return nil, fmt.Errorf("duplicate workspace project %q", path)
		}
		seen[clean] = true
		members = append(members, clean)
	}
	return members, nil
}

func matchWorkspaceReferences(projects []WorkspaceProject) []WorkspaceReference {
	byID := make(map[string][]string)
	for _, project := range projects {
		if project.Facts != nil && project.Facts.Repository.ID != "" {
			id := project.Facts.Repository.ID
			byID[id] = append(byID[id], project.Path)
		}
	}
	references := []WorkspaceReference{}
	for _, project := range projects {
		if project.Facts == nil {
			continue
		}
		for _, outcome := range project.Facts.EngineFacts {
			if outcome.Fact != "dependencies" || outcome.Engine != "cpp" || outcome.Status != "collected" {
				continue
			}
			var dependencyFact struct {
				CMake struct {
					ExternalDependencies []json.RawMessage `json:"externalDependencies"`
				} `json:"cmake"`
			}
			if json.Unmarshal(outcome.Data, &dependencyFact) != nil {
				continue
			}
			for _, declaration := range dependencyFact.CMake.ExternalDependencies {
				var evidence struct {
					Repository contract.RepositoryIdentity `json:"repository"`
				}
				if json.Unmarshal(declaration, &evidence) != nil || evidence.Repository.ID == "" {
					continue
				}
				var candidates []string
				for _, candidate := range byID[evidence.Repository.ID] {
					if candidate != project.Path {
						candidates = append(candidates, candidate)
					}
				}
				ref := WorkspaceReference{From: project.Path, Repository: evidence.Repository, Declaration: declaration}
				switch len(candidates) {
				case 0:
					ref.Status = "external"
				case 1:
					ref.Status, ref.To = "local", candidates[0]
				default:
					ref.Status, ref.Candidates = "ambiguous", candidates
				}
				references = append(references, ref)
			}
		}
	}
	return references
}
