package report

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/edwardnoaland/scryer/internal/contract"
)

func TestTerminalColorPreservesFactsAndExportFormats(t *testing.T) {
	scan := contract.Document{Operation: "scan", Data: json.RawMessage(`{"schemaVersion":1,"root":"/repo","builds":[],"dependencies":[],"modules":[],"language":{},"resolution":{"status":"partial"},"testing":{},"verification":{},"remoteVersions":{"dependencies":[{"module":".","configuration":"test","coordinate":"g:a","current":"1","latest":"2","status":"UPDATE_AVAILABLE"}]}}`)}
	strip := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	for _, document := range []contract.Document{scan, fixtureDocument()} {
		var plain, colored bytes.Buffer
		if err := Terminal(document, &plain, true, true, true, false); err != nil {
			t.Fatal(err)
		}
		if err := Terminal(document, &colored, true, true, true, true); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(colored.String(), "\x1b[1;36m") {
			t.Fatal("missing colored title")
		}
		if strings.Contains(plain.String(), "\x1b[") {
			t.Fatal("disabled color emitted ANSI")
		}
		if strip.ReplaceAllString(colored.String(), "") != plain.String() {
			t.Fatal("color changed report content or alignment")
		}
		md, err := Markdown(document, true)
		if err != nil {
			t.Fatal(err)
		}
		data, err := JSON(document)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(md, []byte{27}) || bytes.Contains(data, []byte{27}) {
			t.Fatal("ANSI leaked into export")
		}
	}
	var colored bytes.Buffer
	if err := Terminal(fixtureDocument(), &colored, false, false, true, true); err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{"\x1b[33mUNKNOWN", "\x1b[1;31m", "\x1b[35m"} {
		if !strings.Contains(colored.String(), needle) {
			t.Fatalf("missing semantic style %q", needle)
		}
	}
}

func TestTerminalStatusesAndZeroCounts(t *testing.T) {
	style := terminalStyle{true}
	for value, code := range map[string]string{"SUCCEEDED": "32", "FAILED": "31", "EXECUTED": "32", "NOT_EXECUTED": "31", "PARTIALLY_EXECUTED": "33", "UNKNOWN": "33", "SKIPPED": "2"} {
		if got := style.status(value); got != "\x1b["+code+"m"+value+"\x1b[0m" {
			t.Fatalf("status %s: %s", value, got)
		}
	}
	result := style.counts(Object{"FAILED": 0, "PASSED": 41})
	if strings.Contains(result, "\x1b[31mFAILED") || !strings.Contains(result, "\x1b[32mPASSED") {
		t.Fatal("zero failures should not appear as actual failures")
	}
}
