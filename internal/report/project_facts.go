package report

import (
	"fmt"
	"io"
	"strings"
)

func writeProjectFacts(out io.Writer, data Object) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Project: %s\n", str(data, "root"))
	if stack := field(data, "stack"); stack != nil {
		fmt.Fprintln(&b, "\nStack")
		for _, language := range objects(stack["languages"]) {
			fmt.Fprintf(&b, "  %s: %s files\n", str(language, "language"), str(language, "files"))
		}
		for _, tool := range objects(stack["buildTools"]) {
			fmt.Fprintf(&b, "  %s: %s\n", str(tool, "tool"), str(tool, "path"))
		}
	}
	if loc := field(data, "loc"); loc != nil {
		fmt.Fprintln(&b, "\nLine counts")
		for _, language := range objects(loc["languages"]) {
			fmt.Fprintf(&b, "  %s: %s code, %s comments, %s blank\n", str(language, "language"), str(language, "code"), str(language, "comments"), str(language, "blank"))
		}
		total := field(loc, "total")
		fmt.Fprintf(&b, "  Total: %s files, %s lines (%s code, %s comments, %s blank)\n", str(total, "files"), str(total, "lines"), str(total, "code"), str(total, "comments"), str(total, "blank"))
	}
	if outcomes := objects(data["engineFacts"]); len(outcomes) != 0 {
		fmt.Fprintln(&b, "\nEngine facts")
		for _, outcome := range outcomes {
			fmt.Fprintf(&b, "  %s [%s]: %s", str(outcome, "fact"), factEngine(outcome), str(outcome, "status"))
			if message := outcome["message"]; message != nil {
				fmt.Fprintf(&b, " — %s", text(message))
			}
			fmt.Fprintln(&b)
			if value := outcome["data"]; value != nil {
				fmt.Fprintln(&b, jsonText(value))
			}
		}
	}
	skipped := field(data, "skipped")
	fmt.Fprintf(&b, "\nSkipped: %s unknown language, %s binary, %s oversized\n", str(skipped, "unknownLanguage"), str(skipped, "binary"), str(skipped, "oversized"))
	_, err := io.WriteString(out, b.String())
	return err
}

func projectFactsMarkdown(data Object) string {
	var b strings.Builder
	section(&b, 1, "Project facts")
	fmt.Fprintf(&b, "%s\n\n", escape(str(data, "root")))
	if stack := field(data, "stack"); stack != nil {
		section(&b, 2, "Stack")
		rows := [][]string{}
		for _, language := range objects(stack["languages"]) {
			rows = append(rows, []string{str(language, "language"), str(language, "files")})
		}
		table(&b, []string{"Language", "Files"}, rows)
		rows = nil
		for _, tool := range objects(stack["buildTools"]) {
			rows = append(rows, []string{str(tool, "tool"), str(tool, "path")})
		}
		if len(rows) > 0 {
			table(&b, []string{"Build tool", "Evidence path"}, rows)
		}
	}
	if loc := field(data, "loc"); loc != nil {
		section(&b, 2, "Line counts")
		rows := [][]string{}
		countRow := func(name string, counts Object) []string {
			return []string{name, str(counts, "files"), str(counts, "lines"), str(counts, "code"), str(counts, "comments"), str(counts, "blank")}
		}
		for _, language := range objects(loc["languages"]) {
			rows = append(rows, countRow(str(language, "language"), language))
		}
		rows = append(rows, countRow("Total", field(loc, "total")))
		table(&b, []string{"Language", "Files", "Lines", "Code", "Comments", "Blank"}, rows)
	}
	if outcomes := objects(data["engineFacts"]); len(outcomes) != 0 {
		section(&b, 2, "Engine facts")
		for _, outcome := range outcomes {
			section(&b, 3, str(outcome, "fact")+" ["+factEngine(outcome)+"]")
			fmt.Fprintf(&b, "%s\n\n", escape(str(outcome, "status")))
			if message := outcome["message"]; message != nil {
				fmt.Fprintf(&b, "%s\n\n", escape(text(message)))
			}
			if value := outcome["data"]; value != nil {
				block(&b, "json", jsonText(value))
			}
		}
	}
	section(&b, 2, "Skipped files")
	skipped := field(data, "skipped")
	table(&b, []string{"Unknown language", "Binary", "Oversized"}, [][]string{{str(skipped, "unknownLanguage"), str(skipped, "binary"), str(skipped, "oversized")}})
	return b.String()
}

func factEngine(outcome Object) string {
	if engine, ok := outcome["engine"].(string); ok {
		return engine
	}
	return "project"
}
