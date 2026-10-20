package report

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/edwardnoaland/scryer/internal/workspace"
)

// Workspace renders the same normalized document for tools and humans.
func Workspace(document workspace.Document, format string) ([]byte, error) {
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, err
	}
	switch format {
	case "json":
		return append(data, '\n'), nil
	case "html":
		template, err := assets.ReadFile("assets/workspace.html")
		if err != nil {
			return nil, err
		}
		script, err := assets.ReadFile("assets/workspace.js")
		if err != nil {
			return nil, err
		}
		logo, err := assets.ReadFile("assets/scryer-logo.png")
		if err != nil {
			return nil, err
		}
		html := strings.ReplaceAll(string(template), "__SCRIPT__", string(script))
		html = strings.ReplaceAll(html, "__LOGO__", base64.StdEncoding.EncodeToString(logo))
		// encoding/json escapes HTML-sensitive characters; insert payload last.
		return []byte(strings.ReplaceAll(html, "__DATA__", string(data))), nil
	default:
		return nil, fmt.Errorf("unsupported workspace format %q", format)
	}
}

func WorkspaceFormat(output string) (string, error) {
	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(output)), ".")
	if format != "json" && format != "html" {
		return "", fmt.Errorf("workspace -o must use .json or .html")
	}
	return format, nil
}

func SaveWorkspace(document workspace.Document, output string, progress io.Writer) error {
	format, err := WorkspaceFormat(output)
	if err != nil {
		return err
	}
	data, err := Workspace(document, format)
	if err != nil {
		return err
	}
	absolute, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	if err = WriteFile(absolute, data); err != nil {
		return err
	}
	_, err = fmt.Fprintln(progress, "scryer: Report saved to "+absolute)
	return err
}
