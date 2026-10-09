package report

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func percent(value any) string {
	if value == nil {
		return "unknown"
	}
	v, err := strconv.ParseFloat(text(value), 64)
	if err != nil {
		return "unknown"
	}
	return fmt.Sprintf("%.1f%%", v)
}
func analyzeTerminal(data Object, out io.Writer, verbose, color bool) {
	style := terminalStyle{color}
	execution := field(data, "execution")
	evidence := field(data, "evidence")
	after := field(data, "after")
	matching := field(data, "matching")
	fmt.Fprintf(out, "%s\nRepository: %s\nBefore: %s\nAfter: %s\nChanged files: %d\nChanged Java methods: %d\nAfter test command: %s\nTest records: %s (records, not unique IDs)\n", style.heading("Scryer analysis"), str(data, "repository"), str(data, "beforeSha"), afterLabel(data), len(array(data["files"])), len(array(data["changes"])), style.status(str(execution, "status")), style.counts(field(evidence, "testRecords")))
	roles := map[string]int{}
	for _, node := range objects(after["nodes"]) {
		roles[str(node, "role")]++
	}
	fmt.Fprintf(out, "After scope: %d production, %d test, %d unknown-role symbols\nCall boundaries: %d in scope/unknown caller (%s repository sites)\n", roles["PRODUCTION"], roles["TEST"], roles["UNKNOWN"], len(array(after["boundaries"])), str(after, "totalBoundaryCount"))
	fmt.Fprintln(out, style.heading("\nChanges"))
	changes := objects(data["changes"])
	for i, change := range changes {
		if !verbose && i >= 8 {
			fmt.Fprintf(out, "  %d more; use --verbose\n", len(changes)-i)
			break
		}
		version := field(change, "after")
		if version == nil {
			version = field(change, "before")
		}
		fmt.Fprintf(out, "  %s %s\n", str(change, "kind"), style.symbol(str(version, "signature"), "CHANGED", "PRODUCTION"))
	}
	if len(changes) == 0 {
		fmt.Fprintln(out, "  No method-body/signature changes identified; file/class context impact is not analyzed yet.")
	}
	fmt.Fprintln(out, style.heading("\nAfter evidence and gaps"))
	for _, dataset := range objects(matching["datasets"]) {
		fmt.Fprintf(out, "  Dataset: %s\n  %s\n  Method execution: %s (%s/%s assessed; UNKNOWN excluded)\n", str(field(dataset, "artifact"), "source"), style.counts(field(dataset, "counts")), style.fact(percent(dataset["methodExecutionPercent"])), str(dataset, "withHits"), str(dataset, "assessed"))
		methods := []Object{}
		for _, method := range objects(dataset["methods"]) {
			if verbose || str(method, "status") != "EXECUTED" {
				methods = append(methods, method)
			}
		}
		order := func(impact string) int {
			switch impact {
			case "CHANGED":
				return 0
			case "CALLER":
				return 1
			default:
				return 2
			}
		}
		sort.SliceStable(methods, func(i, j int) bool { return order(str(methods[i], "impact")) < order(str(methods[j], "impact")) })
		for i, method := range methods {
			if !verbose && i >= 8 {
				fmt.Fprintf(out, "    %d more; use --verbose\n", len(methods)-i)
				break
			}
			fmt.Fprintf(out, "    %s [%s] %s\n", style.status(str(method, "status")), str(method, "impact"), style.symbol(str(method, "signature"), str(method, "impact"), "PRODUCTION"))
			if verbose {
				fmt.Fprintln(out, "      "+str(method, "reason"))
				for _, route := range objects(method["routes"]) {
					fmt.Fprintf(out, "      %s: %s; passed class records: %s (attribution unknown)\n", str(route, "kind"), str(route, "test"), str(route, "passedClassRecords"))
				}
			}
		}
		if len(methods) == 0 {
			fmt.Fprintln(out, "    No method-level gaps in this dataset; this does not prove all execution paths.")
		}
	}
	for _, removed := range array(matching["removed"]) {
		fmt.Fprintf(out, "  REMOVED %s (after coverage not applicable)\n", text(removed))
	}
	for _, unresolved := range array(matching["unresolvedChanges"]) {
		fmt.Fprintf(out, "  UNRESOLVED CHANGE %s\n", text(unresolved))
	}
	fmt.Fprintln(out, "\nMethod hits and static test routes do not prove a specific test/caller path. Percentages are not safety scores.")
	if execution["log"] != nil {
		fmt.Fprintln(out, "Run log: "+str(execution, "log"))
	}
	if evidence["manifest"] != nil {
		fmt.Fprintln(out, "Evidence manifest: "+str(evidence, "manifest"))
	}
	if verbose {
		fmt.Fprintln(out, style.heading("\nFull file changes"))
		for _, file := range objects(data["files"]) {
			fmt.Fprintf(out, "  %s %s → %s\n", str(file, "status"), str(file, "beforePath"), str(file, "afterPath"))
			for _, lines := range objects(file["lines"]) {
				fmt.Fprintf(out, "    before %s +%s → after %s +%s\n", str(field(lines, "before"), "start"), str(field(lines, "before"), "count"), str(field(lines, "after"), "start"), str(field(lines, "after"), "count"))
			}
		}
		for _, name := range []string{"before", "after"} {
			snapshot := field(data, name)
			fmt.Fprintln(out, style.heading("\n"+name+" impact graph (partial)"))
			terminalGraph(snapshot, out, color)
			for _, boundary := range objects(snapshot["boundaries"]) {
				fmt.Fprintf(out, "  %s %s:%s: %s: %s\n", style.warning("BOUNDARY"), str(boundary, "path"), str(boundary, "line"), str(boundary, "reason"), strings.ReplaceAll(str(boundary, "expression"), "\n", " "))
			}
			for _, note := range notes(snapshot["notes"], snapshot["unresolvedChanges"]) {
				fmt.Fprintln(out, "  Note: "+note)
			}
		}
		fmt.Fprintf(out, "\nExecution details\n  Command: %s\n  JAVA_HOME: %s; exit: %s; duration: %sms\n", join(execution["command"]), str(execution, "javaHome"), str(execution, "exitCode"), str(execution, "durationMillis"))
		for _, test := range objects(evidence["tests"]) {
			fmt.Fprintln(out, "  Test report: "+str(field(test, "artifact"), "source"))
			for _, record := range objects(test["cases"]) {
				fmt.Fprintf(out, "    %s %s#%s\n", style.status(str(record, "status")), str(record, "className"), str(record, "name"))
			}
		}
		for _, coverage := range objects(evidence["coverage"]) {
			fmt.Fprintf(out, "  Coverage: %s %s; %d class records\n", str(coverage, "format"), str(field(coverage, "artifact"), "source"), len(array(coverage["classes"])))
			for _, note := range notes(coverage["notes"]) {
				fmt.Fprintln(out, "    Note: "+note)
			}
		}
		for _, artifact := range objects(evidence["artifacts"]) {
			fmt.Fprintf(out, "  Artifact: %s; SHA-256 %s; %s\n", str(artifact, "source"), str(artifact, "sha256"), str(artifact, "retained"))
		}
	}
	for _, note := range notes(data["notes"], execution["notes"], evidence["notes"], matching["notes"]) {
		fmt.Fprintln(out, style.muted("Note: "+note))
	}
	if !verbose {
		fmt.Fprintln(out, "Use --verbose for full graphs, boundaries and execution details.")
	}
}
func terminalGraph(snapshot Object, out io.Writer, color bool) {
	style := terminalStyle{color}
	nodes := objects(snapshot["nodes"])
	names := labels(nodes)
	byID := map[string]Object{}
	outgoing := map[string][]Object{}
	for _, n := range nodes {
		byID[str(n, "id")] = n
	}
	for _, e := range objects(snapshot["edges"]) {
		outgoing[str(e, "caller")] = append(outgoing[str(e, "caller")], e)
	}
	type item struct {
		id    string
		depth int
	}
	visited := map[string]bool{}
	for _, n := range nodes {
		if visited[str(n, "id")] {
			continue
		}
		queue := []item{{str(n, "id"), 0}}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			if visited[current.id] {
				continue
			}
			visited[current.id] = true
			node := byID[current.id]
			fmt.Fprintf(out, "  %s%s [%s] [%s]\n", strings.Repeat("  ", current.depth), style.symbol(names[str(node, "signature")], str(node, "impact"), str(node, "role")), str(node, "impact"), str(node, "role"))
			for _, edge := range outgoing[current.id] {
				target := str(edge, "callee")
				label := names[str(byID[target], "signature")]
				fmt.Fprintf(out, "  %s→ %s (%s)\n", strings.Repeat("  ", current.depth+1), style.symbol(label, str(byID[target], "impact"), str(byID[target], "role")), str(edge, "kind"))
				if !visited[target] {
					queue = append(queue, item{target, current.depth + 1})
				}
			}
		}
	}
}
func analyzeMarkdown(data Object) string {
	var b strings.Builder
	section(&b, 1, "Scryer analysis")
	execution := field(data, "execution")
	evidence := field(data, "evidence")
	matching := field(data, "matching")
	table(&b, []string{"Fact", "Value"}, rows([]string{"Repository", str(data, "repository")}, []string{"Before", str(data, "beforeSha")}, []string{"After", afterLabel(data)}, []string{"Generated", str(data, "generatedAt")}, []string{"Changed files", fmt.Sprint(len(array(data["files"])))}, []string{"Changed Java methods", fmt.Sprint(len(array(data["changes"])))}, []string{"After command", str(execution, "status")}, []string{"Test records", counts(field(evidence, "testRecords"))}))
	b.WriteString("Test records are not unique test IDs. Method hits and static routes do not prove a particular caller path or individual-test attribution. Percentages are not safety scores.\n\n")
	allNodes := append(objects(field(data, "before")["nodes"]), objects(field(data, "after")["nodes"])...)
	names := labels(allNodes)
	section(&b, 2, "After evidence and gaps")
	for _, dataset := range objects(matching["datasets"]) {
		section(&b, 3, "Dataset: "+str(field(dataset, "artifact"), "source"))
		fmt.Fprintf(&b, "Method execution: **%s** (%s/%s assessed; UNKNOWN excluded).\n\n", percent(dataset["methodExecutionPercent"]), str(dataset, "withHits"), str(dataset, "assessed"))
		table(&b, []string{"Status", "Methods"}, countRows(field(dataset, "counts")))
		methodRows := [][]string{}
		for _, method := range objects(dataset["methods"]) {
			label := names[str(method, "signature")]
			if label == "" {
				label = compact(str(method, "signature"))
			}
			instruction := "unknown"
			if hits := field(method, "instructions"); hits != nil {
				instruction = str(hits, "covered") + " / " + str(hits, "missed")
			}
			methodRows = append(methodRows, []string{label, str(method, "impact"), str(method, "status"), instruction, str(field(method, "branches"), "missed"), filepath.Base(str(method, "path")) + ":" + str(method, "line")})
		}
		table(&b, []string{"Production symbol", "Impact", "Evidence", "Instructions hit / missed", "Missed branches", "Source"}, methodRows)
		for _, method := range objects(dataset["methods"]) {
			fmt.Fprintln(&b, "- "+escape(str(method, "signature"))+": "+escape(str(method, "reason")))
			for _, route := range objects(method["routes"]) {
				fmt.Fprintf(&b, "  - %s: %s; %s passed class records; attribution unknown.\n", escape(str(route, "kind")), escape(str(route, "test")), escape(str(route, "passedClassRecords")))
			}
		}
		b.WriteString("\n")
	}
	section(&b, 2, "Before / after impact graphs")
	b.WriteString("Arrows point from caller to callee. Changed symbols are red, callers blue, potential indirect branches amber and tests purple. All resolved scope edges are retained; boundaries remain explicit.\n\n")
	for _, name := range []string{"before", "after"} {
		snapshot := field(data, name)
		section(&b, 3, name)
		if len(array(snapshot["nodes"])) == 0 {
			b.WriteString("No resolved changed scope symbols.\n\n")
		} else {
			block(&b, "mermaid", mermaid(snapshot))
		}
		fmt.Fprintf(&b, "Boundaries: %d in scope/unknown caller; %s repository sites.\n\n", len(array(snapshot["boundaries"])), str(snapshot, "totalBoundaryCount"))
		boundaryRows := [][]string{}
		for _, boundary := range objects(snapshot["boundaries"]) {
			boundaryRows = append(boundaryRows, []string{str(boundary, "caller"), str(boundary, "path") + ":" + str(boundary, "line"), str(boundary, "reason"), str(boundary, "expression")})
		}
		table(&b, []string{"Caller", "Source", "Reason", "Expression"}, boundaryRows)
		for _, note := range notes(snapshot["notes"], snapshot["unresolvedChanges"]) {
			fmt.Fprintln(&b, "- "+escape(note))
		}
		b.WriteString("\n")
	}
	section(&b, 2, "Changes")
	fileRows := [][]string{}
	for _, file := range objects(data["files"]) {
		ranges := []string{}
		for _, line := range objects(file["lines"]) {
			ranges = append(ranges, "before "+str(field(line, "before"), "start")+" +"+str(field(line, "before"), "count")+" → after "+str(field(line, "after"), "start")+" +"+str(field(line, "after"), "count"))
		}
		fileRows = append(fileRows, []string{str(file, "status"), str(file, "beforePath"), str(file, "afterPath"), strings.Join(ranges, "; ")})
	}
	table(&b, []string{"Status", "Before path", "After path", "Changed line ranges"}, fileRows)
	for _, change := range objects(data["changes"]) {
		section(&b, 3, str(change, "kind"))
		for _, name := range []string{"before", "after"} {
			if version := field(change, name); version != nil {
				fmt.Fprintln(&b, escape(name+": "+str(version, "signature")+"; "+str(version, "path")+":"+str(field(version, "lines"), "start"))+"\n")
				block(&b, "java", str(version, "source"))
			}
		}
	}
	section(&b, 2, "Execution and artifacts")
	table(&b, []string{"Fact", "Value"}, rows([]string{"Status", str(execution, "status")}, []string{"After SHA", str(execution, "afterSha")}, []string{"JAVA_HOME", str(execution, "javaHome")}, []string{"Exit", str(execution, "exitCode")}, []string{"Duration", str(execution, "durationMillis") + "ms"}, []string{"Log", str(execution, "log")}, []string{"Manifest", str(evidence, "manifest")}))
	command := []string{}
	for _, part := range array(execution["command"]) {
		command = append(command, text(part))
	}
	block(&b, "text", strings.Join(command, " "))
	for _, test := range objects(evidence["tests"]) {
		section(&b, 3, "Test report: "+str(field(test, "artifact"), "source"))
		records := [][]string{}
		for _, record := range objects(test["cases"]) {
			records = append(records, []string{str(record, "className"), str(record, "name"), str(record, "status"), str(record, "seconds")})
		}
		table(&b, []string{"Class", "Test record", "Status", "Seconds"}, records)
	}
	for _, dataset := range objects(evidence["coverage"]) {
		section(&b, 3, "Coverage: "+str(field(dataset, "artifact"), "source"))
		classRows := [][]string{}
		for _, class := range objects(dataset["classes"]) {
			classRows = append(classRows, []string{str(class, "name"), str(class, "match"), str(class, "classId"), str(class, "compiledFile")})
		}
		table(&b, []string{"Class", "Bytecode match", "Class ID", "Compiled output"}, classRows)
		for _, note := range notes(dataset["notes"]) {
			fmt.Fprintln(&b, "- "+escape(note))
		}
		b.WriteString("\n")
	}
	artifacts := [][]string{}
	for _, artifact := range objects(evidence["artifacts"]) {
		artifacts = append(artifacts, []string{str(artifact, "source"), str(artifact, "sha256"), str(artifact, "retained")})
	}
	table(&b, []string{"Artifact", "SHA-256", "Retained path"}, artifacts)
	section(&b, 2, "Full symbol index")
	symbols := [][]string{}
	seen := map[string]bool{}
	for _, n := range allNodes {
		key := str(n, "id") + "|" + str(n, "path") + "|" + str(n, "line")
		if seen[key] {
			continue
		}
		seen[key] = true
		symbols = append(symbols, []string{names[str(n, "signature")], str(n, "signature"), str(n, "id"), str(n, "path") + ":" + str(n, "line")})
	}
	table(&b, []string{"Label", "Signature", "Binary identity", "Source"}, symbols)
	section(&b, 2, "Limits and unresolved changes")
	for _, note := range notes(data["notes"], execution["notes"], evidence["notes"], matching["notes"], matching["removed"], matching["unresolvedChanges"]) {
		fmt.Fprintln(&b, "- "+escape(note))
	}
	b.WriteString("\n")
	return b.String()
}
func countRows(values Object) [][]string {
	keys := []string{}
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := [][]string{}
	for _, key := range keys {
		result = append(result, []string{key, text(values[key])})
	}
	return result
}

func afterLabel(data map[string]any) string {
	source := object(data["afterSource"])
	if str(source, "kind") == "WORKING_TREE" {
		return ". (working-tree snapshot " + str(data, "afterSha") + "; base HEAD " + str(source, "baseHead") + ")"
	}
	return str(data, "afterSha")
}
