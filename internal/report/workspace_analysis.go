package report

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"strings"

	"github.com/edwardnoaland/scryer/internal/contract"
)

func WorkspaceAnalysisTerminal(result contract.WorkspaceAnalysis, out io.Writer, verbose, color bool) error {
	fmt.Fprintf(out, "Workspace analysis: %s\nRoot: %s\n%d collected, %d failed, %d unsupported\n", result.Status, result.Root, result.Counts["collected"], result.Counts["failed"], result.Counts["unsupported"])
	for _, project := range result.Projects {
		fmt.Fprintf(out, "\n%s [%s]\n", project.Path, project.Status)
		if project.Error != "" {
			fmt.Fprintln(out, project.Error)
		}
		if len(project.Report) == 0 {
			continue
		}
		var comparison struct {
			BeforeSHA   string
			AfterSHA    string
			AfterSource struct{ Kind string }
		}
		if err := json.Unmarshal(project.Report, &comparison); err != nil {
			return err
		}
		if comparison.BeforeSHA != "" {
			fmt.Fprintf(out, "  Before: %s\n  After: %s", comparison.BeforeSHA, comparison.AfterSHA)
			if comparison.AfterSource.Kind != "" {
				fmt.Fprintf(out, " [%s]", comparison.AfterSource.Kind)
			}
			fmt.Fprintln(out)
		}
		if verbose {
			if err := Terminal(project.Document(), out, true, color); err != nil {
				return err
			}
		}
	}
	for _, note := range result.Notes {
		fmt.Fprintln(out, "\nNote:", note)
	}
	return nil
}

func WorkspaceAnalysisMarkdown(result contract.WorkspaceAnalysis) ([]byte, error) {
	var output strings.Builder
	section(&output, 1, "Workspace analysis")
	fmt.Fprintf(&output, "Root: %s\n\nStatus: **%s**\n\n", escape(result.Root), escape(result.Status))
	var rows [][]string
	for _, project := range result.Projects {
		rows = append(rows, []string{project.Path, project.Status, project.Error})
	}
	table(&output, []string{"Project", "Status", "Diagnostic"}, rows)
	for _, note := range result.Notes {
		fmt.Fprintf(&output, "\n%s\n", escape(note))
	}
	for _, project := range result.Projects {
		output.WriteString("\n---\n\n")
		section(&output, 2, project.Path)
		fmt.Fprintf(&output, "Status: %s\n\n", escape(project.Status))
		if project.Error != "" {
			fmt.Fprintf(&output, "%s\n\n", escape(project.Error))
		}
		if len(project.Report) == 0 {
			continue
		}
		data, err := Markdown(project.Document())
		if err != nil {
			return nil, err
		}
		output.Write(data)
	}
	// The full JSON also preserves engine-specific fields without dedicated views.
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}
	section(&output, 2, "Complete workspace JSON")
	// Four backticks avoid closing the block on source snippets containing triples.
	fence := "````"
	for strings.Contains(string(raw), fence) {
		fence += "`"
	}
	fmt.Fprintf(&output, "\n%sjson\n%s\n%s\n", fence, raw, fence)
	return []byte(output.String()), nil
}

type workspaceAnalysisPage struct {
	Result   contract.WorkspaceAnalysis
	JSON     template.URL
	Projects []workspaceAnalysisFrame
}
type workspaceAnalysisFrame struct{ Path, Status, Error, HTML string }

func WorkspaceAnalysisHTML(result contract.WorkspaceAnalysis) ([]byte, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	page := workspaceAnalysisPage{Result: result, JSON: template.URL("data:application/json;base64," + base64.StdEncoding.EncodeToString(raw))}
	for _, project := range result.Projects {
		frame := workspaceAnalysisFrame{Path: project.Path, Status: project.Status, Error: project.Error}
		if len(project.Report) > 0 {
			child, err := HTML(project.Document())
			if err != nil {
				return nil, err
			}
			frame.HTML = string(child)
		}
		page.Projects = append(page.Projects, frame)
	}
	var output bytes.Buffer
	err = workspaceAnalysisTemplate.Execute(&output, page)
	return output.Bytes(), err
}

var workspaceAnalysisTemplate = template.Must(template.New("workspace-analysis").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Scryer workspace analysis</title>
<style>body{margin:0;background:#101827;color:#e4eaf4;font:16px system-ui,sans-serif}main{max-width:1500px;margin:auto;padding:32px}h1{margin-bottom:8px}p{overflow-wrap:anywhere}a{color:#9fc4ff}table{width:100%;border-collapse:collapse;margin:24px 0}th,td{text-align:left;padding:12px;border-bottom:1px solid #334155}details{border:1px solid #334155;border-radius:12px;margin:18px 0;overflow:hidden}summary{padding:18px;cursor:pointer}iframe{display:block;width:100%;height:900px;border:0;background:#101827}.diagnostic{padding:0 18px;color:#fbbf24}.status{color:#9fc4ff}</style></head>
<body><main><h1>Workspace analysis</h1><p>{{.Result.Root}}</p><p class="status">{{.Result.Status}} · {{index .Result.Counts "collected"}} collected · {{index .Result.Counts "failed"}} failed · {{index .Result.Counts "unsupported"}} unsupported</p>
<a href="{{.JSON}}" download="workspace-analysis.json">Download complete JSON</a>
{{range .Result.Notes}}<p>{{.}}</p>{{end}}
<table><thead><tr><th>Project</th><th>Status</th><th>Diagnostic</th></tr></thead><tbody>{{range .Projects}}<tr><td>{{.Path}}</td><td>{{.Status}}</td><td>{{.Error}}</td></tr>{{end}}</tbody></table>
{{range .Projects}}<details><summary>{{.Path}} · {{.Status}}</summary>{{if .Error}}<p class="diagnostic">{{.Error}}</p>{{end}}{{if .HTML}}<iframe title="{{.Path}} project report" loading="lazy" sandbox="allow-scripts allow-downloads" srcdoc="{{.HTML}}"></iframe>{{end}}</details>{{end}}
</main></body></html>`))
