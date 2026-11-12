package report

import (
	"fmt"
	"strings"
)

func communicationMarkdown(b *strings.Builder, data Object, componentNames map[string]string) {
	for _, mechanism := range ipcReports(data) {
		links := objects(mechanism.data["links"])
		if len(links) == 0 {
			continue
		}
		section(b, 2, mechanism.title)
		headers := append([]string{}, mechanism.columns...)
		if mechanism.direction {
			headers = append(headers, "Data direction")
		}
		var entries [][]string
		for _, link := range links {
			from, to := ipcParticipants(link, componentNames)
			entry := []string{from, to, str(link, "name"), str(link, "status")}
			if mechanism.direction {
				entry = append(entry, str(link, "direction"))
			}
			entries = append(entries, entry)
		}
		table(b, headers, entries)
	}
}

func writeCommunicationSummary(b *strings.Builder, data Object) {
	names := cppComponentNames(objects(data["engineFacts"]))
	for _, mechanism := range ipcReports(data) {
		links := objects(mechanism.data["links"])
		if len(links) == 0 {
			continue
		}
		fmt.Fprintf(b, "\n%s\n", mechanism.title)
		for _, link := range links {
			from, to := ipcParticipants(link, names)
			status := str(link, "status")
			if mechanism.direction {
				status += ", " + str(link, "direction")
			}
			fmt.Fprintf(b, "  %s / %s [%s] %s\n", from, to, status, str(link, "name"))
		}
	}
}

type ipcReport struct {
	title     string
	data      Object
	columns   []string
	direction bool
}

func ipcReports(data Object) []ipcReport {
	windows := field(field(field(data, "communication"), "ipc"), "windows")
	return []ipcReport{
		{"IPC (Windows named pipes)", field(windows, "namedPipe"), []string{"Client", "Server", "Pipe", "Status"}, true},
		{"IPC (Windows shared memory)", field(windows, "sharedMemory"), []string{"Opener", "Creator", "Mapping", "Status"}, false},
	}
}

func ipcParticipants(link Object, names map[string]string) (string, string) {
	label := func(id string) string {
		if name := names[id]; name != "" {
			return name
		}
		return id
	}
	to := label(optional(link, "to"))
	if to == "" {
		var candidates []string
		for _, candidate := range array(link["candidates"]) {
			candidates = append(candidates, label(text(candidate)))
		}
		to = strings.Join(candidates, ", ")
	}
	return label(str(link, "from")), to
}
