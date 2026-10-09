package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/edwardnoaland/scryer/internal/application"
	"github.com/edwardnoaland/scryer/internal/contract"
)

type fakeAnalyzer struct {
	calls   int
	request contract.ScanRequest
}

func (f *fakeAnalyzer) Stack() string { return "java" }
func (f *fakeAnalyzer) Scan(ctx context.Context, r contract.ScanRequest) (contract.Document, error) {
	f.calls++
	f.request = r
	return contract.Document{Operation: "scan", Data: json.RawMessage(`{"schemaVersion":1,"root":"/repo","builds":[],"dependencies":[],"modules":[],"language":{},"resolution":{},"testing":{},"verification":{}}`)}, nil
}
func (f *fakeAnalyzer) Analyze(context.Context, contract.AnalyzeRequest) (contract.Document, error) {
	f.calls++
	return contract.Document{}, nil
}
func TestCommandsValidateBeforeInvokingAnalyzers(t *testing.T) {
	for _, args := range [][]string{{"scan", ".", "--build-tool", "unknown"}, {"scan", ".", "--color", "bad"}, {"analyze", "--before", "HEAD", "--after", ".", "--color", "bad"}, {"analyze", "--before", "HEAD"}, {"analyze", "--before", "HEAD", "--after", "HEAD", "--json", "--verbose"}, {"analyze", "--before", "HEAD", "--after", "HEAD", "--skip-tests", "--test-command", "echo test"}, {"analyze", "--before", "HEAD", "--after", "HEAD", "-o", "bad.txt"}, {"scan", ".", "--stack", "cpp"}} {
		fake := &fakeAnalyzer{}
		out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
		root := NewRoot(&application.Service{Analyzer: fake}, out, errOut)
		root.SetArgs(args)
		if err := root.Execute(); err == nil {
			t.Fatalf("accepted %v", args)
		}
		if fake.calls != 0 {
			t.Fatal("invalid args reached analyzer")
		}
	}
}
func TestScanPresentationFlagsStayInGo(t *testing.T) {
	fake := &fakeAnalyzer{}
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	root := NewRoot(&application.Service{Analyzer: fake}, out, errOut)
	root.SetArgs([]string{"scan", t.TempDir(), "--static", "--remote-list", "--build-tool", "maven", "--json", "--color", "never"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if fake.calls != 1 || !fake.request.Static || !fake.request.RemoteList || fake.request.BuildTool != "maven" {
		t.Fatal("lost analysis arguments")
	}
	if !json.Valid(out.Bytes()) {
		t.Fatal("stdout was not JSON")
	}
	if errOut.Len() != 0 {
		t.Fatal("unexpected diagnostic")
	}
}

func TestScanColorFlagControlsOnlyTerminalOutput(t *testing.T) {
	for _, sample := range []struct {
		mode    string
		json    bool
		colored bool
	}{
		{"always", false, true}, {"never", false, false}, {"auto", false, false}, {"always", true, false},
	} {
		fake := &fakeAnalyzer{}
		out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
		root := NewRoot(&application.Service{Analyzer: fake}, out, errOut)
		args := []string{"scan", t.TempDir(), "--static", "--color", sample.mode}
		if sample.json {
			args = append(args, "--json")
		}
		root.SetArgs(args)
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(out.Bytes(), []byte{27}) != sample.colored {
			t.Fatalf("incorrect color for %+v", sample)
		}
	}
}
