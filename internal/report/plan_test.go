package report

import (
	"strings"
	"testing"

	"github.com/edwardnoaland/scryer/internal/workspace"
)

func TestDraftTemplatePreservesIDsAndDoesNotInventSteps(t *testing.T) {
	facts := workspace.Facts{Projects: []workspace.Project{{ID: "a: #\nquoted", RepositoryPath: "org/repo"}, {ID: "blocked"}}, Dependencies: []workspace.Dependency{{Source: "blocked", Target: "blocked", Internal: true}}}
	doc := workspace.SuggestPhases(facts)
	data, err := PlanTemplate(doc)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, part := range []string{"status: draft", "type: mod-cli", `"a: #\nquoted"`, "steps: []", "path: null", "repositoryPath: \"org/repo\""} {
		if !strings.Contains(text, part) {
			t.Fatalf("missing %q: %s", part, text)
		}
	}
	if strings.Contains(text, `repo: "blocked"`) {
		t.Fatal("blocked repo included in runnable phases")
	}
	if _, err = PlanTemplate(workspace.NewDocument(facts)); err == nil {
		t.Fatal("accepted facts without suggestions")
	}
}
