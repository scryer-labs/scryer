package report

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

func escape(value string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\\", "\\\\", "|", "\\|", "`", "\\`", "*", "\\*", "_", "\\_", "[", "\\[", "]", "\\]", "\r", "", "\n", "<br>").Replace(value)
}
func section(b *strings.Builder, level int, title string) {
	fmt.Fprintf(b, "%s %s\n\n", strings.Repeat("#", level), escape(title))
}
func table(b *strings.Builder, headers []string, rows [][]string) {
	write := func(row []string) {
		values := []string{}
		for _, v := range row {
			values = append(values, escape(v))
		}
		fmt.Fprintln(b, "| "+strings.Join(values, " | ")+" |")
	}
	write(headers)
	dividers := make([]string, len(headers))
	for i := range dividers {
		dividers[i] = "---"
	}
	write(dividers)
	for _, row := range rows {
		write(row)
	}
	b.WriteString("\n")
}
func block(b *strings.Builder, language, source string) {
	longest, current := 0, 0
	for _, c := range source {
		if c == '`' {
			current++
			if current > longest {
				longest = current
			}
		} else {
			current = 0
		}
	}
	length := longest + 1
	if length < 3 {
		length = 3
	}
	fence := strings.Repeat("`", length)
	fmt.Fprintf(b, "%s%s\n%s\n%s\n\n", fence, language, strings.TrimRight(source, "\n"), fence)
}
func compact(signature string) string {
	owner, rest, ok := strings.Cut(signature, "#")
	if !ok {
		return signature
	}
	parts := strings.Split(owner, ".")
	return parts[len(parts)-1] + "#" + rest
}
func labels(nodes []Object) map[string]string {
	groups := map[string]map[string]bool{}
	for _, n := range nodes {
		signature := str(n, "signature")
		label := compact(signature)
		if groups[label] == nil {
			groups[label] = map[string]bool{}
		}
		groups[label][signature] = true
	}
	result := map[string]string{}
	for label, group := range groups {
		for signature := range group {
			if len(group) > 1 {
				result[signature] = signature
			} else {
				result[signature] = label
			}
		}
	}
	return result
}
func jsonText(value any) string { data, _ := json.MarshalIndent(value, "", "  "); return string(data) }
func scanMarkdown(data Object, tree bool) string {
	var b strings.Builder
	section(&b, 1, "Scryer repository scan")
	b.WriteString("Repository facts and collection limitations; not build/test execution evidence.\n\n")
	section(&b, 2, "Project")
	language := field(data, "language")
	builds := []string{}
	for _, build := range objects(data["builds"]) {
		builds = append(builds, str(build, "tool")+" "+str(build, "version"))
	}
	table(&b, []string{"Fact", "Value"}, rows([]string{"Repository", str(data, "root")}, []string{"Build", strings.Join(builds, ", ")}, []string{"Selected build tool", selectedBuildTool(data)}, []string{"Java source", join(language["sourceVersions"])}, []string{"Java target", join(language["targetVersions"])}, []string{"Toolchain", join(language["toolchainVersions"])}, []string{"Frameworks", join(data["frameworks"])}, []string{"Resolution", str(field(data, "resolution"), "status")}))
	section(&b, 2, "Modules")
	moduleRows := [][]string{}
	for _, m := range objects(data["modules"]) {
		moduleRows = append(moduleRows, []string{str(m, "id"), str(m, "directory"), str(m, "definition")})
	}
	table(&b, []string{"Module", "Directory", "Definition"}, moduleRows)
	section(&b, 2, "Dependencies")
	declarations := directDeclarations(data)
	direct := map[string]bool{}
	for _, dependency := range declarations {
		direct[str(dependency, "module")+"|"+str(dependency, "notation")] = true
	}
	resolution := field(data, "resolution")
	table(&b, []string{"Fact", "Value"}, rows(
		[]string{"Direct (observed)", fmt.Sprint(len(direct))},
		[]string{"Resolved components", str(resolution, "resolvedComponentCount")},
		[]string{"Conflict selections", str(resolution, "conflictCount")},
		[]string{"Forced overrides", str(resolution, "overrideCount")},
	))
	dependencyRows := [][]string{}
	for _, d := range directDeclarations(data) {
		dependencyRows = append(dependencyRows, []string{str(d, "module"), str(d, "configuration"), str(d, "notation"), str(d, "declaredVersion"), selectedVersions(data, d), str(d, "source")})
	}
	table(&b, []string{"Module", "Configuration", "Dependency", "Declared version", "Selected version", "Source"}, dependencyRows)
	b.WriteString("Declared and selected versions are distinct observations. Missing values remain unknown.\n\n")
	if remote := field(data, "remoteVersions"); remote != nil {
		section(&b, 2, "Remote dependency releases")
		fmt.Fprintln(&b, escape(str(remote, "source"))+"\n\nChecked: "+escape(str(remote, "checkedAt"))+". Metadata releases may include prereleases; this is not compatibility advice.\n")
		order, groups := group(objects(remote["dependencies"]))
		for _, key := range order {
			section(&b, 3, key)
			rows := [][]string{}
			for _, d := range groups[key] {
				rows = append(rows, []string{str(d, "coordinate"), str(d, "current"), str(d, "currentSource"), str(d, "latest"), str(d, "status"), str(d, "metadataUrl"), str(d, "note")})
			}
			table(&b, []string{"Dependency", "Current", "Current source", "Remote release", "Comparison", "Metadata", "Limitation"}, rows)
		}
	}
	section(&b, 2, "Testing")
	testing := field(data, "testing")
	table(&b, []string{"Fact", "Value"}, rows([]string{"Frameworks", join(testing["frameworks"])}, []string{"Mock libraries", join(testing["mockLibraries"])}, []string{"Unit source files", str(testing, "testSourceFiles")}, []string{"Integration source files", str(testing, "integrationSourceFiles")}, []string{"JaCoCo", str(testing, "jacocoConfigured")}, []string{"Disabled signals", str(testing, "disabledAnnotationSignals")}, []string{"Existing coverage reports", join(testing["coverageReports"])}))
	b.WriteString("Test counts classify source files; they do not prove execution coverage.\n\n")
	section(&b, 2, "Source layout")
	sources := field(data, "sources")
	table(&b, []string{"Kind", "Roots"}, rows([]string{"Production", join(sources["productionRoots"])}, []string{"Tests", join(sources["testRoots"])}, []string{"Integration", join(sources["integrationRoots"])}, []string{"Generated", join(sources["generatedRoots"])}))
	section(&b, 2, "Code characteristics")
	signalRows := [][]string{}
	for _, signal := range objects(data["signals"]) {
		signalRows = append(signalRows, []string{str(signal, "name"), join(signal["locations"]), str(signal, "provenance")})
	}
	table(&b, []string{"Signal", "Locations", "Provenance"}, signalRows)
	section(&b, 2, "Verification (suggested commands; may be incomplete or inaccurate)")
	verification := field(data, "verification")
	table(&b, []string{"Kind", "Suggestion / observation"}, rows([]string{"Build", join(verification["buildCommands"])}, []string{"Test", join(verification["testCommands"])}, []string{"Package", join(verification["packageCommands"])}, []string{"CI configs", join(verification["ciFiles"])}))
	b.WriteString("Commands follow build conventions; scan does not verify build/test success.\n\n")
	section(&b, 2, "Collection notes and failures")
	for _, note := range notes(data["notes"], field(data, "resolution")["notes"]) {
		fmt.Fprintln(&b, "- "+escape(note))
	}
	b.WriteString("\n")
	for _, graph := range objects(field(data, "resolution")["configurations"]) {
		if graph["error"] != nil {
			fmt.Fprintln(&b, "- "+escape(str(graph, "module")+" / "+str(graph, "configuration")+": "+str(graph, "error")))
		}
	}
	if tree {
		section(&b, 2, "Resolved dependency tree")
		var output strings.Builder
		dependencyTree(data, &output)
		block(&b, "text", output.String())
	}
	return b.String()
}
func mermaid(snapshot Object) string {
	var b strings.Builder
	b.WriteString("flowchart LR\n")
	nodes := objects(snapshot["nodes"])
	names := labels(nodes)
	ids := map[string]string{}
	for i, n := range nodes {
		ids[str(n, "id")] = fmt.Sprintf("n%d", i)
	}
	encode := func(value string) string {
		var output strings.Builder
		for _, c := range value {
			if unicode.IsLetter(c) || unicode.IsDigit(c) || strings.ContainsRune(" .,_$()-", c) {
				output.WriteRune(c)
			} else {
				fmt.Fprintf(&output, "#%d;", c)
			}
		}
		return output.String()
	}
	for _, n := range nodes {
		id := ids[str(n, "id")]
		label := names[str(n, "signature")] + " [" + str(n, "impact") + "] [" + str(n, "role") + "]"
		fmt.Fprintf(&b, "  %s[\"%s\"]\n", id, encode(label))
		style := "indirect"
		switch str(n, "impact") {
		case "CHANGED":
			style = "changed"
		case "CALLER":
			style = "caller"
		}
		if str(n, "role") == "TEST" {
			style = "test"
		}
		fmt.Fprintf(&b, "  class %s %s\n", id, style)
	}
	for _, edge := range objects(snapshot["edges"]) {
		fmt.Fprintf(&b, "  %s -->|%s| %s\n", ids[str(edge, "caller")], encode(str(edge, "kind")), ids[str(edge, "callee")])
	}
	b.WriteString("  classDef changed fill:#ffe3e5,stroke:#d73952,color:#541325\n  classDef caller fill:#dfedff,stroke:#3978cf,color:#143359\n  classDef indirect fill:#fff0ce,stroke:#bf811a,color:#523805\n  classDef test fill:#eee4ff,stroke:#8653c5,color:#3b1766\n")
	return b.String()
}

func selectedBuildTool(data Object) string {
	if data["selectedBuildTool"] == nil {
		return "automatic"
	}
	return str(data, "selectedBuildTool")
}
