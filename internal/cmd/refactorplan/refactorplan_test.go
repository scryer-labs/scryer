package refactorplan

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateOutputsWithoutAnalyzerOrRunner(t *testing.T) {
	dir := t.TempDir()
	projects, dependencies := filepath.Join(dir, "p.csv"), filepath.Join(dir, "d.csv")
	for path, value := range map[string]string{projects: "projectId\na\nb\n", dependencies: "source,target\nb,a\n"} {
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, extension := range []string{"json", "html", "yaml"} {
		t.Run(extension, func(t *testing.T) {
			command := New()
			var out, progress bytes.Buffer
			command.SetOut(&out)
			command.SetErr(&progress)
			output := filepath.Join(dir, "plan."+extension)
			command.SetArgs([]string{"generate", "--projects", projects, "--dependencies", dependencies, "-o", output})
			if err := command.Execute(); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			if len(data) == 0 || out.Len() != 0 {
				t.Fatal("expected file-only output")
			}
			if extension == "yaml" && !strings.Contains(string(data), "type: mod-cli") {
				t.Fatal("missing default provider")
			}
		})
	}
}
