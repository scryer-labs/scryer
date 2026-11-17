package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"strings"

	"github.com/edwardnoaland/scryer/internal/contract"
)

type coverageView struct {
	contract.CoverageReport
	Evidence *contract.CoverageEvidence
	Rows     [][]string
}

func coverageFor(document contract.Document) (coverageView, error) {
	var view coverageView
	if err := document.Validate(); err != nil {
		return view, err
	}
	if err := json.Unmarshal(document.Data, &view.CoverageReport); err != nil {
		return view, err
	}
	if len(view.Coverage) == 0 {
		return view, nil
	}
	if err := json.Unmarshal(view.Coverage, &view.Evidence); err != nil {
		return view, err
	}
	for _, file := range view.Evidence.Files {
		row := []string{file.Path}
		for _, metric := range []*contract.CoverageCount{file.Summary.Lines, file.Summary.Functions, file.Summary.Regions} {
			value := "unknown"
			if metric != nil {
				value = fmt.Sprintf("%d / %d", metric.Covered, metric.Count)
			}
			row = append(row, value)
		}
		view.Rows = append(view.Rows, row)
	}
	return view, nil
}

func writeCoverage(out io.Writer, document contract.Document) error {
	view, err := coverageFor(document)
	if err != nil {
		return err
	}
	var output strings.Builder
	fmt.Fprintf(&output, "Coverage: %s\nProject: %s\n", view.Status, view.Root)
	if view.Message != "" {
		fmt.Fprintln(&output, view.Message)
	}
	for _, run := range view.Runs {
		fmt.Fprintf(&output, "%s: %s (%d profiles)\n", run.Name, run.Status, run.Profiles)
		if run.Message != "" {
			fmt.Fprintln(&output, run.Message)
		}
	}
	if evidence := view.Evidence; evidence != nil {
		fmt.Fprintf(&output, "Source match: %s (%s)\n", evidence.SourceMatch, evidence.Granularity)
		for _, row := range view.Rows {
			fmt.Fprintf(&output, "%s: %s lines, %s functions, %s regions\n", row[0], row[1], row[2], row[3])
		}
		for _, note := range evidence.Notes {
			fmt.Fprintln(&output, note)
		}
	}
	_, err = io.WriteString(out, output.String())
	return err
}

func coverageMarkdown(document contract.Document) ([]byte, error) {
	view, err := coverageFor(document)
	if err != nil {
		return nil, err
	}
	var output strings.Builder
	section(&output, 1, "Runtime coverage")
	fmt.Fprintf(&output, "**Project** %s\n\n**Status** %s\n\n", escape(view.Root), escape(view.Status))
	if view.Message != "" {
		fmt.Fprintln(&output, escape(view.Message))
	}
	if len(view.Runs) != 0 {
		section(&output, 2, "Test runs")
		var rows [][]string
		for _, run := range view.Runs {
			rows = append(rows, []string{run.Name, run.Status, fmt.Sprint(run.Profiles), run.Message})
		}
		table(&output, []string{"Test", "Status", "Profiles", "Diagnostic"}, rows)
	}
	if evidence := view.Evidence; evidence != nil {
		fmt.Fprintf(&output, "**Artifact** %s\n\n**Source match** %s (%s)\n\n",
			escape(evidence.Artifact), escape(evidence.SourceMatch), escape(evidence.Granularity))
		if len(view.Rows) > 0 {
			table(&output, []string{"File", "Lines", "Functions", "Regions"}, view.Rows)
		}
		for _, note := range evidence.Notes {
			fmt.Fprintf(&output, "- %s\n", escape(note))
		}
	}
	return []byte(output.String()), nil
}

var coveragePage = template.Must(template.New("coverage").Parse(`<!doctype html>
<html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width">
<title>Runtime coverage</title>
<style>
:root { color-scheme: light dark }
body { font: 1rem/1.5 system-ui, sans-serif; max-width: 75ch; margin: 2rem auto; padding: 0 1rem; overflow-wrap: anywhere }
table { border-collapse: collapse; width: 100%; margin-top: .5rem }
th, td { text-align: left; padding: .4rem .6rem; border-bottom: 1px solid GrayText }
input { font: inherit; margin: .5rem 0; max-width: 90% }
</style>
<main>
<h1>Runtime coverage</h1>
<p><strong>Project</strong> {{.Root}}</p><p><strong>Status</strong> {{.Status}}</p>
{{if .Message}}<p>{{.Message}}</p>{{end}}
{{if .Runs}}<h2>Test runs</h2>
<ul>{{range .Runs}}<li>{{.Name}}: {{.Status}} ({{.Profiles}} profiles)
{{if .Message}}<details><summary>Diagnostic</summary><pre>{{.Message}}</pre></details>{{end}}</li>{{end}}</ul>
{{end}}
{{with .Evidence}}
<p><strong>Source match</strong> {{.SourceMatch}} ({{.Granularity}})</p>
<details><summary>Artifact and evidence notes</summary><p>{{.Artifact}}</p><p>SHA-256 {{.SHA256}}</p>
<ul>{{range .Notes}}<li>{{.}}</li>{{end}}</ul></details>
{{end}}
{{if .Rows}}
<label>Filter files <input id="filter" type="search"></label>
<table><thead><tr><th scope="col">File</th><th scope="col">Lines</th><th scope="col">Functions</th><th scope="col">Regions</th></tr></thead>
<tbody>{{range .Rows}}<tr>{{range .}}<td>{{.}}</td>{{end}}</tr>{{end}}</tbody></table>
<script>
const filter = document.querySelector("#filter");
const rows = document.querySelectorAll("tbody tr");
filter.addEventListener("input", () => {
  const value = filter.value.toLowerCase();
  for (const row of rows) row.hidden = !row.cells[0].textContent.toLowerCase().includes(value);
});
</script>
{{end}}
</main></html>`))

func coverageHTML(document contract.Document) ([]byte, error) {
	view, err := coverageFor(document)
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	err = coveragePage.Execute(&output, view)
	return output.Bytes(), err
}
