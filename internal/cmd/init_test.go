package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/edwardnoaland/scryer/internal/application"
	"github.com/edwardnoaland/scryer/internal/configinit"
	"github.com/edwardnoaland/scryer/internal/projectscope"
	"gopkg.in/yaml.v2"
)

func TestScopeInitializationUsesValidTemplatesAndPreservesUserFiles(t *testing.T) {
	directory := t.TempDir()
	fake := &fakeAnalyzer{}
	for _, scope := range []string{"project", "workspace"} {
		var output bytes.Buffer
		command := NewRoot(&application.Service{Analyzer: fake}, &output, &output)
		command.SetArgs([]string{scope, "init", directory, "--json", "--compact"})
		if err := command.Execute(); err != nil {
			t.Fatal(err)
		}
		var result configinit.Result
		if err := json.Unmarshal(output.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Scope != configinit.Scope(scope) || result.Status != "created" || result.Operation != "init" {
			t.Fatal(result)
		}
		configPath := filepath.Join(directory, ".scryer", scope+".yaml")
		if scope == "project" {
			parsed, err := projectscope.ReadConfig(directory)
			if err != nil || parsed == nil {
				t.Fatal(parsed, err)
			}
			inventory, err := projectscope.Build(context.Background(), directory, parsed)
			if err != nil || inventory.Scope == nil {
				t.Fatal(inventory, err)
			}
		} else {
			data, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatal(err)
			}
			var parsed struct {
				Projects []string `yaml:"projects"`
			}
			if err = yaml.UnmarshalStrict(data, &parsed); err != nil || parsed.Projects == nil || len(parsed.Projects) != 0 {
				t.Fatal(parsed, err)
			}
		}
		// Even malformed user configuration is not replaced or interpreted by init.
		custom := []byte("user content that must remain unchanged\n")
		if err := os.WriteFile(configPath, custom, 0600); err != nil {
			t.Fatal(err)
		}
		output.Reset()
		command = NewRoot(&application.Service{Analyzer: fake}, &output, &output)
		command.SetArgs([]string{scope, "init", directory, "--json"})
		if err := command.Execute(); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(output.Bytes(), &result); err != nil || result.Status != "exists" {
			t.Fatal(result, err)
		}
		data, err := os.ReadFile(configPath)
		if err != nil || !bytes.Equal(data, custom) {
			t.Fatalf("configuration replaced: %s (%v)", data, err)
		}
	}
	if fake.calls != 0 {
		t.Fatal("initialization reached an engine")
	}
	entries, err := os.ReadDir(filepath.Join(directory, ".scryer"))
	if err != nil || len(entries) != 2 {
		t.Fatal(entries, err)
	}
}
