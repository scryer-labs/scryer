package terminal

import (
	"bytes"
	"testing"
)

func TestColorPreferenceForRedirectedOutput(t *testing.T) {
	output := &bytes.Buffer{}
	t.Setenv("TERM", "xterm-256color")
	if UseColor("auto", output) {
		t.Fatal("redirected output must be plain")
	}
	if !UseColor("always", output) || UseColor("never", output) {
		t.Fatal("explicit color preference lost")
	}
	t.Setenv("NO_COLOR", "")
	if UseColor("auto", output) {
		t.Fatal("NO_COLOR must disable automatic color")
	}
	if !UseColor("always", output) {
		t.Fatal("explicit always must override NO_COLOR")
	}
}
