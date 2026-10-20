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
