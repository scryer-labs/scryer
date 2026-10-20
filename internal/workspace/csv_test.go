package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func load(t *testing.T, projects, dependencies string) (Facts, error) {
	t.Helper()
	dir := t.TempDir()
	p, d := filepath.Join(dir, "projects.csv"), filepath.Join(dir, "dependencies.csv")
	if err := os.WriteFile(p, []byte(projects), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(d, []byte(dependencies), 0600); err != nil {
		t.Fatal(err)
	}
	return ReadCSV(p, d)
}
func TestModerneCoordinatesAndExternalDependencies(t *testing.T) {
	facts, err := load(t, "# @name projects\ngroupId,artifactId,repositoryPath,extra\ng,a,org/a,ignored\ng,b,org/b,ignored\n", "# @name dependencies\nprojectName,groupId,artifactId,scope\ng:b,g,a,compile\ng:b,g,a,test\ng:b,external,lib,compile\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(facts.Projects) != 2 || len(facts.Dependencies) != 2 {
		t.Fatalf("unexpected facts: %+v", facts)
	}
	if edge := facts.Dependencies[0]; edge.Source != "g:b" || edge.Target != "g:a" || !edge.Internal {
		t.Fatalf("unexpected edge: %+v", edge)
	}
	if facts.Dependencies[1].Internal {
		t.Fatal("external target marked internal")
	}
}
func TestSimpleCSVAndQuotedNames(t *testing.T) {
	facts, err := load(t, "projectId,extra\n\"a,one\",x\nb,x\n", "source,target\nb,\"a,one\"\n")
	if err != nil {
		t.Fatal(err)
	}
	if facts.Dependencies[0].Target != "a,one" {
		t.Fatal(facts)
	}
}
func TestInvalidCSV(t *testing.T) {
	cases := []struct{ name, projects, deps, want string }{
		{"duplicate", "projectId\na\na\n", "source,target\n", "duplicate project"},
		{"ambiguous", "groupId,artifactId\nx,a\ny,a\n", "source,target\nx:a,a\n", "ambiguous"},
		{"unknown source", "projectId\na\n", "source,target\nb,a\n", "source is not"},
		{"missing target", "projectId\na\n", "source,target\na,\n", "must not be blank"},
		{"bad row", "projectId\na\n", "source,target\na\n", "column count"},
		{"no header", "", "source,target\n", "missing CSV header"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := load(t, c.projects, c.deps)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("expected %q, got %v", c.want, err)
			}
		})
	}
}
