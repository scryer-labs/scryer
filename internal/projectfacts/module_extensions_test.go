package projectfacts

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestModuleExtensionsRemainVisibleToStackDispatchAndLOC(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"one.cjs", "two.mjs", "three.mts", "four.cts"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("const value = 1;\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	facts, err := Collect(context.Background(), root, true, true)
	if err != nil {
		t.Fatal(err)
	}
	languages := map[string]uint64{}
	for _, language := range facts.Stack.Languages {
		languages[language.Language] = uint64(language.Files)
	}
	if languages["JavaScript"] != 2 || languages["TypeScript"] != 2 || facts.Skipped.UnknownLanguage != 0 {
		t.Fatalf("module files were omitted from dispatch: %#v", facts)
	}
}
