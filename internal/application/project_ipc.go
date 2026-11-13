package application

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/edwardnoaland/scryer/internal/framework/tauri"
)

type ipcInput struct {
	engine  string
	Calls   []json.RawMessage `json:"calls"`
	Windows struct {
		NamedPipe struct {
			Endpoints  []json.RawMessage `json:"endpoints"`
			Parameters []json.RawMessage `json:"parameterEvidence"`
		} `json:"namedPipe"`
		SharedMemory *struct {
			Mappings []struct {
				Name       string            `json:"name"`
				Operations []json.RawMessage `json:"operations"`
			} `json:"mappings"`
		} `json:"sharedMemory"`
	} `json:"windows"`
}

type ipcEntrypoints []struct {
	ComponentIDs []string `json:"componentIds"`
}

type pipeParameter struct {
	AccessMode  string `json:"accessMode"`
	AccessFlags string `json:"accessFlags"`
	Symbol      string `json:"symbol"`
	Index       int    `json:"parameterIndex"`
	Role        string `json:"role"`
	SourceRole  string `json:"sourceRole"`
}

func ipcInputs(outcomes []contract.FactOutcome) []ipcInput {
	var inputs []ipcInput
	for _, outcome := range outcomes {
		if outcome.Fact != "ipc" || outcome.Status != "collected" {
			continue
		}
		var input ipcInput
		if json.Unmarshal(outcome.Data, &input) == nil {
			input.engine = outcome.Engine
			inputs = append(inputs, input)
		}
	}
	return inputs
}

func collectPipeParameters(inputs []ipcInput) map[string][]json.RawMessage {
	parameters := map[string][]json.RawMessage{}
	for _, input := range inputs {
		for _, raw := range input.Windows.NamedPipe.Parameters {
			var parameter pipeParameter
			if json.Unmarshal(raw, &parameter) == nil && parameter.SourceRole != "test" && parameter.Symbol != "" {
				parameters[parameter.Symbol] = append(parameters[parameter.Symbol], raw)
			}
		}
	}
	return parameters
}

func newIPC() *contract.IPC {
	result := &contract.IPC{}
	result.Windows.NamedPipe = contract.NamedPipes{Endpoints: []contract.IPCEndpoint{}, Links: []contract.PipeLink{}}
	return result
}

func addSharedMemory(ipc *contract.IPC) *contract.SharedMemory {
	if ipc.Windows.SharedMemory == nil {
		ipc.Windows.SharedMemory = &contract.SharedMemory{Endpoints: []contract.IPCEndpoint{}, Links: []contract.SharedMemoryLink{}}
	}
	return ipc.Windows.SharedMemory
}

func aggregateProjectIPC(outcomes []contract.FactOutcome) *contract.IPC {
	inputs := ipcInputs(outcomes)
	if len(inputs) == 0 {
		return nil
	}
	result := newIPC()
	result.Tauri = tauri.Aggregate(outcomes)
	pipes := &result.Windows.NamedPipe
	localParameters := map[string]map[string][]json.RawMessage{"": collectPipeParameters(inputs)}
	for _, input := range inputs {
		for _, raw := range input.Windows.NamedPipe.Endpoints {
			endpoint, entries, ok := ipcObservation(input.engine, raw)
			if ok {
				addPipeEndpoints(pipes, endpoint, entries)
			}
		}
		for _, raw := range input.Calls {
			for _, endpoint := range wrappedPipeEndpoints(input.engine, raw, localParameters, []string{""}) {
				appendEndpoint(&pipes.Endpoints, "named-pipe", endpoint)
			}
		}
		if input.Windows.SharedMemory != nil {
			memory := addSharedMemory(result)
			for _, mapping := range input.Windows.SharedMemory.Mappings {
				if strings.TrimSpace(mapping.Name) == "" {
					continue
				}
				for _, raw := range mapping.Operations {
					endpoint, entries, ok := ipcObservation(input.engine, raw)
					if !ok || (endpoint.Role != "create" && endpoint.Role != "open") {
						continue
					}
					endpoint.Name = mapping.Name
					for _, participant := range endpointParticipants(endpoint, entries) {
						appendEndpoint(&memory.Endpoints, "shared-memory", participant)
					}
				}
			}
		}
	}
	matchIPCScope(result, func(endpoint contract.IPCEndpoint) string { return endpoint.ComponentID })
	return result
}

func ipcObservation(engine string, raw json.RawMessage) (contract.IPCEndpoint, ipcEntrypoints, bool) {
	var observation struct {
		Name        string         `json:"name"`
		Role        string         `json:"role"`
		Operation   string         `json:"operation"`
		SourceRole  string         `json:"sourceRole"`
		AccessMode  string         `json:"accessMode"`
		AccessFlags string         `json:"accessFlags"`
		Entrypoints ipcEntrypoints `json:"entrypoints"`
	}
	if json.Unmarshal(raw, &observation) != nil {
		return contract.IPCEndpoint{}, nil, false
	}
	if observation.Role == "" {
		observation.Role = observation.Operation
	}
	return contract.IPCEndpoint{
		Name: observation.Name, Role: observation.Role, Engine: engine, SourceRole: observation.SourceRole,
		AccessMode: observation.AccessMode, AccessFlags: observation.AccessFlags, Observation: raw,
	}, observation.Entrypoints, true
}

func endpointParticipants(endpoint contract.IPCEndpoint, entries ipcEntrypoints) []contract.IPCEndpoint {
	var components []string
	for _, entry := range entries {
		for _, id := range entry.ComponentIDs {
			if id != "" {
				components = appendUnique(components, id)
			}
		}
	}
	if len(components) == 0 {
		return []contract.IPCEndpoint{endpoint}
	}
	result := make([]contract.IPCEndpoint, 0, len(components))
	for _, component := range components {
		endpoint.ComponentID = component
		result = append(result, endpoint)
	}
	return result
}

func appendEndpoint(endpoints *[]contract.IPCEndpoint, mechanism string, endpoint contract.IPCEndpoint) {
	endpoint.ID = fmt.Sprintf("%s:%d", mechanism, len(*endpoints)+1)
	*endpoints = append(*endpoints, endpoint)
}

func addPipeEndpoints(pipes *contract.NamedPipes, endpoint contract.IPCEndpoint, entries ipcEntrypoints) {
	if (endpoint.Role != "client" && endpoint.Role != "server") || pipeKey(endpoint.Name) == "" {
		return
	}
	for _, participant := range endpointParticipants(endpoint, entries) {
		appendEndpoint(&pipes.Endpoints, "named-pipe", participant)
	}
}

func wrappedPipeEndpoints(engine string, raw json.RawMessage, parameters map[string]map[string][]json.RawMessage, providers []string) []contract.IPCEndpoint {
	var call struct {
		Symbols     []string       `json:"candidateSymbols"`
		SourceRole  string         `json:"sourceRole"`
		Entrypoints ipcEntrypoints `json:"entrypoints"`
		Arguments   []struct {
			Index int    `json:"parameterIndex"`
			Value string `json:"value"`
		} `json:"arguments"`
	}
	if json.Unmarshal(raw, &call) != nil {
		return nil
	}
	for _, symbol := range call.Symbols {
		var project string
		var records []json.RawMessage
		for _, provider := range providers {
			if found := parameters[provider][symbol]; len(found) != 0 {
				if records != nil {
					return nil
				}
				project, records = provider, found
			}
		}
		if len(records) == 0 {
			continue
		}
		var endpoints []contract.IPCEndpoint
		for _, record := range records {
			var parameter pipeParameter
			if json.Unmarshal(record, &parameter) != nil || (parameter.Role != "client" && parameter.Role != "server") {
				continue
			}
			for _, arg := range call.Arguments {
				if arg.Index == parameter.Index && pipeKey(arg.Value) != "" {
					endpoint := contract.IPCEndpoint{
						Engine: engine, Role: parameter.Role, Name: arg.Value,
						SourceRole: call.SourceRole, Call: raw, ParameterEvidence: record, LibraryProject: project,
						AccessMode: parameter.AccessMode, AccessFlags: parameter.AccessFlags,
					}
					endpoints = append(endpoints, endpointParticipants(endpoint, call.Entrypoints)...)
				}
			}
		}
		return endpoints
	}
	return nil
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
