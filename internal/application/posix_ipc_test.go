package application

import (
	"encoding/json"
	"github.com/edwardnoaland/scryer/internal/contract"
	"testing"
)

func TestPOSIXResourcesReuseLanguageNeutralEndpointsAcrossScopes(t *testing.T) {
	outcomes := []contract.FactOutcome{{Fact: "ipc", Status: "collected", Engine: "cpp", Data: json.RawMessage(`{"posix":{"unixSocket":{"endpoints":[{"name":"/tmp/demo","transport":"datagram","role":"send","entrypoints":[{"componentIds":["client"]}]},{"name":"/tmp/demo","transport":"datagram","role":"bind","entrypoints":[{"componentIds":["server"]}]}]},"sharedMemory":{"endpoints":[{"name":"/demo","role":"open","entrypoints":[{"componentIds":["client"]}]},{"name":"/demo","role":"create","entrypoints":[{"componentIds":["server"]}]}]}}}`)}}
	local := aggregateProjectIPC(outcomes)
	if len(local.Posix.UnixSocket.Links) != 1 || local.Posix.UnixSocket.Links[0].To != "server" || local.Posix.UnixSocket.Links[0].Basis != "resource-name-candidate" {
		t.Fatal(local.Posix)
	}
	if len(local.Posix.SharedMemory.Links) != 1 || local.Posix.SharedMemory.Links[0].To != "server" {
		t.Fatal(local.Posix)
	}
	remote := aggregateProjectIPC([]contract.FactOutcome{{Fact: "ipc", Status: "collected", Engine: "another-engine", Data: json.RawMessage(`{"posix":{"unixSocket":{"endpoints":[{"name":"/tmp/demo","transport":"datagram","role":"send"}]},"sharedMemory":{"endpoints":[{"name":"/demo","role":"open"}]}}}`)}})
	workspace := aggregateWorkspaceIPC([]WorkspaceProject{{Path: "provider", Facts: &contract.ProjectFacts{Communication: &contract.Communication{IPC: local}}}, {Path: "consumer", Facts: &contract.ProjectFacts{Communication: &contract.Communication{IPC: remote}}}}, nil)
	found := false
	for _, link := range workspace.Posix.UnixSocket.Links {
		if link.From == "consumer" && link.To == "provider" {
			found = true
		}
	}
	if !found {
		t.Fatal(workspace.Posix)
	}
}
func TestPOSIXDoesNotPairUnknownOwnersOrDifferentTransports(t *testing.T) {
	ipc := newIPC()
	resources := &addPOSIX(ipc).UnixSocket
	resources.Endpoints = []contract.IPCEndpoint{{ID: "a", Name: "/tmp/demo", Role: "send", Transport: "datagram", ComponentID: "a"}, {ID: "b", Name: "/tmp/demo", Role: "bind", Transport: "stream", ComponentID: "b"}, {ID: "c", Name: "/tmp/demo", Role: "bind", Transport: "datagram"}}
	matchIPCScope(ipc, func(endpoint contract.IPCEndpoint) string { return endpoint.ComponentID })
	if len(resources.Links) != 1 || resources.Links[0].Status != "unmatched" {
		t.Fatal(resources)
	}
}
