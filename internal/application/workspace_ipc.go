package application

import (
	"encoding/json"

	"github.com/edwardnoaland/scryer/internal/contract"
)

func aggregateWorkspaceIPC(projects []WorkspaceProject, libraries []WorkspaceReference) *contract.IPC {
	result := newIPC()
	pipes := &result.Windows.NamedPipe
	providers := map[string][]string{}
	providerPaths := map[string]bool{}
	for _, library := range libraries {
		if library.Status == "local" {
			providers[library.From] = appendUnique(providers[library.From], library.To)
			providerPaths[library.To] = true
		}
	}
	inputsByProject := map[string][]ipcInput{}
	parameters := map[string]map[string][]json.RawMessage{}
	for _, project := range projects {
		if project.Facts == nil {
			continue
		}
		usesLibrary := len(providers[project.Path]) != 0
		providesLibrary := providerPaths[project.Path]
		if usesLibrary || providesLibrary {
			inputs := ipcInputs(project.Facts.EngineFacts)
			if usesLibrary {
				inputsByProject[project.Path] = inputs
			}
			if providesLibrary {
				parameters[project.Path] = collectPipeParameters(inputs)
			}
		}
	}
	for _, project := range projects {
		if project.Facts == nil {
			continue
		}
		var ipc *contract.IPC
		if project.Facts.Communication != nil {
			ipc = project.Facts.Communication.IPC
		}
		if ipc == nil {
			ipc = aggregateProjectIPC(project.Facts.EngineFacts)
		}
		if ipc == nil {
			continue
		}
		type callKey struct{ engine, observation string }
		localCalls := map[callKey]bool{}
		for _, endpoint := range ipc.Windows.NamedPipe.Endpoints {
			if len(endpoint.Call) != 0 {
				localCalls[callKey{endpoint.Engine, string(endpoint.Call)}] = true
			}
			appendWorkspaceEndpoint(&pipes.Endpoints, "named-pipe", project.Path, endpoint)
		}
		for _, input := range inputsByProject[project.Path] {
			for _, raw := range input.Calls {
				if localCalls[callKey{input.engine, string(raw)}] {
					continue
				}
				for _, endpoint := range wrappedPipeEndpoints(input.engine, raw, parameters, providers[project.Path]) {
					appendWorkspaceEndpoint(&pipes.Endpoints, "named-pipe", project.Path, endpoint)
				}
			}
		}
		if local := ipc.Windows.SharedMemory; local != nil {
			memory := addSharedMemory(result)
			for _, endpoint := range local.Endpoints {
				appendWorkspaceEndpoint(&memory.Endpoints, "shared-memory", project.Path, endpoint)
			}
		}
	}
	matchIPCScope(result, func(endpoint contract.IPCEndpoint) string {
		if endpoint.SourceRole == "test" {
			return ""
		}
		return endpoint.Project
	})
	return result
}

func appendWorkspaceEndpoint(endpoints *[]contract.IPCEndpoint, mechanism, project string, endpoint contract.IPCEndpoint) {
	endpoint.Project, endpoint.ProjectEndpointID = project, endpoint.ID
	appendEndpoint(endpoints, mechanism, endpoint)
}
