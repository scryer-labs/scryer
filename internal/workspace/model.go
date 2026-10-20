package workspace

import (
	"cmp"
	"slices"
)

type Project struct {
	ID             string `json:"id"`
	RepositoryPath string `json:"repositoryPath,omitempty"`
}
type Dependency struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Internal bool   `json:"internal"`
	// Empty string denotes unspecified scope, not an observed production scope.
	Scopes []string `json:"scopes"`
}
type Facts struct {
	Projects     []Project    `json:"projects"`
	Dependencies []Dependency `json:"dependencies"`
}
type Document struct {
	SchemaVersion int      `json:"schemaVersion"`
	Kind          string   `json:"kind"`
	Direction     string   `json:"edgeDirection"`
	Facts         Facts    `json:"facts"`
	Notes         []string `json:"notes"`
}

func NewDocument(facts Facts) Document {
	// Copy before normalizing; reporting must not mutate collector results.
	normalized := Facts{Projects: append([]Project{}, facts.Projects...), Dependencies: make([]Dependency, len(facts.Dependencies))}
	for i, edge := range facts.Dependencies {
		edge.Scopes = append([]string{}, edge.Scopes...)
		slices.Sort(edge.Scopes)
		normalized.Dependencies[i] = edge
	}
	slices.SortFunc(normalized.Projects, func(a, b Project) int { return cmp.Compare(a.ID, b.ID) })
	slices.SortFunc(normalized.Dependencies, func(a, b Dependency) int {
		if c := cmp.Compare(a.Source, b.Source); c != 0 {
			return c
		}
		return cmp.Compare(a.Target, b.Target)
	})
	return Document{SchemaVersion: 1, Kind: "workspace-dependencies", Direction: "consumer-to-dependency", Facts: normalized, Notes: []string{
		"Declared dependencies only; not runtime calls, compatibility proof or test execution evidence.",
		"Internal means the target matches a project in the supplied CSV; external targets are retained.",
		"Rows are combined by source and target; observed scopes are preserved, versions and other columns are not modeled.",
	}}
}
