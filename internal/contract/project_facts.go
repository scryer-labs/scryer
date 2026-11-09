package contract

const ProjectFactsOperation = "project-facts"

type ProjectFacts struct {
	SchemaVersion int           `json:"schemaVersion"`
	Root          string        `json:"root"`
	Stack         *ProjectStack `json:"stack,omitempty"`
	LOC           *ProjectLOC   `json:"loc,omitempty"`
	Skipped       SkippedFiles  `json:"skipped"`
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
