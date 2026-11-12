package contract

import "encoding/json"

const ProjectFactsOperation = "project-facts"
const EngineFactsOperation = "engine-facts"

type FactOutcome struct {
	Fact    string          `json:"fact"`
	Engine  string          `json:"engine,omitempty"`
	Status  string          `json:"status"`
	Data    json.RawMessage `json:"data,omitempty"`
	Message string          `json:"message,omitempty"`
}

func IsEngineScanFact(fact string) bool {
	switch fact {
	case "architecture", "files", "ipc", "dependencies", "call-graph", "ownership", "test-coverage":
		return true
	default:
		return false
	}
}

type RepositoryIdentity struct {
	Host      string `json:"host"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	ID        string `json:"id,omitempty"`
	Remote    string `json:"remote,omitempty"`
}

type ProjectFacts struct {
	Communication *Communication     `json:"communication,omitempty"`
	EngineFacts   []FactOutcome      `json:"engineFacts,omitempty"`
	SchemaVersion int                `json:"schemaVersion"`
	Root          string             `json:"root"`
	Name          string             `json:"name"`
	Repository    RepositoryIdentity `json:"repository"`
	Stack         *ProjectStack      `json:"stack,omitempty"`
	LOC           *ProjectLOC        `json:"loc,omitempty"`
	Skipped       SkippedFiles       `json:"skipped"`
}

type ProjectStack struct {
	Languages  []LanguagePresence  `json:"languages"`
	BuildTools []BuildToolEvidence `json:"buildTools"`
}

type LanguagePresence struct {
	Language string `json:"language"`
	Files    int64  `json:"files"`
}

type BuildToolEvidence struct {
	Tool string `json:"tool"`
	Path string `json:"path"`
}

type LineCounts struct {
	Files    int64 `json:"files"`
	Lines    int64 `json:"lines"`
	Code     int64 `json:"code"`
	Comments int64 `json:"comments"`
	Blank    int64 `json:"blank"`
}

type LanguageLOC struct {
	Language string `json:"language"`
	LineCounts
}

type ProjectLOC struct {
	Total     LineCounts    `json:"total"`
	Languages []LanguageLOC `json:"languages"`
}

type SkippedFiles struct {
	UnknownLanguage int `json:"unknownLanguage"`
	Binary          int `json:"binary"`
	Oversized       int `json:"oversized"`
}
