package report

import (
	"embed"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/edwardnoaland/scryer/internal/contract"
)

//go:embed assets/*
var assets embed.FS

func HTML(document contract.Document) ([]byte, error) {
	if document.Operation != "analyze" {
		return nil, fmt.Errorf("HTML currently supports analyze reports")
	}
	if _, err := decode(document); err != nil {
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
	data, err := JSON(document)
	if err != nil {
		return nil, err
	}
	safe := strings.NewReplacer("&", "\\u0026", "<", "\\u003c", ">", "\\u003e", "\u2028", "\\u2028", "\u2029", "\\u2029").Replace(string(data))
	// Report data is inserted last so source strings containing template tokens remain literal.
	return []byte(strings.ReplaceAll(template, "__SCRYER_DATA__", safe)), nil
}
