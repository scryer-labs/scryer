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
