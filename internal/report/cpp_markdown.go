package report

import (
	"fmt"
	"strings"
)

func cppComponentNames(outcomes []Object) map[string]string {
	names := map[string]string{}
	for _, outcome := range outcomes {
		if factEngine(outcome) != "cpp" || str(outcome, "fact") != "architecture" {
			continue
		}
		for _, component := range objects(field(outcome, "data")["components"]) {
			names[str(component, "componentId")] = str(component, "name")
		}
	}
	return names
}

func cppMarkdown(b *strings.Builder, fact string, data Object, names map[string]string) bool {
	switch fact {
	case "architecture":
		cppArchitectureMarkdown(b, data)
	case "files":
		cppFilesMarkdown(b, data, names)
	case "call-graph":
		cppCallsMarkdown(b, data)
	case "dependencies":
		cppArchitectureMarkdown(b, field(data, "cmake"))
		cppIncludesMarkdown(b, objects(data["includes"]))
	default:
		return false
	}
	return true
}

func cppArchitectureMarkdown(b *strings.Builder, data Object) {
	if data == nil {
		return
	}
	targets := objects(data["components"])
	if _, ok := data["targets"]; ok {
		targets = objects(data["targets"])
	}
	section(b, 4, "CMake components")
	var entries [][]string
	for _, item := range targets {
		entries = append(entries, []string{str(item, "name"), str(item, "kind"), optional(item, "type"), optional(item, "aliasOf"), cppLocation(item)})
	}
	table(b, []string{"Component", "Kind", "Type", "Alias of", "Declaration"}, entries)
	section(b, 4, "Link declarations")
	entries = nil
	for _, item := range objects(data["links"]) {
		entries = append(entries, []string{str(item, "from"), str(item, "to"), optional(item, "visibility"), cppLocation(item)})
	}
	table(b, []string{"From", "To", "Visibility", "Declaration"}, entries)
	if dependencies := objects(data["externalDependencies"]); len(dependencies) > 0 {
		section(b, 4, "External declarations")
		entries = nil
		for _, item := range dependencies {
			entries = append(entries, []string{str(item, "name"), str(item, "command"), join(item["arguments"]), cppLocation(item)})
		}
		table(b, []string{"Name", "Command", "Arguments", "Declaration"}, entries)
	}
}

func cppFilesMarkdown(b *strings.Builder, data Object, names map[string]string) {
	files := objects(data["files"])
	refs := objects(data["references"])
	matched := 0
	for _, ref := range refs {
		if ref["to"] != nil {
			matched++
		}
	}
	fmt.Fprintf(b, "%d files · %d include references · %d without a unique match\n\n", len(files), len(refs), len(refs)-matched)
	groups := map[string][][]string{}
	var order []string
	for _, file := range files {
		ids := array(file["componentIds"])
		if len(ids) == 0 {
			ids = []any{"Without a declared component"}
		}
		for _, id := range ids {
			key := text(id)
			if _, exists := groups[key]; !exists {
				order = append(order, key)
			}
			var contexts []string
			for _, context := range array(file["searchComponentIds"]) {
				contexts = append(contexts, cppComponentLabel(text(context), names))
			}
			groups[key] = append(groups[key], []string{str(file, "path"), strings.Join(contexts, ", ")})
		}
	}
	for _, key := range order {
		section(b, 4, cppComponentLabel(key, names))
		table(b, []string{"File", "Include search contexts"}, groups[key])
	}
	cppIncludesMarkdown(b, refs)
}

func cppIncludesMarkdown(b *strings.Builder, refs []Object) {
	section(b, 4, "Include references")
	var matched, unmatched [][]string
	for _, ref := range refs {
		var evidence []string
		for _, candidate := range objects(ref["candidates"]) {
			for _, dir := range objects(candidate["searchPaths"]) {
				evidence = append(evidence, str(candidate, "path")+" via "+cppLocation(dir))
			}
		}
		row := []string{cppLocation(ref), str(ref, "target"), optional(ref, "to"), optional(ref, "resolution"), strings.Join(evidence, ", ")}
		if ref["to"] == nil {
			unmatched = append(unmatched, row)
		} else {
			matched = append(matched, row)
		}
	}
	headers := []string{"Source", "Include", "Matched file", "Resolution", "Search evidence"}
	if len(unmatched) > 0 {
		section(b, 5, "Without a unique match")
		table(b, headers, unmatched)
	}
	if len(matched) > 0 {
		section(b, 5, "Matched")
		table(b, headers, matched)
	}
}

func cppCallsMarkdown(b *strings.Builder, data Object) {
	symbols := objects(data["symbols"])
	calls := objects(data["calls"])
	fmt.Fprintf(b, "%d definitions · %d syntactic call sites. Name candidates are navigation hints.\n\n", len(symbols), len(calls))
	names := map[string]string{}
	var definitions, sites [][]string
	for _, symbol := range symbols {
		name := optional(symbol, "name")
		if name == "" {
			name = str(symbol, "kind")
		}
		names[str(symbol, "symbolId")] = name
		definitions = append(definitions, []string{name, cppRangeLocation(field(symbol, "definition"))})
	}
	for _, call := range calls {
		sites = append(sites, []string{names[optional(call, "callerSymbolId")], str(call, "callee"), names[optional(call, "candidateSymbolId")], str(call, "resolution"), cppRangeLocation(field(call, "site"))})
	}
	table(b, []string{"Symbol", "Definition"}, definitions)
	table(b, []string{"Caller", "Callee expression", "Candidate", "Resolution", "Call site"}, sites)
}

func cppLocation(data Object) string {
	return str(data, "file") + ":" + str(data, "line")
}

func cppRangeLocation(data Object) string {
	start := field(data, "start")
	return str(data, "file") + ":" + str(start, "line") + ":" + str(start, "column")
}

func cppComponentLabel(id string, names map[string]string) string {
	if name := names[id]; name != "" {
		return name
	}
	return id
}

func optional(data Object, key string) string {
	if value := data[key]; value != nil {
		return text(value)
	}
	return ""
}
