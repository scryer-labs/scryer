// Package report renders common facts. It does not execute analyzers or infer new evidence.
package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/edwardnoaland/scryer/internal/contract"
)

type Object = map[string]any

func object(value any) Object               { result, _ := value.(map[string]any); return result }
func field(value Object, key string) Object { return object(value[key]) }
func objects(value any) []Object {
	var result []Object
	for _, item := range array(value) {
		if item := object(item); item != nil {
			result = append(result, item)
		}
	}
	return result
}
func array(value any) []any { result, _ := value.([]any); return result }
func text(value any) string {
	if value == nil {
		return "unknown"
	}
	return fmt.Sprint(value)
}
func str(value Object, key string) string { return text(value[key]) }
func join(value any) string {
	items := array(value)
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = text(item)
	}
	if len(parts) == 0 {
		return "unknown"
	}
	return strings.Join(parts, ", ")
}
func boolean(value any) bool             { result, _ := value.(bool); return result }
func rows(values ...[]string) [][]string { return values }
func decode(document contract.Document) (Object, error) {
	if err := document.Validate(); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(document.Data))
	decoder.UseNumber()
	var result Object
	err := decoder.Decode(&result)
	return result, err
}
func JSON(document contract.Document) ([]byte, error) {
	var output bytes.Buffer
	if err := json.Indent(&output, document.Data, "", "  "); err != nil {
		return nil, err
	}
	output.WriteByte('\n')
	return output.Bytes(), nil
}
func Terminal(document contract.Document, out io.Writer, dependencies, tree, verbose, color bool) error {
	data, err := decode(document)
	if err != nil {
		return err
	}
	if document.Operation == contract.ProjectFactsOperation {
		return writeProjectFacts(out, data)
	}
	if document.Operation == "scan" {
		viewFor(document.Stack).scanTerminal(data, out, dependencies, tree, color)
	} else {
		analyzeTerminal(data, out, verbose, color, viewFor(document.Stack).analysis)
	}
	return nil
}
func Markdown(document contract.Document, tree bool) ([]byte, error) {
	data, err := decode(document)
	if err != nil {
		return nil, err
	}
	if document.Operation == contract.ProjectFactsOperation {
		return []byte(projectFactsMarkdown(data)), nil
	}
	if document.Operation == "scan" {
		return []byte(viewFor(document.Stack).scanMarkdown(data, tree)), nil
	}
	return []byte(analyzeMarkdown(data, viewFor(document.Stack).analysis)), nil
}

// WriteFile atomically replaces only after rendering has succeeded, preserving an existing report on failure.
func WriteFile(path string, contents []byte) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".scryer-report-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if _, err = temporary.Write(contents); err != nil {
		temporary.Close()
		return err
	}
	if err = temporary.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
func counts(values Object) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := []string{}
	for _, key := range keys {
		parts = append(parts, key+": "+text(values[key]))
	}
	if len(parts) == 0 {
		return "unknown"
	}
	return strings.Join(parts, ", ")
}
func notes(values ...any) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		for _, item := range array(value) {
			t := text(item)
			if !seen[t] {
				seen[t] = true
				result = append(result, t)
			}
		}
	}
	return result
}
