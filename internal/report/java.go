package report

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

var javaView = engineView{
	scanTerminal: javaScanTerminal,
	scanMarkdown: javaScanMarkdown,
	analysis: analyzeView{
		changedSymbols:   "Changed Java methods",
		sourceLanguage:   "java",
		runtimeField:     "javaHome",
		runtimeLabel:     "JAVA_HOME",
		htmlRuntimeLabel: "Target JAVA_HOME",
	},
}

// The Java scan view reads the existing Java v1 fact shape.
func javaScanTerminal(data Object, out io.Writer, dependencies, tree, color bool) {
	style := terminalStyle{color}
	header := func(title string) { fmt.Fprintln(out, style.heading(title)) }
	row := func(key string, value any) {
		fmt.Fprintf(out, "  %s%s\n", style.muted(fmt.Sprintf("%-26s", key)), style.fact(text(value)))
	}
	header("Project")
	row("Repository", data["root"])
	language := field(data, "language")
	row("Java source / target", join(language["sourceVersions"])+" / "+join(language["targetVersions"]))
	row("Toolchain", join(language["toolchainVersions"]))
	builds := []string{}
	wrappers := []string{}
	for _, build := range objects(data["builds"]) {
		builds = append(builds, str(build, "tool")+" "+style.fact(str(build, "version")))
		status := "not found"
		if boolean(build["wrapperPresent"]) {
			status = "present"
		}
		wrappers = append(wrappers, str(build, "tool")+": "+style.status(status))
	}
	row("Build", strings.Join(builds, ", "))
	if data["selectedBuildTool"] != nil {
		row("Selected build tool", str(data, "selectedBuildTool")+" (explicit)")
	}
	row("Wrapper", strings.Join(wrappers, ", "))
	modules := map[string]bool{}
	for _, m := range objects(data["modules"]) {
		modules[str(m, "id")] = true
	}
	row("Modules", len(modules))
	resolution := field(data, "resolution")
	declared := directDeclarations(data)
	unique := map[string]bool{}
	for _, d := range declared {
		unique[str(d, "module")+"|"+str(d, "notation")] = true
	}
	header("\nDependencies")
	row("Direct (observed)", len(unique))
	row("Resolved components", resolution["resolvedComponentCount"])
	directIDs := map[string]bool{}
	componentIDs := map[string]bool{}
	for _, graph := range objects(resolution["configurations"]) {
		for _, edge := range objects(graph["edges"]) {
			if boolean(edge["direct"]) && edge["to"] != nil {
				directIDs[str(edge, "to")] = true
			}
		}
		for _, node := range objects(graph["nodes"]) {
			if node["group"] != nil {
				componentIDs[str(node, "id")] = true
			}
		}
	}
	var transitive any
	if resolution["resolvedComponentCount"] != nil {
		count := 0
		for id := range componentIDs {
			if !directIDs[id] {
				count++
			}
		}
		transitive = count
	}
	row("Transitive-only", transitive)
	row("Conflict selections", resolution["conflictCount"])
	row("Forced overrides", resolution["overrideCount"])
	row("Resolution", resolution["status"])
	header("\nKey dependencies")
	shown := map[string]bool{}
	count := 0
	for _, d := range declared {
		coordinate := str(d, "notation")
		if shown[coordinate] || !strings.Contains(coordinate, "spring-boot") {
			continue
		}
		shown[coordinate] = true
		current := selectedVersions(data, d)
		if current == "unavailable" {
			current = str(d, "version")
		}
		row(coordinate, current)
		count++
		if count == 5 {
			break
		}
	}
	testing := field(data, "testing")
	header("\nTesting")
	row("Framework", join(testing["frameworks"]))
	row("Mockito", join(testing["mockLibraries"]))
	row("Unit source files", testing["testSourceFiles"])
	row("Integration files", testing["integrationSourceFiles"])
	jacoco := "not detected"
	if boolean(testing["jacocoConfigured"]) {
		jacoco = "configured"
	}
	row("JaCoCo", jacoco)
	row("Disabled signals", testing["disabledAnnotationSignals"])
	fmt.Fprintln(out, "  Test counts classify source files, not execution evidence.")
	header("\nCode Characteristics")
	for _, signal := range objects(data["signals"]) {
		state := "not detected"
		if len(array(signal["locations"])) > 0 {
			state = "detected"
		}
		row(str(signal, "name"), state)
	}
	tooling := field(data, "compileTooling")
	processors := "unknown"
	if tooling["annotationProcessors"] != nil {
		processors = fmt.Sprintf("%d observed", len(array(tooling["annotationProcessors"])))
	}
	row("Annotation processors", processors)
	generated := "unknown"
	if roots := field(data, "sources")["generatedRoots"]; roots != nil {
		generated = fmt.Sprintf("%d existing", len(array(roots)))
	}
	row("Generated roots", generated)
	verification := field(data, "verification")
	header("\nVerification (suggested commands; may be incomplete or inaccurate)")
	row("Build", join(verification["buildCommands"]))
	row("Test", join(verification["testCommands"]))
	row("CI configs", len(array(verification["ciFiles"])))
	fmt.Fprintln(out, "  Commands inferred from build conventions; scan does not verify build/test success.")
	header("\nCollection notes")
	for _, note := range notes(data["notes"], resolution["notes"]) {
		fmt.Fprintln(out, style.muted("  "+note))
	}
	if remote := field(data, "remoteVersions"); remote != nil {
		header("\nRemote dependency releases")
		fmt.Fprintln(out, "  "+str(remote, "source"))
		fmt.Fprintln(out, "  Checked: "+str(remote, "checkedAt"))
		order, groups := group(objects(remote["dependencies"]))
		for _, key := range order {
			fmt.Fprintln(out, style.heading("\n  "+key))
			for _, d := range groups[key] {
				comparison := str(d, "current") + " → " + str(d, "latest") + " [" + str(d, "status") + "]"
				comparison = style.paint(remoteColor(str(d, "status")), comparison)
				fmt.Fprintf(out, "    %s (%s): %s\n", str(d, "coordinate"), str(d, "currentSource"), comparison)
				if d["note"] != nil {
					fmt.Fprintln(out, "      "+str(d, "note"))
				}
			}
		}
	}
	if updates := field(data, "toolUpdates"); updates != nil {
		header("\nToolchain / Build tool updates")
		fmt.Fprintln(out, "  Checked: "+str(updates, "checkedAt"))
		for _, tool := range objects(updates["tools"]) {
			comparison := str(tool, "current") + " → " + str(tool, "latest") + " [" + str(tool, "status") + "]"
			fmt.Fprintf(out, "  %s / %s: %s\n", str(tool, "tool"), str(tool, "channel"), style.paint(remoteColor(str(tool, "status")), comparison))
			fmt.Fprintln(out, style.muted("    Current source: "+str(tool, "currentSource")))
			if tool["note"] != nil {
				fmt.Fprintln(out, style.warning("    "+str(tool, "note")))
			}
		}
		for _, note := range notes(updates["notes"]) {
			fmt.Fprintln(out, style.warning("  "+note))
		}
	}

	if dependencies {
		header("\nDirect dependencies")
		order, groups := group(declared)
		for _, key := range order {
			fmt.Fprintln(out, style.heading("  "+key))
			for _, d := range groups[key] {
				fmt.Fprintf(out, "    %s:%s; selected %s; %s\n", str(d, "notation"), str(d, "version"), selectedVersions(data, d), str(d, "source"))
			}
		}
	}
	if tree {
		dependencyTreeStyled(data, out, color)
	}
}

func dependencyTree(data Object, out io.Writer) { dependencyTreeStyled(data, out, false) }
func dependencyTreeStyled(data Object, out io.Writer, color bool) {
	style := terminalStyle{color}
	fmt.Fprintln(out, style.heading("\nResolved dependency tree (selected graph; shared/cyclic nodes expanded once)"))
	for _, graph := range objects(field(data, "resolution")["configurations"]) {
		fmt.Fprintf(out, "  %s / %s [%s]\n", str(graph, "module"), str(graph, "configuration"), style.status(str(graph, "status")))
		if graph["error"] != nil {
			fmt.Fprintln(out, style.paint("31", "    "+str(graph, "error")))
		}
		nodes := map[string]Object{}
		edges := map[string][]Object{}
		for _, n := range objects(graph["nodes"]) {
			nodes[str(n, "id")] = n
		}
		for _, edge := range objects(graph["edges"]) {
			edges[str(edge, "from")] = append(edges[str(edge, "from")], edge)
		}
		type item struct {
			id    string
			depth int
		}
		queue := []item{{str(graph, "root"), 0}}
		visited := map[string]bool{}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			if visited[current.id] {
				continue
			}
			visited[current.id] = true
			for _, edge := range edges[current.id] {
				id := str(edge, "to")
				n := nodes[id]
				label := str(edge, "requested")
				if n != nil {
					label = str(n, "group") + ":" + str(n, "artifact") + ":" + str(n, "version")
				}
				if edge["unresolved"] != nil {
					label += style.paint("31", " UNRESOLVED "+str(edge, "unresolved"))
				}
				if visited[id] {
					label += " (shared/cycle)"
				}
				fmt.Fprintln(out, "    "+strings.Repeat("  ", current.depth)+"→ "+label)
				if !visited[id] {
					queue = append(queue, item{id, current.depth + 1})
				}
			}
		}
	}
}

func javaScanMarkdown(data Object, tree bool) string {
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
	if updates := field(data, "toolUpdates"); updates != nil {
		section(&b, 2, "Toolchain / Build tool updates")
		fmt.Fprintln(&b, "Checked: "+escape(str(updates, "checkedAt"))+".\n")
		toolRows := [][]string{}
		for _, tool := range objects(updates["tools"]) {
			toolRows = append(toolRows, []string{str(tool, "tool"), str(tool, "channel"), str(tool, "current"), str(tool, "currentSource"), str(tool, "latest"), str(tool, "status"), str(tool, "metadataUrl"), toolUpdateNote(tool)})
		}
		table(&b, []string{"Tool", "Release channel", "Current", "Current source", "Available release", "Comparison", "Metadata", "Limitation"}, toolRows)
		for _, note := range notes(updates["notes"]) {
			fmt.Fprintln(&b, escape(note)+"\n")
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

func group(dependencies []Object) ([]string, map[string][]Object) {
	order := []string{}
	groups := map[string][]Object{}
	for _, dependency := range dependencies {
		key := str(dependency, "module") + " / " + str(dependency, "configuration")
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], dependency)
	}
	return order, groups
}
func directDeclarations(data Object) []Object {
	result := []Object{}
	seen := map[string]bool{}
	add := func(d Object) {
		key := str(d, "module") + "|" + str(d, "configuration") + "|" + str(d, "notation") + "|" + str(d, "version") + "|" + str(d, "type") + "|" + str(d, "classifier")
		if !seen[key] {
			seen[key] = true
			result = append(result, d)
		}
	}
	for _, d := range objects(data["dependencies"]) {
		if str(d, "kind") != "managed declaration" {
			add(d)
		}
	}
	for _, project := range objects(field(data, "resolution")["projects"]) {
		for _, d := range objects(project["declaredDependencies"]) {
			if str(d, "kind") == "dependency" && d["group"] != nil {
				add(Object{"module": project["id"], "configuration": d["configuration"], "notation": str(d, "group") + ":" + str(d, "artifact"), "version": d["version"], "declaredVersion": d["version"], "source": "evaluated Gradle model"})
			}
		}
	}
	return result
}
func selectedVersions(data Object, dependency Object) string {
	versions := map[string]bool{}
	coordinate := str(dependency, "notation")
	for _, graph := range objects(field(data, "resolution")["configurations"]) {
		if graph["module"] != dependency["module"] {
			continue
		}
		for _, node := range objects(graph["nodes"]) {
			if str(node, "group")+":"+str(node, "artifact") == coordinate && node["version"] != nil {
				versions[str(node, "version")] = true
			}
		}
	}
	result := []string{}
	for version := range versions {
		result = append(result, version)
	}
	sort.Strings(result)
	if len(result) == 0 {
		return "unavailable"
	}
	return strings.Join(result, " / ")
}
func selectedBuildTool(data Object) string {
	if data["selectedBuildTool"] == nil {
		return "automatic"
	}
	return str(data, "selectedBuildTool")
}

func toolUpdateNote(tool Object) string {
	if tool["note"] == nil {
		return ""
	}
	return str(tool, "note")
}
