package report

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/edwardnoaland/scryer/internal/contract"
)

func ProjectScanHTML(document contract.Document) ([]byte, error) {
	if err := document.Validate(); err != nil {
		return nil, err
	}
	return ScanHTML(struct {
		Projects []map[string]json.RawMessage `json:"projects"`
	}{Projects: []map[string]json.RawMessage{{"facts": document.Data}}})
}

// ScanHTML renders the same embedded canvas for one project or workspace members.
func ScanHTML(scan any) ([]byte, error) {
	data, err := json.Marshal(scan)
	if err != nil {
		return nil, err
	}
	page, err := assets.ReadFile("assets/scan.html")
	if err != nil {
		return nil, err
	}
	style, err := assets.ReadFile("assets/scan.css")
	if err != nil {
		return nil, err
	}
	script, err := assets.ReadFile("assets/scan.js")
	if err != nil {
		return nil, err
	}
	result := strings.NewReplacer("__SCAN_CSS__", string(style), "__SCAN_JS__", string(script), "__SCAN_DATA__", string(data))
	return []byte(result.Replace(string(page))), nil
}

func WorkspaceScanMarkdown(scan any) ([]byte, error) {
	raw, err := json.Marshal(scan)
	if err != nil {
		return nil, err
	}
	var data Object
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	var b strings.Builder
	section(&b, 1, "Workspace scan")
	fmt.Fprintf(&b, "%s\n\n", escape(str(data, "root")))
	if deps := field(data, "dependencies"); deps != nil {
		section(&b, 2, "Workspace references")
		fmt.Fprintf(&b, "%s\n\n", escape(str(deps, "scope")))
		fmt.Fprintf(&b, "Unsupported dependency types: %s\n\n", escape(join(deps["unsupported"])))
		var entries [][]string
		for _, ref := range objects(deps["references"]) {
			target := optional(ref, "to")
			if target == "" {
				target = str(field(ref, "repository"), "id")
			}
			entries = append(entries, []string{str(ref, "from"), target, str(ref, "status"), cppLocation(field(ref, "declaration"))})
		}
		table(&b, []string{"From", "To", "Status", "Declaration"}, entries)
	}
	for _, project := range objects(data["projects"]) {
		if facts := field(project, "facts"); facts != nil {
			b.WriteString(projectFactsMarkdown(facts))
		} else {
			section(&b, 2, str(project, "path"))
			fmt.Fprintf(&b, "%s\n\n", escape(str(project, "error")))
		}
	}
	return []byte(b.String()), nil
}
