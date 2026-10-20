// Package workspace collects cross-repository facts without invoking a build tool.
package workspace

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
)

type row struct {
	values map[string]string
	line   int
}

// ReadCSV accepts Moderne exports or source,target edge lists. Unknown columns
// are ignored; identifiers and dependency direction are never inferred from order.
func ReadCSV(projectsPath, dependenciesPath string) (Facts, error) {
	projects, err := readRows(projectsPath, false)
	if err != nil {
		return Facts{}, err
	}
	facts := Facts{Projects: []Project{}, Dependencies: []Dependency{}}
	aliases := map[string][]string{}
	known := map[string]bool{}
	for _, r := range projects {
		id := first(r.values, "projectId", "artifactId", "projectName")
		artifact, group := r.values["artifactId"], r.values["groupId"]
		if artifact != "" && group != "" {
			id = group + ":" + artifact
		}
		if id == "" {
			return Facts{}, rowError(projectsPath, r.line, "project identity must not be blank")
		}
		if known[id] {
			return Facts{}, rowError(projectsPath, r.line, "duplicate project identity: "+id)
		}
		known[id] = true
		facts.Projects = append(facts.Projects, Project{ID: id, RepositoryPath: r.values["repositoryPath"]})
		unique := map[string]bool{}
		for _, alias := range []string{id, artifact, r.values["projectId"], r.values["projectName"], r.values["repositoryPath"]} {
			if alias != "" && !unique[alias] {
				aliases[alias] = append(aliases[alias], id)
				unique[alias] = true
			}
		}
	}
	if len(facts.Projects) == 0 {
		return Facts{}, fmt.Errorf("%s: no projects found", projectsPath)
	}
	dependencies, err := readRows(dependenciesPath, true)
	if err != nil {
		return Facts{}, err
	}
	seen := map[[2]string]int{}
	for _, r := range dependencies {
		source := first(r.values, "source", "projectName", "projectId", "repositoryPath")
		target := first(r.values, "target")
		if target == "" {
			target = r.values["artifactId"]
			if target != "" && r.values["groupId"] != "" {
				target = r.values["groupId"] + ":" + target
			}
		}
		if source == "" || target == "" {
			return Facts{}, rowError(dependenciesPath, r.line, "dependency source and target must not be blank")
		}
		resolve := func(value string) (string, error) {
			matches := aliases[value]
			if len(matches) > 1 {
				return "", rowError(dependenciesPath, r.line, "ambiguous project reference: "+value)
			}
			if len(matches) == 1 {
				return matches[0], nil
			}
			return value, nil
		}
		source, err = resolve(source)
		if err != nil {
			return Facts{}, err
		}
		target, err = resolve(target)
		if err != nil {
			return Facts{}, err
		}
		if !known[source] {
			return Facts{}, rowError(dependenciesPath, r.line, "dependency source is not in projects: "+source)
		}
		key := [2]string{source, target}
		index, exists := seen[key]
		if !exists {
			index = len(facts.Dependencies)
			facts.Dependencies = append(facts.Dependencies, Dependency{Source: source, Target: target, Internal: known[target], Scopes: []string{}})
			seen[key] = index
		}
		scope := r.values["scope"]
		edge := &facts.Dependencies[index]
		if !slices.Contains(edge.Scopes, scope) {
			edge.Scopes = append(edge.Scopes, scope)
		}

	}
	return facts, nil
}

func first(values map[string]string, names ...string) string {
	for _, name := range names {
		if value := values[name]; value != "" {
			return value
		}
	}
	return ""
}
func rowError(path string, line int, message string) error {
	return fmt.Errorf("%s:%d: %s", path, line, message)
}

func readRows(path string, dependencies bool) ([]row, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	input := bufio.NewReader(file)
	if prefix, _ := input.Peek(3); string(prefix) == "\xef\xbb\xbf" {
		_, _ = input.Discard(3)
	}
	reader := csv.NewReader(input)
	reader.Comment = '#'
	reader.FieldsPerRecord = -1
	var header []string
	var rows []row
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		line, _ := reader.FieldPos(0)
		if header == nil {
			header = record
			seen := map[string]bool{}
			for i, value := range header {
				header[i] = strings.TrimSpace(strings.TrimPrefix(value, "\ufeff"))
				if header[i] == "" || seen[header[i]] {
					return nil, rowError(path, line, "blank or duplicate column name")
				}
				seen[header[i]] = true
			}
			projectHeader := seen["artifactId"] || seen["projectId"] || seen["projectName"]
			if dependencies {
				sourceHeader := seen["source"] || seen["projectName"] || seen["projectId"] || seen["repositoryPath"]
				targetHeader := seen["target"] || seen["artifactId"]
				if !sourceHeader || !targetHeader {
					return nil, rowError(path, line, "expected source,target or Moderne consumer and artifact columns")
				}
			} else if !projectHeader {
				return nil, rowError(path, line, "expected artifactId, projectId or projectName")
			}
			continue
		}
		if len(record) != len(header) {
			return nil, rowError(path, line, "column count does not match header")
		}
		values := map[string]string{}
		for i, value := range record {
			values[header[i]] = strings.TrimSpace(value)
		}
		rows = append(rows, row{values, line})
	}
	if header == nil {
		return nil, fmt.Errorf("%s: missing CSV header", path)
	}
	return rows, nil
}
