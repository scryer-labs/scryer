// Package projectfacts collects local repository facts without an engine.
package projectfacts

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"sync"

	"github.com/boyter/scc/v3/processor"
	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/edwardnoaland/scryer/internal/repository"
)

const maxFileSize = 8 * 1024 * 1024

var initCounter sync.Once

var buildTools = map[string]string{
	"go.mod":              "Go modules",
	"Cargo.toml":          "Cargo",
	"CMakeLists.txt":      "CMake",
	"pom.xml":             "Maven",
	"build.gradle":        "Gradle",
	"build.gradle.kts":    "Gradle",
	"settings.gradle":     "Gradle",
	"settings.gradle.kts": "Gradle",
	"package.json":        "Node package manifest",
	"pyproject.toml":      "Python project metadata",
	"Makefile":            "Make",
}

func Collect(ctx context.Context, directory string, stack, loc bool) (contract.ProjectFacts, error) {
	var result contract.ProjectFacts
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return result, err
	}
	absolute, err = filepath.EvalSymlinks(absolute)
	if err != nil {
		return result, err
	}
	root, err := os.OpenRoot(absolute)
	if err != nil {
		return result, err
	}
	defer root.Close()

	result.Root, result.SchemaVersion = absolute, contract.ReportVersion
	result.Name = filepath.Base(absolute)
	result.Repository = repository.Identity(ctx, absolute)
	if stack {
		result.Stack = &contract.ProjectStack{Languages: []contract.LanguagePresence{}, BuildTools: []contract.BuildToolEvidence{}}
	}
	if loc {
		result.LOC = &contract.ProjectLOC{Languages: []contract.LanguageLOC{}}
	}
	initCounter.Do(processor.ProcessConstants)
	totals := map[string]contract.LineCounts{}

	err = visitFiles(ctx, root, func(path string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := root.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		} // Tracked file deleted in the working tree.
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		filename := filepath.Base(path)
		if tool := buildTools[filename]; stack && tool != "" {
			result.Stack.BuildTools = append(result.Stack.BuildTools, contract.BuildToolEvidence{Tool: tool, Path: filepath.ToSlash(path)})
		}
		candidates, _ := processor.DetectLanguage(filename)
		if len(candidates) == 0 {
			result.Skipped.UnknownLanguage++
			return nil
		}
		if info.Size() > maxFileSize {
			result.Skipped.Oversized++
			return nil
		}
		content, err := readSource(root, path)
		if err != nil {
			return err
		}
		if len(content) > maxFileSize {
			result.Skipped.Oversized++
			return nil
		}
		if bytes.IndexByte(content, 0) >= 0 {
			result.Skipped.Binary++
			return nil
		}
		if len(candidates) > 1 {
			candidates = slices.Clone(candidates)
			slices.Sort(candidates)
		}
		language := processor.DetermineLanguage(filename, "", candidates, content)
		if language == processor.SheBang {
			language, _ = processor.DetectSheBang(string(content))
		}
		if language == "" {
			result.Skipped.UnknownLanguage++
			return nil
		}

		counts := totals[language]
		counts.Files++
		if loc {
			job := processor.FileJob{Filename: filename, Language: language, Content: content, Bytes: int64(len(content))}
			processor.CountStats(&job)
			counts.Lines += job.Lines
			counts.Code += job.Code
			counts.Comments += job.Comment
			counts.Blank += job.Blank
		}
		totals[language] = counts
		return nil
	})
	if err != nil {
		return result, fmt.Errorf("collect project facts: %w", err)
	}

	languages := make([]string, 0, len(totals))
	for name := range totals {
		languages = append(languages, name)
	}
	sort.Strings(languages)
	for _, name := range languages {
		counts := totals[name]
		if stack {
			result.Stack.Languages = append(result.Stack.Languages, contract.LanguagePresence{Language: name, Files: counts.Files})
		}
		if loc {
			result.LOC.Languages = append(result.LOC.Languages, contract.LanguageLOC{Language: name, LineCounts: counts})
			result.LOC.Total.Files += counts.Files
			result.LOC.Total.Lines += counts.Lines
			result.LOC.Total.Code += counts.Code
			result.LOC.Total.Comments += counts.Comments
			result.LOC.Total.Blank += counts.Blank
		}
	}
	if stack {
		sort.Slice(result.Stack.BuildTools, func(i, j int) bool { return result.Stack.BuildTools[i].Path < result.Stack.BuildTools[j].Path })
	}
	return result, nil
}

func readSource(root *os.Root, path string) ([]byte, error) {
	file, err := root.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(io.LimitReader(file, maxFileSize+1))
}
