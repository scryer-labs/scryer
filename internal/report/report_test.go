package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edwardnoaland/scryer/internal/contract"
)

func fixtureDocument() contract.Document {
	return contract.Document{Operation: "analyze", Data: json.RawMessage(`{"schemaVersion":1,"generatedAt":"now","repository":"</script><script>alert(1)</script>","beforeSha":"before","afterSha":"after","files":[],"changes":[],"before":{"nodes":[],"edges":[],"boundaries":[],"notes":[],"unresolvedChanges":[],"totalBoundaryCount":0},"after":{"nodes":[{"id":"a","signature":"one.Example#target()","role":"PRODUCTION","impact":"CHANGED","path":"src/Example.java","line":1},{"id":"b","signature":"two.Example#target()","role":"TEST","impact":"CALLER","path":"test/Example.java","line":2}],"edges":[{"caller":"b","callee":"a","kind":"DIRECT"},{"caller":"a","callee":"b","kind":"REFLECTION"}],"boundaries":[{"caller":"a","path":"src/Example.java","line":1,"reason":"unknown | <value>","expression":"__SCRYER_DATA__"}],"notes":[],"unresolvedChanges":[],"totalBoundaryCount":1},"execution":{"status":"SKIPPED","command":[],"notes":[]},"evidence":{"testRecords":null,"tests":[],"coverage":[],"artifacts":[],"notes":[]},"matching":{"datasets":[{"artifact":null,"methods":[{"symbol":"a","signature":"one.Example#target()","path":"src/Example.java","line":1,"status":"UNKNOWN","impact":"CHANGED","reason":"no coverage","instructions":null,"branches":null,"routes":[]}],"counts":{"UNKNOWN":1},"withHits":0,"assessed":0,"methodExecutionPercent":null}],"removed":[],"unresolvedChanges":[],"notes":[]},"notes":[]}`)}
}
func TestHTMLRemainsOfflineAndScriptSafe(t *testing.T) {
	data, err := HTML(fixtureDocument())
	if err != nil {
		t.Fatal(err)
	}
	html := string(data)
	if strings.Contains(html, `</script><script>alert(1)</script>`) {
		t.Fatal("unsafe embedded source")
	}
	for _, needle := range []string{`\u003c/script\u003e`, "data:image/png;base64,", "ScryerReport", "__SCRYER_DATA__"} {
		if !strings.Contains(html, needle) {
			t.Fatalf("missing %s", needle)
		}
	}
	if strings.Contains(html, "__SCRYER_STYLE__") {
		t.Fatal("unfilled asset")
	}
}
func TestMarkdownKeepsUnknownFullIdentitiesAndCycleEdges(t *testing.T) {
	data, err := Markdown(fixtureDocument(), false)
	if err != nil {
		t.Fatal(err)
	}
	markdown := string(data)
	for _, needle := range []string{"UNKNOWN", "**unknown** (0/0 assessed", "one.Example#target()", "two.Example#target()", "n1 -->|DIRECT| n0", "n0 -->|REFLECTION| n1", "unknown \\| &lt;value&gt;", "class n1 test"} {
		if !strings.Contains(markdown, needle) {
			t.Fatalf("missing %q", needle)
		}
	}
	if strings.Contains(markdown, "0.0%") {
		t.Fatal("unknown turned into zero")
	}
}
func TestAtomicReportPreservesExistingFileOnDestinationFailure(t *testing.T) {
	directory := t.TempDir()
	file := filepath.Join(directory, "report.md")
	if err := WriteFile(file, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(file, []byte("second")); err != nil {
		t.Fatal(err)
	}
	actual, _ := os.ReadFile(file)
	if string(actual) != "second" {
		t.Fatal("replacement failed")
	}
	target := filepath.Join(directory, "existing-directory")
	os.Mkdir(target, 0700)
	os.WriteFile(filepath.Join(target, "keep"), []byte("keep"), 0600)
	if err := WriteFile(target, []byte("bad")); err == nil {
		t.Fatal("replaced directory")
	}
	if _, err := os.Stat(filepath.Join(target, "keep")); err != nil {
		t.Fatal("lost existing destination")
	}
}
