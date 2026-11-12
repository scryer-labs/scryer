package application

import (
	"encoding/json"
	"fmt"
	"strings"
)

type WorkspaceIPC struct {
	Windows struct {
		NamedPipe WorkspaceNamedPipes `json:"namedPipe"`
	} `json:"windows"`
}

type WorkspaceNamedPipes struct {
	Endpoints []WorkspacePipeEndpoint `json:"endpoints"`
	Links     []WorkspacePipeLink     `json:"links"`
}

type WorkspacePipeEndpoint struct {
	AccessMode        string          `json:"accessMode,omitempty"`
	AccessFlags       string          `json:"accessFlags,omitempty"`
	ID                string          `json:"id"`
	Project           string          `json:"project"`
	Engine            string          `json:"engine"`
	Role              string          `json:"role"`
	Name              string          `json:"name"`
	SourceRole        string          `json:"sourceRole"`
	Observation       json.RawMessage `json:"observation,omitempty"`
	Call              json.RawMessage `json:"call,omitempty"`
	ParameterEvidence json.RawMessage `json:"parameterEvidence,omitempty"`
	LibraryProject    string          `json:"libraryProject,omitempty"`
}

type WorkspacePipeLink struct {
	Direction         string   `json:"direction"`
	From              string   `json:"from"`
	To                string   `json:"to,omitempty"`
	Name              string   `json:"name"`
	Status            string   `json:"status"`
	ClientEndpointID  string   `json:"clientEndpointId"`
	ServerEndpointIDs []string `json:"serverEndpointIds,omitempty"`
	Candidates        []string `json:"candidates,omitempty"`
}

type pipeInput struct {
	Calls   []json.RawMessage `json:"calls"`
	Windows struct {
		NamedPipe struct {
			Endpoints  []json.RawMessage `json:"endpoints"`
			Parameters []json.RawMessage `json:"parameterEvidence"`
		} `json:"namedPipe"`
	} `json:"windows"`
}

type pipeParameter struct {
	AccessMode  string `json:"accessMode"`
	AccessFlags string `json:"accessFlags"`
	Symbol      string `json:"symbol"`
	Index       int    `json:"parameterIndex"`
	Role        string `json:"role"`
	SourceRole  string `json:"sourceRole"`
}

type ipcInput struct {
	project string
	engine  string
	facts   pipeInput
}

func aggregateWorkspaceIPC(projects []WorkspaceProject, libraries []WorkspaceReference) *WorkspaceIPC {
	result := &WorkspaceIPC{}
	pipes := &result.Windows.NamedPipe
	pipes.Endpoints, pipes.Links = []WorkspacePipeEndpoint{}, []WorkspacePipeLink{}
	var inputs []ipcInput
	parameters := map[string]map[string][]json.RawMessage{}
	for _, project := range projects {
		if project.Facts == nil {
			continue
		}
		for _, outcome := range project.Facts.EngineFacts {
			if outcome.Fact != "ipc" || outcome.Status != "collected" {
				continue
			}
			var facts pipeInput
			if json.Unmarshal(outcome.Data, &facts) != nil {
				continue
			}
			inputs = append(inputs, ipcInput{project.Path, outcome.Engine, facts})
			if parameters[project.Path] == nil {
				parameters[project.Path] = map[string][]json.RawMessage{}
			}
			for _, raw := range facts.Windows.NamedPipe.Parameters {
				var parameter pipeParameter
				if json.Unmarshal(raw, &parameter) == nil && parameter.SourceRole != "test" && parameter.Symbol != "" {
					parameters[project.Path][parameter.Symbol] = append(parameters[project.Path][parameter.Symbol], raw)
				}
			}
		}
	}

	providers := map[string][]string{}
	for _, library := range libraries {
		if library.Status == "local" {
			providers[library.From] = appendUnique(providers[library.From], library.To)
		}
	}
	for _, input := range inputs {
		for _, raw := range input.facts.Windows.NamedPipe.Endpoints {
			var endpoint WorkspacePipeEndpoint
			if json.Unmarshal(raw, &endpoint) == nil {
				endpoint.Project, endpoint.Engine, endpoint.Observation = input.project, input.engine, raw
				pipes.add(endpoint)
			}
		}
		for _, raw := range input.facts.Calls {
			pipes.addWrappedEndpoints(input, raw, parameters, providers[input.project])
		}
	}
	pipes.match()
	return result
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func (p *WorkspaceNamedPipes) add(endpoint WorkspacePipeEndpoint) {
	if (endpoint.Role != "client" && endpoint.Role != "server") || pipeKey(endpoint.Name) == "" {
		return
	}
	endpoint.ID = fmt.Sprintf("named-pipe:%d", len(p.Endpoints)+1)
	p.Endpoints = append(p.Endpoints, endpoint)
}

func (p *WorkspaceNamedPipes) addWrappedEndpoints(input ipcInput, raw json.RawMessage, parameters map[string]map[string][]json.RawMessage, providers []string) {
	var call struct {
		Symbols    []string `json:"candidateSymbols"`
		SourceRole string   `json:"sourceRole"`
		Arguments  []struct {
			Index int    `json:"parameterIndex"`
			Value string `json:"value"`
		} `json:"arguments"`
	}
	if json.Unmarshal(raw, &call) != nil {
		return
	}
	for _, symbol := range call.Symbols {
		var project string
		var records []json.RawMessage
		for _, provider := range providers {
			if found := parameters[provider][symbol]; len(found) != 0 {
				if project != "" {
					return
				}
				project, records = provider, found
			}
		}
		if project == "" {
			continue
		}
		for _, record := range records {
			var parameter pipeParameter
			if json.Unmarshal(record, &parameter) != nil {
				continue
			}
			for _, arg := range call.Arguments {
				if arg.Index == parameter.Index {
					p.add(WorkspacePipeEndpoint{
						Project: input.project, Engine: input.engine, Role: parameter.Role, Name: arg.Value,
						SourceRole: call.SourceRole, Call: raw, ParameterEvidence: record, LibraryProject: project,
						AccessMode: parameter.AccessMode, AccessFlags: parameter.AccessFlags,
					})
				}
			}
		}
		return
	}
}

// The first matcher accepts local Windows pipe names. A remote host needs
// deployment identity before it can be paired with a local server.
func pipeKey(name string) string {
	const prefix = `\\.\pipe\`
	normalized := strings.ToLower(name)
	key, ok := strings.CutPrefix(normalized, prefix)
	if !ok || key == "" {
		return ""
	}
	return key
}

func (p *WorkspaceNamedPipes) match() {
	servers := map[string][]WorkspacePipeEndpoint{}
	for _, endpoint := range p.Endpoints {
		if endpoint.Role == "server" && endpoint.SourceRole != "test" {
			key := pipeKey(endpoint.Name)
			servers[key] = append(servers[key], endpoint)
		}
	}
	for _, client := range p.Endpoints {
		if client.Role != "client" || client.SourceRole == "test" {
			continue
		}
		link := WorkspacePipeLink{From: client.Project, Name: client.Name, ClientEndpointID: client.ID, Status: "unmatched", Direction: "unknown"}
		var matchedDirection string
		for _, server := range servers[pipeKey(client.Name)] {
			if server.Project == client.Project {
				continue
			}
			link.Candidates = appendUnique(link.Candidates, server.Project)
			link.ServerEndpointIDs = append(link.ServerEndpointIDs, server.ID)
			direction := pipeDirection(client.AccessMode, server.AccessMode)
			if matchedDirection == "" {
				matchedDirection = direction
			} else if matchedDirection != direction {
				matchedDirection = "unknown"
			}
		}
		switch len(link.Candidates) {
		case 1:
			link.To, link.Status, link.Candidates = link.Candidates[0], "matched", nil
			link.Direction = matchedDirection
		case 0:
		default:
			link.Status = "ambiguous"
		}
		p.Links = append(p.Links, link)
	}
}

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
