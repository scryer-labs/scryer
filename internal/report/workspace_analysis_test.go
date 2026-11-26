package report

import (
	"encoding/base64"
	"encoding/json"
	"html"
	"regexp"
	"strings"
	"testing"

	"github.com/edwardnoaland/scryer/internal/contract"
)

func TestWorkspaceAnalysisPreservesUnknownProjectFieldsAndEscapesDiagnostics(t *testing.T) {
	result := contract.WorkspaceAnalysis{SchemaVersion: 1, Root: "/workspace", Status: "failed", Counts: map[string]int{"failed": 1}, Projects: []contract.ProjectAnalysis{{Path: "<project>", Status: "failed", Error: `<img src=x onerror=alert(1)>`, ReportOperation: contract.CoverageOperation, Report: json.RawMessage(`{"schemaVersion":1,"root":"/repo","status":"unavailable","engineExtension":{"keep":"original"}}`)}}}
	markdown, err := WorkspaceAnalysisMarkdown(result)
	if err != nil || !strings.Contains(string(markdown), `"engineExtension"`) {
		t.Fatalf("lost project evidence: %s %v", markdown, err)
	}
	page, err := WorkspaceAnalysisHTML(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(page), `<img src=x`) || !strings.Contains(string(page), `sandbox="allow-scripts allow-downloads"`) {
		t.Fatal("unescaped diagnostic or unisolated child report")
	}
	download := regexp.MustCompile(`data:application/json;base64,([A-Za-z0-9+/=]+)`).FindSubmatch(page)
	if len(download) != 2 {
		t.Fatal("missing complete JSON download")
	}
	raw, err := base64.StdEncoding.DecodeString(string(download[1]))
	if err != nil {
		t.Fatal(err)
	}
	var restored contract.WorkspaceAnalysis
	if err = json.Unmarshal(raw, &restored); err != nil || !strings.Contains(string(restored.Projects[0].Report), `"engineExtension"`) {
		t.Fatal("lost unknown fields")
	}
	frame := regexp.MustCompile(`srcdoc="([^"]*)"`).FindSubmatch(page)
	if len(frame) != 2 || !strings.Contains(html.UnescapeString(string(frame[1])), "<!doctype html>") {
		t.Fatal("missing independently rendered child report")
	}
}

func Test_should_distinguish_unassessed_context_when_no_changed_symbols_are_projected(t *testing.T) {
	project := contract.ProjectAnalysis{Status: "collected", ReportOperation: "analyze", Report: json.RawMessage(`{"changes":[],"matching":{"unresolvedChanges":["Changed go.mod"]}}`)}
	if summary := workspaceChangeSummary(project); !strings.Contains(summary, "unassessed") {
		t.Fatal(summary)
	}
	project.Report = json.RawMessage(`{"changes":[],"matching":{"unresolvedChanges":[]}}`)
	if summary := workspaceChangeSummary(project); !strings.Contains(summary, "supplied projection") || strings.Contains(summary, "unassessed") {
		t.Fatal(summary)
	}
}
