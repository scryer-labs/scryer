package engine

import (
	"archive/zip"
	"bytes"
	"context"
	"path/filepath"
	"testing"
)

func Test_should_reject_unsafe_release_archive_entries(t *testing.T) {
	for _, name := range []string{"../escaped", "inside/../../escaped", "/absolute", "bin/a:b", "bin\\escape", installRecord} {
		t.Run(name, func(t *testing.T) {
			var data bytes.Buffer
			writer := zip.NewWriter(&data)
			if _, err := writer.Create(name); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			archive, err := zip.NewReader(bytes.NewReader(data.Bytes()), int64(data.Len()))
			if err != nil {
				t.Fatal(err)
			}
			if err := extractPackage(context.Background(), archive, filepath.Join(t.TempDir(), "package")); err == nil {
				t.Fatalf("accepted unsafe archive entry %q", name)
			}
		})
	}
}

func Test_should_use_configured_release_source(t *testing.T) {
	store := Store{Home: t.TempDir()}
	sources := map[string]releaseSource{"cpp": {Repository: "example/cpp", Asset: "package-{version}-{os}-{arch}.zip"}}
	if err := writeJSON(filepath.Join(store.Home, "engine-sources.json"), sources); err != nil {
		t.Fatal(err)
	}
	source, err := store.releaseSource("cpp")
	if err != nil || source != sources["cpp"] {
		t.Fatalf("got %+v, %v", source, err)
	}
}
