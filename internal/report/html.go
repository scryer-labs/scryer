package report

import (
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/edwardnoaland/scryer/internal/contract"
)

//go:embed assets/*
var assets embed.FS

func HTML(document contract.Document) ([]byte, error) {
	if document.Operation == contract.CoverageOperation {
		return coverageHTML(document)
	}
	if document.Operation != "analyze" {
		return nil, fmt.Errorf("HTML currently supports analyze reports")
	}
	decoded, err := decode(document)
	if err != nil {
		return nil, err
	}
	read := func(name string) (string, error) {
		bytes, err := assets.ReadFile("assets/" + name)
		return string(bytes), err
	}
	template, err := read("report.html")
	if err != nil {
		return nil, err
	}
	stack := document.Stack
	if stack == "" {
		stack = "unknown"
	}
	template = strings.ReplaceAll(template, "__SCRYER_STACK__", escape(strings.ToUpper(stack)))
	for token, name := range map[string]string{"__SCRYER_STYLE__": "report.css", "__SCRYER_APP__": "report.js"} {
		value, err := read(name)
		if err != nil {
			return nil, err
		}
		template = strings.ReplaceAll(template, token, value)
	}
	model, err := read("graph-model.js")
	if err != nil {
		return nil, err
	}
	graph, err := read("graph.js")
	if err != nil {
		return nil, err
	}
	template = strings.ReplaceAll(template, "__SCRYER_GRAPH__", model+"\n"+graph)
	logo, err := assets.ReadFile("assets/scryer-logo.png")
	if err != nil {
		return nil, err
	}
	template = strings.ReplaceAll(template, "__SCRYER_LOGO__", "data:image/png;base64,"+base64.StdEncoding.EncodeToString(logo))
	view := viewFor(document.Stack).analysis
	runtimeRows := [][2]string{}
	if view.runtimeField != "" {
		runtimeRows = append(runtimeRows, [2]string{view.htmlRuntimeLabel, view.runtimeField})
	}
	runtimeJSON, err := json.Marshal(runtimeRows)
	if err != nil {
		return nil, err
	}
	template = strings.ReplaceAll(template, "__SCRYER_EXECUTION_DETAILS__", string(runtimeJSON))
	if coverage := decoded["coverageReport"]; coverage != nil {
		coverageDocument, err := contract.NewDocument(document.Stack, contract.CoverageOperation, coverage)
		if err != nil {
			return nil, err
		}
		page, err := coverageHTML(coverageDocument)
		if err != nil {
			return nil, err
		}
		link := `<section class="limits"><h2>Full coverage report</h2><a download="coverage.html" href="data:text/html;base64,` +
			base64.StdEncoding.EncodeToString(page) + `">Save full coverage HTML</a></section>`
		template = strings.Replace(template, "<footer>", link+"<footer>", 1)
	}
	data, err := JSON(document)
	if err != nil {
		return nil, err
	}
	safe := strings.NewReplacer("&", "\\u0026", "<", "\\u003c", ">", "\\u003e", "\u2028", "\\u2028", "\u2029", "\\u2029").Replace(string(data))
	// Report data is inserted last so source strings containing template tokens remain literal.
	return []byte(strings.ReplaceAll(template, "__SCRYER_DATA__", safe)), nil
}
