package application

import (
	"strings"

	"github.com/edwardnoaland/scryer/internal/contract"
)

// pipeDirection uses endpoint-local access. The server must allow every
// direction requested by the client for the two handles to be compatible.
func pipeDirection(client, server string) string {
	switch server {
	case "read", "write", "read-write":
	default:
		return "unknown"
	}
	switch client {
	case "write":
		if server == "read" || server == "read-write" {
			return "client-to-server"
		}
	case "read":
		if server == "write" || server == "read-write" {
			return "server-to-client"
		}
	case "read-write":
		if server == "read-write" {
			return "duplex"
		}
	default:
		return "unknown"
	}
	return "incompatible"
}

// The matcher accepts local Windows pipe names. A remote host needs
// deployment identity before it can be paired with a local server.
func pipeKey(name string) string {
	key, ok := strings.CutPrefix(strings.ToLower(name), `\\.\pipe\`)
	if !ok {
		return ""
	}
	return key
}

func matchIPCScope(ipc *contract.IPC, participant func(contract.IPCEndpoint) string) {
	pipes := &ipc.Windows.NamedPipe
	for _, match := range matchIPCEndpoints(pipes.Endpoints, "client", "server", pipeKey, participant) {
		link := contract.PipeLink{IPCLink: match.link, ClientEndpointID: match.source.ID, Direction: "unknown"}
		var direction string
		for _, server := range match.peers {
			link.ServerEndpointIDs = append(link.ServerEndpointIDs, server.ID)
			observed := pipeDirection(match.source.AccessMode, server.AccessMode)
			if direction == "" {
				direction = observed
			} else if direction != observed {
				direction = "unknown"
			}
		}
		if link.Status == "matched" {
			link.Direction = direction
		}
		pipes.Links = append(pipes.Links, link)
	}
	if memory := ipc.Windows.SharedMemory; memory != nil {
		name := func(name string) string { return name }
		for _, match := range matchIPCEndpoints(memory.Endpoints, "open", "create", name, participant) {
			link := contract.SharedMemoryLink{IPCLink: match.link, OpenerEndpointID: match.source.ID}
			for _, creator := range match.peers {
				link.CreatorEndpointIDs = append(link.CreatorEndpointIDs, creator.ID)
			}
			memory.Links = append(memory.Links, link)
		}
	}
}

type ipcMatch struct {
	link   contract.IPCLink
	source contract.IPCEndpoint
	peers  []contract.IPCEndpoint
}

func matchIPCEndpoints(endpoints []contract.IPCEndpoint, sourceRole, peerRole string, resourceKey func(string) string, participant func(contract.IPCEndpoint) string) []ipcMatch {
	peersByResource := map[string][]contract.IPCEndpoint{}
	for _, endpoint := range endpoints {
		key := resourceKey(endpoint.Name)
		if endpoint.Role == peerRole && key != "" && participant(endpoint) != "" {
			peersByResource[key] = append(peersByResource[key], endpoint)
		}
	}
	var matches []ipcMatch
	for _, endpoint := range endpoints {
		from := participant(endpoint)
		if endpoint.Role != sourceRole || from == "" {
			continue
		}
		match := ipcMatch{source: endpoint, link: contract.IPCLink{From: from, Name: endpoint.Name, Status: "unmatched"}}
		for _, peer := range peersByResource[resourceKey(endpoint.Name)] {
			to := participant(peer)
			if to != from {
				match.link.Candidates = appendUnique(match.link.Candidates, to)
				match.peers = append(match.peers, peer)
			}
		}
		switch len(match.link.Candidates) {
		case 0:
		case 1:
			match.link.To, match.link.Status, match.link.Candidates = match.link.Candidates[0], "matched", nil
		default:
			match.link.Status = "ambiguous"
		}
		matches = append(matches, match)
	}
	return matches
}
