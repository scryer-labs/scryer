package contract

import "encoding/json"

type Communication struct {
	IPC *IPC `json:"ipc,omitempty"`
}

type IPC struct {
	Tauri   *TauriIPC `json:"tauri,omitempty"`
	Windows struct {
		NamedPipe    NamedPipes    `json:"namedPipe"`
		SharedMemory *SharedMemory `json:"sharedMemory,omitempty"`
	} `json:"windows"`
}

type NamedPipes struct {
	Endpoints []IPCEndpoint `json:"endpoints"`
	Links     []PipeLink    `json:"links"`
}

type SharedMemory struct {
	Endpoints []IPCEndpoint      `json:"endpoints"`
	Links     []SharedMemoryLink `json:"links"`
}

type IPCEndpoint struct {
	ID                string          `json:"id"`
	Project           string          `json:"project,omitempty"`
	ProjectEndpointID string          `json:"projectEndpointId,omitempty"`
	ComponentID       string          `json:"componentId,omitempty"`
	Engine            string          `json:"engine"`
	Role              string          `json:"role"`
	Name              string          `json:"name"`
	SourceRole        string          `json:"sourceRole"`
	AccessMode        string          `json:"accessMode,omitempty"`
	AccessFlags       string          `json:"accessFlags,omitempty"`
	Observation       json.RawMessage `json:"observation,omitempty"`
	Call              json.RawMessage `json:"call,omitempty"`
	ParameterEvidence json.RawMessage `json:"parameterEvidence,omitempty"`
	LibraryProject    string          `json:"libraryProject,omitempty"`
}

type IPCLink struct {
	From       string   `json:"from"`
	To         string   `json:"to,omitempty"`
	Name       string   `json:"name"`
	Status     string   `json:"status"`
	Candidates []string `json:"candidates,omitempty"`
}

type PipeLink struct {
	IPCLink
	Direction         string   `json:"direction"`
	ClientEndpointID  string   `json:"clientEndpointId"`
	ServerEndpointIDs []string `json:"serverEndpointIds,omitempty"`
}

type SharedMemoryLink struct {
	IPCLink
	OpenerEndpointID   string   `json:"openerEndpointId"`
	CreatorEndpointIDs []string `json:"creatorEndpointIds,omitempty"`
}

type TauriIPC struct {
	Apps []TauriApp `json:"apps"`
}

type TauriApp struct {
	Config    string           `json:"config"`
	Endpoints []TauriEndpoint  `json:"endpoints"`
	Links     []TauriLink      `json:"links"`
	Unmatched []TauriUnmatched `json:"unmatched,omitempty"`
}

type TauriEndpoint struct {
	Engine           string       `json:"engine"`
	Kind             string       `json:"kind"`
	Role             string       `json:"role"`
	Name             string       `json:"name"`
	File             string       `json:"file"`
	Line             uint         `json:"line"`
	Column           uint         `json:"column"`
	ResolutionReason string       `json:"resolutionReason,omitempty"`
	ModuleID         string       `json:"moduleId,omitempty"`
	Symbol           string       `json:"symbol,omitempty"`
	Declaration      *TauriSource `json:"declaration,omitempty"`
}

type TauriSource struct {
	File   string `json:"file"`
	Line   uint   `json:"line"`
	Column uint   `json:"column"`
}

type TauriUnmatched struct {
	Endpoint TauriEndpoint `json:"endpoint"`
	Reason   string        `json:"reason"`
}

type TauriLink struct {
	Kind         string         `json:"kind"`
	Name         string         `json:"name"`
	From         TauriEndpoint  `json:"from"`
	To           TauriEndpoint  `json:"to"`
	Registration *TauriEndpoint `json:"registration,omitempty"`
}
