package report

import (
	"fmt"
	"io"
	"strings"
)

func scanTerminal(data Object, out io.Writer, dependencies, tree, color bool) {
	header := func(title string) { fmt.Fprintln(out, title) }
	row := func(key string, value any) { fmt.Fprintf(out, "  %-26s%s\n", key, text(value)) }
	header("Project")
	row("Repository", data["root"])
	language := field(data, "language")
	row("Java source / target", join(language["sourceVersions"])+" / "+join(language["targetVersions"]))
	row("Toolchain", join(language["toolchainVersions"]))
	builds := []string{}
	wrappers := []string{}
	for _, build := range objects(data["builds"]) {
		builds = append(builds, str(build, "tool")+" "+str(build, "version"))
		status := "not found"
		if boolean(build["wrapperPresent"]) {
			status = "present"
		}
		wrappers = append(wrappers, str(build, "tool")+": "+status)
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
		fmt.Fprintln(out, "  "+note)
	}
	if remote := field(data, "remoteVersions"); remote != nil {
		header("\nRemote dependency releases")
		fmt.Fprintln(out, "  "+str(remote, "source"))
		fmt.Fprintln(out, "  Checked: "+str(remote, "checkedAt"))
		order, groups := group(objects(remote["dependencies"]))
		for _, key := range order {
			fmt.Fprintln(out, "\n  "+key)
			for _, d := range groups[key] {
				comparison := str(d, "current") + " → " + str(d, "latest") + " [" + str(d, "status") + "]"
				if color {
					code := "31"
					switch str(d, "status") {
					case "CURRENT":
						code = "32"
					case "UPDATE_AVAILABLE", "CURRENT_AHEAD":
						code = "33"
					}
					comparison = "\x1b[" + code + "m" + comparison + "\x1b[0m"
				}
				fmt.Fprintf(out, "    %s (%s): %s\n", str(d, "coordinate"), str(d, "currentSource"), comparison)
				if d["note"] != nil {
					fmt.Fprintln(out, "      "+str(d, "note"))
				}
			}
		}
	}
	if dependencies {
		header("\nDirect dependencies")
		order, groups := group(declared)
		for _, key := range order {
			fmt.Fprintln(out, "  "+key)
			for _, d := range groups[key] {
				fmt.Fprintf(out, "    %s:%s; selected %s; %s\n", str(d, "notation"), str(d, "version"), selectedVersions(data, d), str(d, "source"))
			}
		}
	}
	if tree {
		dependencyTree(data, out)
	}
}

func dependencyTree(data Object, out io.Writer) {
	fmt.Fprintln(out, "\nResolved dependency tree (selected graph; shared/cyclic nodes expanded once)")
	for _, graph := range objects(field(data, "resolution")["configurations"]) {
		fmt.Fprintf(out, "  %s / %s [%s]\n", str(graph, "module"), str(graph, "configuration"), str(graph, "status"))
		if graph["error"] != nil {
			fmt.Fprintln(out, "    "+str(graph, "error"))
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
					label += " UNRESOLVED " + str(edge, "unresolved")
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
