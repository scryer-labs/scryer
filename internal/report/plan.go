package report

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/edwardnoaland/scryer/internal/workspace"
)

// PlanTemplate exports configuration for a future platform, never an executable
// Scryer contract. IDs are identities; SCM repository paths are not local paths.
func PlanTemplate(document workspace.Document) ([]byte, error) {
	if document.Suggestion == nil {
		return nil, fmt.Errorf("plan template requires phase suggestions")
	}
	quote := func(value string) string { data, _ := json.Marshal(value); return string(data) }
	var out strings.Builder
	out.WriteString("# Draft platform configuration, not a Moderne or Scryer executable format.\n# Fill local paths and recipe steps, review ordering and validation/publication gates.\n# Phases are suggestions. No upgrade strategy or recipe is selected automatically.\nversion: 1\nstatus: draft\nprovider:\n  type: mod-cli\nrepositories:\n")
	for _, project := range document.Facts.Projects {
		fmt.Fprintf(&out, "  - id: %s\n    path: null # TODO: local checkout path\n", quote(project.ID))
		if project.RepositoryPath != "" {
			fmt.Fprintf(&out, "    repositoryPath: %s\n", quote(project.RepositoryPath))
		}
	}
	out.WriteString("suggestion:\n  rule: " + quote(document.Suggestion.Rule) + "\n")
	for _, list := range []struct {
		name   string
		values []string
	}{{"cyclic", document.Suggestion.Cyclic}, {"blocked", document.Suggestion.Blocked}} {
		fmt.Fprintf(&out, "  %s:", list.name)
		if len(list.values) == 0 {
			out.WriteString(" []\n")
		} else {
			out.WriteByte('\n')
			for _, id := range list.values {
				fmt.Fprintf(&out, "    - %s\n", quote(id))
			}
		}
	}
	if len(document.Suggestion.Phases) == 0 {
		out.WriteString("phases: []\n")
	} else {
		out.WriteString("phases:\n")
		for _, phase := range document.Suggestion.Phases {
			fmt.Fprintf(&out, "  - id: %s\n    repositories:\n", quote(phase.ID))
			for _, id := range phase.Projects {
				fmt.Fprintf(&out, "      - repo: %s\n        steps: [] # TODO: platform recipe steps and validation\n", quote(id))
			}
		}
	}
	return []byte(out.String()), nil
}

func PlanFormat(output string) (string, error) {
	lower := strings.ToLower(output)
	if strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") {
		return "yaml", nil
	}
	format, err := WorkspaceFormat(output)
	if err != nil {
		return "", fmt.Errorf("refactor-plan -o must use .json, .html, .yaml or .yml")
	}
	return format, nil
}
