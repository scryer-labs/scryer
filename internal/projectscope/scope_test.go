package projectscope

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGlobSegmentsExclusionsAndEmptyInclude(t *testing.T) {
	for _, test := range []struct {
		pattern, file string
		want          bool
	}{{"**/*", "a.cpp", true}, {"**/*.cpp", "src/a.cpp", true}, {"src/**", "src/nested/a.cpp", true}, {"src/*", "src/nested/a.cpp", false}, {"**/test?.[ch]", "test1.c", true}} {
		if match(test.pattern, test.file) != test.want {
			t.Fatal(test)
		}
	}
	for _, pattern := range []string{"../x", "/x", "a/**b", "[bad", "a\\b"} {
		if validatePattern(pattern) == nil {
			t.Fatal(pattern)
		}
	}
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".scryer"), 0700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, ".scryer/project.yaml")
	if err := os.WriteFile(configPath, []byte("schemaVersion: 1\nscan:\n  include: []\n"), 0600); err != nil {
		t.Fatal(err)
	}
	config, err := ReadConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := Build(context.Background(), root, config)
	if err != nil {
		t.Fatal(err)
	}
	if inventory.Scope == nil || inventory.Scope.Included != 0 || len(inventory.Files) != 0 {
		t.Fatal(inventory)
	}
	if err := os.WriteFile(configPath, []byte("schemaVersion: 1\nscan:\n  excludes: []\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadConfig(root); err == nil {
		t.Fatal("accepted misspelled config key")
	}
}
