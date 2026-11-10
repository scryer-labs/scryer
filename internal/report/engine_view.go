package report

import (
	"fmt"
	"io"
)

// engineView contains presentation choices. Engines supply JSON, not rendering code.
type engineView struct {
	scanTerminal func(Object, io.Writer, bool, bool)
	scanMarkdown func(Object, bool) string
	analysis     analyzeView
}

type analyzeView struct {
	changedSymbols   string
	sourceLanguage   string
	runtimeField     string
	runtimeLabel     string
	htmlRuntimeLabel string
}

func viewFor(stack string) engineView {
	switch stack {
	case "", "java":
		// Existing unlabelled documents use the legacy Java v1 schema.
		return javaView
	default:
		return engineView{
			scanTerminal: func(data Object, out io.Writer, _, _ bool) {
				fmt.Fprintln(out, jsonText(data))
			},
			scanMarkdown: func(data Object, _ bool) string {
				return "\x60\x60\x60json\n" + jsonText(data) + "\n\x60\x60\x60\n"
			},
			analysis: analyzeView{changedSymbols: "Changed symbols"},
		}
	}
}
