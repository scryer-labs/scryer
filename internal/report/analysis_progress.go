package report

import (
	"fmt"
	"io"
	"strings"
)

func writeAnalysisProgress(out io.Writer, data Object) {
	if status, ok := data["status"]; ok {
		fmt.Fprintln(out, "Analysis status:", status)
	}
	for _, outcome := range objects(data["engineOutcomes"]) {
		fmt.Fprintf(out, "  %s [%s]\n", str(outcome, "engine"), str(outcome, "status"))
		for _, stage := range objects(outcome["stages"]) {
			fmt.Fprintf(out, "    %s: %s %s\n", str(stage, "name"), str(stage, "status"), optional(stage, "message"))
		}
	}
	if len(objects(data["changes"])) == 0 && data["status"] == "collected" {
		fmt.Fprintln(out, "No changed code symbols detected by the collected engines (documentation-only changes can produce this result).")
	}
}
func analysisProgressMarkdown(data Object) string {
	if data["status"] == nil {
		return ""
	}
	var output strings.Builder
	fmt.Fprintf(&output, "**Analysis status: %s**\n\n", escape(str(data, "status")))
	var rows [][]string
	for _, outcome := range objects(data["engineOutcomes"]) {
		for _, stage := range objects(outcome["stages"]) {
			rows = append(rows, []string{str(outcome, "engine"), str(stage, "name"), str(stage, "status"), optional(stage, "message")})
		}
	}
	table(&output, []string{"Engine", "Stage", "Status", "Diagnostic"}, rows)
	if len(objects(data["changes"])) == 0 && data["status"] == "collected" {
		output.WriteString("No changed code symbols detected by the collected engines; documentation-only changes can produce this result.\n\n")
	}
	return output.String()
}
