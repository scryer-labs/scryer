package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edwardnoaland/scryer/internal/application"
	"github.com/edwardnoaland/scryer/internal/contract"
)

type fakeAnalyzer struct {
	calls          int
	request        contract.ScanRequest
	analyzeRequest contract.AnalyzeRequest
}

func (f *fakeAnalyzer) Stack() string { return "java" }
func (f *fakeAnalyzer) Scan(ctx context.Context, r contract.ScanRequest) (contract.Document, error) {
	f.calls++
	f.request = r
	return contract.Document{Operation: "scan", Data: json.RawMessage(`{"schemaVersion":1,"root":"/repo","builds":[],"dependencies":[],"modules":[],"language":{},"resolution":{},"testing":{},"verification":{}}`)}, nil
}
func (f *fakeAnalyzer) Analyze(_ context.Context, request contract.AnalyzeRequest) (contract.Document, error) {
	f.calls++
	f.analyzeRequest = request
	data, err := json.Marshal(map[string]any{
		"schemaVersion": 1, "repository": request.Repository,
		"beforeSha": request.Before, "afterSha": request.After,
		"execution": map[string]any{"status": "SKIPPED"},
	})
	return contract.Document{Stack: "java", Operation: "analyze", Data: data}, err
}
func TestCommandsValidateBeforeInvokingAnalyzers(t *testing.T) {
	for _, args := range [][]string{{"scan", ".", "--build-tool", "unknown"}, {"scan", ".", "--color", "bad"}, {"analyze", "--before", "HEAD", "--after", ".", "--color", "bad"}, {"analyze", "--before", "HEAD"}, {"analyze", "--before", "HEAD", "--after", "HEAD", "--json", "--verbose"}, {"analyze", "--before", "HEAD", "--after", "HEAD", "--skip-tests", "--test-command", "echo test"}, {"analyze", "--before", "HEAD", "--after", "HEAD", "-o", "bad.txt"}, {"scan", ".", "--stack", "cpp"}} {
		fake := &fakeAnalyzer{}
		out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
		root := NewRoot(&application.Service{Analyzer: fake}, out, errOut)
		root.SetArgs(append([]string{"project"}, args...))
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
	root.SetArgs([]string{"project", "scan", t.TempDir(), "--static", "--remote-list", "--build-tool", "maven", "--json", "--color", "never"})
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
		args := []string{"project", "scan", t.TempDir(), "--static", "--color", sample.mode}
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

func Test_should_reject_refactor_plan_when_command_has_been_removed(t *testing.T) {
	fake := &fakeAnalyzer{}
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	root := NewRoot(&application.Service{Analyzer: fake}, out, errOut)
	root.SetArgs([]string{"refactor-plan", "generate"})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), `unknown command "refactor-plan"`) {
		t.Fatalf("expected removed command to be rejected, got %v", err)
	}
	if exitCode(err) != 2 || fake.calls != 0 || out.Len() != 0 {
		t.Fatal("removed command must fail as a usage error without invoking an analyzer or producing output")
	}
}

func Test_should_default_to_current_directory_when_project_scan_has_no_path(t *testing.T) {
	fake := &fakeAnalyzer{}
	var out, diagnostics bytes.Buffer
	root := NewRoot(&application.Service{Analyzer: fake}, &out, &diagnostics)
	root.SetArgs([]string{"project", "scan", "--static", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if fake.calls != 1 || fake.request.Repository != cwd || !fake.request.Static || !json.Valid(out.Bytes()) {
		t.Fatal("default path or scan behavior was not preserved")
	}
}

func Test_should_show_scope_commands_when_root_help_is_requested(t *testing.T) {
	var out, diagnostics bytes.Buffer
	root := NewRoot(nil, &out, &diagnostics)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, scope := range []string{"project", "workspace"} {
		if !strings.Contains(out.String(), scope) {
			t.Fatalf("missing scope %s", scope)
		}
	}
	for _, legacy := range []string{"  scan ", "  analyze ", "refactor-plan"} {
		if strings.Contains(out.String(), legacy) {
			t.Fatalf("legacy command advertised: %s", legacy)
		}
	}
}

func Test_should_return_usage_errors_when_nested_command_arguments_are_invalid(t *testing.T) {
	for _, args := range [][]string{
		{"project", "scan", "one", "two"},
		{"project", "scan", "--facts", "stack,loc"},
		{"project", "analyze", "--before", "HEAD"},
		{"workspace", "import"},
		{"workspace", "import", "--projects", "p.csv", "--dependencies", "d.csv", "-o", "bad.yaml"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			fake := &fakeAnalyzer{}
			var out, diagnostics bytes.Buffer
			root := NewRoot(&application.Service{Analyzer: fake}, &out, &diagnostics)
			root.SetArgs(args)
			err := root.Execute()
			if err == nil || exitCode(err) != 2 || fake.calls != 0 {
				t.Fatalf("expected usage error before analyzer invocation, got %v", err)
			}
		})
	}
}

func Test_should_preserve_json_when_transitional_entries_forward_to_scope_commands(t *testing.T) {
	for _, args := range [][]string{{"scan", "--static", "--json"}, {"project", "scan", "--static", "--json"}} {
		fake := &fakeAnalyzer{}
		var out, diagnostics bytes.Buffer
		root := NewRoot(&application.Service{Analyzer: fake}, &out, &diagnostics)
		root.SetArgs(args)
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
		if !json.Valid(out.Bytes()) || fake.calls != 1 {
			t.Fatal("transitional invocation changed the JSON result")
		}
	}

	dir := t.TempDir()
	projects, dependencies := filepath.Join(dir, "projects.csv"), filepath.Join(dir, "dependencies.csv")
	for path, contents := range map[string]string{projects: "projectId\na\nb\n", dependencies: "source,target\nb,a\n"} {
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var expected []byte
	for _, command := range [][]string{{"workspace", "import"}, {"analyze", "workspace"}} {
		var out, diagnostics bytes.Buffer
		root := NewRoot(nil, &out, &diagnostics)
		root.SetArgs(append(command, "--projects", projects, "--dependencies", dependencies, "--json"))
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
		if !json.Valid(out.Bytes()) {
			t.Fatal("workspace stdout is not JSON")
		}
		if expected == nil {
			expected = append([]byte{}, out.Bytes()...)
		} else if !bytes.Equal(expected, out.Bytes()) {
			t.Fatal("workspace fact contract changed between entries")
		}
	}
}

func Test_should_preserve_analysis_options_when_project_analyze_uses_default_or_explicit_path(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "--quiet"},
		{"-c", "user.name=Scryer Test", "-c", "user.email=test@example.invalid", "commit", "--quiet", "--allow-empty", "-m", "baseline"},
	} {
		command := exec.Command("git", args...)
		command.Dir = dir
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("prepare repository: %v: %s", err, output)
		}
	}
	t.Chdir(dir)
	for _, entry := range [][]string{{"project", "analyze"}, {"analyze"}} {
		for _, path := range [][]string{nil, {dir}} {
			fake := &fakeAnalyzer{}
			var out, diagnostics bytes.Buffer
			root := NewRoot(&application.Service{Analyzer: fake}, &out, &diagnostics)
			args := append(append([]string{}, entry...), path...)
			root.SetArgs(append(args, "--before", "HEAD", "--after", "HEAD", "--build-tool", "maven", "--test-command", "./mvnw verify", "--json"))
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			request := fake.analyzeRequest
			if fake.calls != 1 || request.BuildTool != "maven" || request.TestCommand != "./mvnw verify" || request.Before != request.After || len(request.Before) != 40 || !json.Valid(out.Bytes()) {
				t.Fatal("nested analysis lost input options or snapshot resolution")
			}
			if (diagnostics.Len() != 0) != (len(entry) == 1) {
				t.Fatal("compatibility warning must be emitted only by the legacy entry")
			}
		}
	}
}
