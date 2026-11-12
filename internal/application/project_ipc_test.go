package application

import (
	"encoding/json"
	"testing"

	"github.com/edwardnoaland/scryer/internal/contract"
)

func TestIPCMatchesComponentsThenReusesEndpointsAcrossProjects(t *testing.T) {
	local := []contract.FactOutcome{{Fact: "ipc", Engine: "cpp", Status: "collected", Data: json.RawMessage(`{
		"windows": {
			"namedPipe": {
				"endpoints": [
					{"role":"client","name":"\\\\.\\pipe\\demo","sourceRole":"source","accessMode":"read-write","entrypoints":[{"componentIds":["client-exe"]}]},
					{"role":"server","name":"\\\\.\\pipe\\demo","sourceRole":"source","accessMode":"read-write","entrypoints":[{"componentIds":["server-exe","client-exe"]}]},
					{"role":"server","name":"\\\\.\\pipe\\demo","sourceRole":"test","accessMode":"read-write"}
				]
			},
			"sharedMemory": {
				"mappings": [{"name":"Local\\demo","operations":[
					{"operation":"create","sourceRole":"source","entrypoints":[{"componentIds":["server-exe"]}]},
					{"operation":"open","sourceRole":"test","entrypoints":[{"componentIds":["tester-exe"]}]}
				]}]
			}
		}
	}`)}}
	project := aggregateProjectIPC(local)
	pipes := project.Windows.NamedPipe
	if len(pipes.Endpoints) != 4 || len(pipes.Links) != 1 {
		t.Fatalf("unexpected project named pipes: %+v", pipes)
	}
	if link := pipes.Links[0]; link.From != "client-exe" || link.To != "server-exe" || link.Direction != "duplex" {
		t.Fatalf("wrong project pipe link: %+v", link)
	}
	memory := project.Windows.SharedMemory
	if len(memory.Links) != 1 || memory.Links[0].From != "tester-exe" || memory.Links[0].To != "server-exe" {
		t.Fatalf("wrong project mapping link: %+v", memory.Links)
	}

	remote := aggregateProjectIPC([]contract.FactOutcome{{Fact: "ipc", Engine: "rust", Status: "collected", Data: json.RawMessage(`{
		"windows": {
			"namedPipe": {"endpoints":[{"role":"client","name":"\\\\.\\pipe\\demo","sourceRole":"source","accessMode":"write"}]},
			"sharedMemory": {"mappings":[{"name":"Local\\demo","operations":[{"operation":"open","sourceRole":"source"}]}]}
		}
	}`)}})
	workspace := aggregateWorkspaceIPC([]WorkspaceProject{
		{Path: "local", Facts: &contract.ProjectFacts{Communication: &contract.Communication{IPC: project}}},
		{Path: "remote", Facts: &contract.ProjectFacts{Communication: &contract.Communication{IPC: remote}}},
	}, nil)
	for _, link := range workspace.Windows.NamedPipe.Links {
		if link.From == "remote" && (link.To != "local" || link.Direction != "client-to-server") {
			t.Fatalf("wrong workspace pipe link: %+v", link)
		}
		if link.From == "local" && link.Status != "unmatched" {
			t.Fatalf("workspace matched an internal relationship again: %+v", link)
		}
	}
	if len(workspace.Windows.NamedPipe.Links) != 2 {
		t.Fatalf("missing workspace pipe links: %+v", workspace.Windows.NamedPipe.Links)
	}
	memory = workspace.Windows.SharedMemory
	if len(memory.Links) != 1 || memory.Links[0].From != "remote" || memory.Links[0].To != "local" {
		t.Fatalf("wrong workspace mapping link: %+v", memory.Links)
	}
	for _, endpoint := range memory.Endpoints {
		if endpoint.ProjectEndpointID == "" || endpoint.Project == "" {
			t.Fatalf("lost the project endpoint reference: %+v", endpoint)
		}
	}
}
