package report

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/edwardnoaland/scryer/internal/workspace"
)

func TestWorkspaceOutputStableAndHTMLSafe(t *testing.T) {
	facts := workspace.Facts{Projects: []workspace.Project{{ID: "b"}, {ID: "</script><img onerror=alert(1)>"}}, Dependencies: []workspace.Dependency{}}
	document := workspace.NewDocument(facts)
	data, err := Workspace(document, "json")
	if err != nil {
		t.Fatal(err)
	}
	var decoded workspace.Document
	if err = json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.SchemaVersion != 1 || decoded.Direction != "consumer-to-dependency" || len(decoded.Facts.Projects) != 2 {
		t.Fatal(decoded)
	}
	html, err := Workspace(document, "html")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(html), "</script><img onerror") || strings.Contains(string(html), "__DATA__") {
		t.Fatal("unsafe or unsubstituted HTML")
	}
	if !strings.Contains(string(html), "setPointerCapture") || !strings.Contains(string(html), "application/json") {
		t.Fatal("missing interaction or document")
	}
}
