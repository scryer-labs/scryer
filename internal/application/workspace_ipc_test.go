package application

import (
	"encoding/json"
	"testing"

	"github.com/edwardnoaland/scryer/internal/contract"
)

func TestWorkspaceIPCMatchesDirectAndWrappedEndpointsAcrossEngines(t *testing.T) {
	project := func(path, engine, facts string) WorkspaceProject {
		return WorkspaceProject{Path: path, Facts: &contract.ProjectFacts{EngineFacts: []contract.FactOutcome{
			{Engine: engine, Fact: "ipc", Status: "collected", Data: json.RawMessage(facts)},
		}}}
	}
	projects := []WorkspaceProject{
		project("wrapper", "cpp", `{"windows":{"namedPipe":{"parameterEvidence":[{"symbol":"library::X","parameterIndex":1,"role":"server","sourceRole":"source","accessMode":"read-write"}]}}}`),
		project("app", "cpp", `{"calls":[{"candidateSymbols":["app::library::X","library::X"],"sourceRole":"source","arguments":[{"parameterIndex":1,"value":"\\\\.\\pipe\\Test"}]}]}`),
		project("client", "rust", `{"windows":{"namedPipe":{"endpoints":[{"name":"\\\\.\\pipe\\test","role":"client","sourceRole":"source","accessMode":"read-write"}]}}}`),
		project("test", "cpp", `{"windows":{"namedPipe":{"endpoints":[{"name":"\\\\.\\pipe\\test","role":"server","sourceRole":"test"}]}}}`),
	}
	withoutLibrary := aggregateWorkspaceIPC(projects, nil).Windows.NamedPipe
	if len(withoutLibrary.Links) != 1 || withoutLibrary.Links[0].Status != "unmatched" {
		t.Fatalf("resolved a wrapper without its library relationship: %+v", withoutLibrary.Links)
	}
	libraries := []WorkspaceReference{{From: "app", To: "wrapper", Status: "local"}}
	pipes := aggregateWorkspaceIPC(projects, libraries).Windows.NamedPipe
	if len(pipes.Links) != 1 || pipes.Links[0].From != "client" || pipes.Links[0].To != "app" || pipes.Links[0].Status != "matched" || pipes.Links[0].Direction != "duplex" {
		t.Fatalf("unexpected IPC graph: %+v", pipes.Links)
	}
	if endpoint := pipes.Endpoints[0]; endpoint.LibraryProject != "wrapper" || len(endpoint.ParameterEvidence) == 0 {
		t.Fatalf("lost wrapper evidence: %+v", endpoint)
	}

	projects = append(projects, project("other-client", "cpp", `{"windows":{"namedPipe":{"endpoints":[{"name":"\\\\.\\pipe\\test","role":"client","sourceRole":"source"}]}}}`))
	shared := aggregateWorkspaceIPC(projects, libraries).Windows.NamedPipe
	if len(shared.Links) != 2 || shared.Links[1].Direction != "unknown" || shared.Links[0].ClientEndpointID == shared.Links[1].ClientEndpointID ||
		len(shared.Links[0].ServerEndpointIDs) != 1 || len(shared.Links[1].ServerEndpointIDs) != 1 ||
		shared.Links[0].ServerEndpointIDs[0] != shared.Links[1].ServerEndpointIDs[0] {
		t.Fatalf("expected two clients to share one server endpoint: %+v", shared.Links)
	}

	projects = append(projects, project("other", "rust", `{"windows":{"namedPipe":{"endpoints":[{"name":"\\\\.\\pipe\\test","role":"server","sourceRole":"source"}]}}}`))
	ambiguous := aggregateWorkspaceIPC(projects, libraries).Windows.NamedPipe
	if edge := ambiguous.Links[0]; edge.Status != "ambiguous" || edge.Direction != "unknown" || edge.To != "" || len(edge.Candidates) != 2 {
		t.Fatalf("selected a server from ambiguous candidates: %+v", edge)
	}
}

func TestPipeDirectionRespectsBothAccessModes(t *testing.T) {
	for _, example := range []struct {
		client, server, want string
	}{
		{"write", "read", "client-to-server"},
		{"write", "read-write", "client-to-server"},
		{"read", "write", "server-to-client"},
		{"read", "read-write", "server-to-client"},
		{"read-write", "read-write", "duplex"},
		{"read-write", "read", "incompatible"},
		{"read", "read", "incompatible"},
		{"write", "unknown", "unknown"},
		{"", "read-write", "unknown"},
	} {
		if got := pipeDirection(example.client, example.server); got != example.want {
			t.Errorf("client %q and server %q: got %q, want %q", example.client, example.server, got, example.want)
		}
	}
}
