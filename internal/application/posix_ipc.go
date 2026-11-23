package application

import (
	"encoding/json"
	"strings"

	"github.com/edwardnoaland/scryer/internal/contract"
)

type posixResourceInput struct {
	Endpoints    []json.RawMessage `json:"endpoints"`
	Observations []json.RawMessage `json:"observations"`
}

func addPOSIX(result *contract.IPC) *contract.POSIXIPC {
	if result.Posix == nil {
		empty := func() contract.RuntimeResources {
			return contract.RuntimeResources{Endpoints: []contract.IPCEndpoint{}, Links: []contract.RuntimeResourceLink{}, Observations: []contract.RuntimeObservation{}}
		}
		result.Posix = &contract.POSIXIPC{UnixSocket: empty(), SharedMemory: empty()}
	}
	return result.Posix
}
func addProjectPOSIX(result *contract.IPC, input ipcInput) {
	if input.Posix == nil {
		return
	}
	posix := addPOSIX(result)
	add := func(target *contract.RuntimeResources, source posixResourceInput, mechanism string) {
		for _, raw := range source.Endpoints {
			endpoint, entries, ok := ipcObservation(input.engine, raw)
			if !ok || endpoint.Name == "" {
				continue
			}
			for _, participant := range endpointParticipants(endpoint, entries) {
				appendEndpoint(&target.Endpoints, mechanism, participant)
			}
		}
		for _, raw := range source.Observations {
			target.Observations = append(target.Observations, contract.RuntimeObservation{Engine: input.engine, Data: raw})
		}
	}
	add(&posix.UnixSocket, input.Posix.UnixSocket, "posix-unix-socket")
	add(&posix.SharedMemory, input.Posix.SharedMemory, "posix-shared-memory")
}
func addWorkspacePOSIX(result, projectIPC *contract.IPC, project string) {
	if projectIPC.Posix == nil {
		return
	}
	target := addPOSIX(result)
	add := func(out *contract.RuntimeResources, in contract.RuntimeResources, mechanism string) {
		for _, endpoint := range in.Endpoints {
			appendWorkspaceEndpoint(&out.Endpoints, mechanism, project, endpoint)
		}
		for _, observation := range in.Observations {
			observation.Project = project
			out.Observations = append(out.Observations, observation)
		}
	}
	add(&target.UnixSocket, projectIPC.Posix.UnixSocket, "posix-unix-socket")
	add(&target.SharedMemory, projectIPC.Posix.SharedMemory, "posix-shared-memory")
}
func matchPOSIXScope(ipc *contract.IPC, participant func(contract.IPCEndpoint) string) {
	if ipc.Posix == nil {
		return
	}
	match := func(resources *contract.RuntimeResources, sourceRole, peerRole, transport, direction string) {
		var endpoints []contract.IPCEndpoint
		for _, endpoint := range resources.Endpoints {
			if transport == "" || endpoint.Transport == transport {
				endpoints = append(endpoints, endpoint)
			}
		}
		// Relative socket paths depend on process working directories and cannot be paired safely.
		key := func(name string) string {
			if strings.HasPrefix(name, "/") || strings.HasPrefix(name, "\x00") {
				return name
			}
			return ""
		}
		for _, found := range matchIPCEndpoints(endpoints, sourceRole, peerRole, key, participant) {
			link := contract.RuntimeResourceLink{IPCLink: found.link, SourceEndpointID: found.source.ID, Basis: "resource-name-candidate", Direction: direction}
			for _, peer := range found.peers {
				link.PeerEndpointIDs = append(link.PeerEndpointIDs, peer.ID)
			}
			resources.Links = append(resources.Links, link)
		}
	}
	match(&ipc.Posix.UnixSocket, "send", "bind", "datagram", "sender-to-bound-receiver")
	match(&ipc.Posix.SharedMemory, "open", "create", "", "unknown")
}
