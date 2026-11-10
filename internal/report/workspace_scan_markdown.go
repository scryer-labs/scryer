package report

import (
	"encoding/json"
	"fmt"
	"strings"
)

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
