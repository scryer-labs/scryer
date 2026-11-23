package configinit

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func TestConcurrentInitializationCreatesOnlyOneFile(t *testing.T) {
	directory := t.TempDir()
	var lock sync.Mutex
	created, existing := 0, 0
	var group sync.WaitGroup
	for i := 0; i < 6; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			result, err := Initialize(context.Background(), directory, Project)
			if err != nil {
				t.Error(err)
				return
			}
			lock.Lock()
			defer lock.Unlock()
			if result.Status == "created" {
				created++
			} else if result.Status == "exists" {
				existing++
			}
		}()
	}
	group.Wait()
	if created != 1 || existing != 5 {
		t.Fatal(created, existing)
	}
	content, err := os.ReadFile(filepath.Join(directory, ".scryer/project.yaml"))
	if err != nil || string(content) != projectTemplate {
		t.Fatal("incomplete configuration", err)
	}
}

func TestInitializationCannotWriteThroughAnExternalConfigDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires Windows privileges")
	}
	directory, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(directory, ".scryer")); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(context.Background(), directory, Project); err == nil {
		t.Fatal("wrote outside requested root")
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatal(entries, err)
	}
}
