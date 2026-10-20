package analyze

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceDoesNotRequireProjectRefsOrAnalyzer(t *testing.T) {
	dir := t.TempDir()
	p, d := filepath.Join(dir, "projects.csv"), filepath.Join(dir, "deps.csv")
	if err := os.WriteFile(p, []byte("projectName\na\nb\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(d, []byte("source,target\nb,a\n"), 0600); err != nil {
		t.Fatal(err)
	}
	command := New(nil)
	var out bytes.Buffer
	command.SetOut(&out)
	command.SetArgs([]string{"workspace", "--projects", p, "--dependencies", d})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "b -> a") {
		t.Fatal(out.String())
	}
}

func TestWorkspaceJSONFileAndStdout(t *testing.T) {
	dir := t.TempDir()
	p, d := filepath.Join(dir, "p.csv"), filepath.Join(dir, "d.csv")
	for path, text := range map[string]string{p: "projectId\na\n", d: "source,target\n"} {
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	command := New(nil)
	var out, progress bytes.Buffer
	command.SetOut(&out)
	command.SetErr(&progress)
	output := filepath.Join(dir, "report.json")
	command.SetArgs([]string{"workspace", "--projects", p, "--dependencies", d, "--json", "-o", output})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != out.String() || !strings.Contains(out.String(), `"schemaVersion": 1`) {
		t.Fatal("stdout and file must contain same JSON")
	}
}
