package contract

import "encoding/json"

// These core DTOs let in-process analyzers supply the same v1 contract without a JVM.
// Document.Data remains lossless for additional Java/other-stack facts.
type BuildFact struct {
	Tool           string  `json:"tool"`
	Version        *string `json:"version"`
	Definition     string  `json:"definition"`
	WrapperPresent bool    `json:"wrapperPresent"`
}
type ModuleFact struct {
	ID         string `json:"id"`
	Directory  string `json:"directory"`
	Definition string `json:"definition"`
}
type DependencyFact struct {
	Module          string  `json:"module"`
	Configuration   string  `json:"configuration"`
	Notation        string  `json:"notation"`
	Version         *string `json:"version"`
	DeclaredVersion *string `json:"declaredVersion"`
	Source          string  `json:"source"`
	Kind            string  `json:"kind"`
}
type LanguageFacts struct {
	SourceVersions    []string `json:"sourceVersions"`
	TargetVersions    []string `json:"targetVersions"`
	ToolchainVersions []string `json:"toolchainVersions"`
}
type ToolUpdate struct {
	Tool          string  `json:"tool"`
	Channel       string  `json:"channel"`
	Current       *string `json:"current"`
	CurrentSource string  `json:"currentSource"`
	Latest        *string `json:"latest"`
	Status        string  `json:"status"`
	MetadataURL   *string `json:"metadataUrl"`
	Note          *string `json:"note"`
}
type ToolUpdates struct {
	CheckedAt string       `json:"checkedAt"`
	Tools     []ToolUpdate `json:"tools"`
	Notes     []string     `json:"notes"`
}
type ScanReport struct {
	ToolUpdates   *ToolUpdates               `json:"toolUpdates,omitempty"`
	SchemaVersion int                        `json:"schemaVersion"`
	Root          string                     `json:"root"`
	Builds        []BuildFact                `json:"builds"`
	Modules       []ModuleFact               `json:"modules"`
	Dependencies  []DependencyFact           `json:"dependencies"`
	Language      LanguageFacts              `json:"language"`
	Testing       map[string]any             `json:"testing"`
	Verification  map[string]any             `json:"verification"`
	Resolution    map[string]any             `json:"resolution"`
	Notes         []string                   `json:"notes"`
	Extensions    map[string]json.RawMessage `json:"extensions,omitempty"`
}
type LineRange struct {
	Start int `json:"start"`
	Count int `json:"count"`
}
type LineChange struct {
	Before LineRange `json:"before"`
	After  LineRange `json:"after"`
}
type FileChange struct {
	Status     string       `json:"status"`
	BeforePath *string      `json:"beforePath"`
	AfterPath  *string      `json:"afterPath"`
	Lines      []LineChange `json:"lines"`
}
type DeclarationVersion struct {
	Signature string    `json:"signature"`
	Path      *string   `json:"path"`
	Lines     LineRange `json:"lines"`
	Source    string    `json:"source"`
}
type Change struct {
	Kind   string              `json:"kind"`
	Before *DeclarationVersion `json:"before"`
	After  *DeclarationVersion `json:"after"`
}
type Symbol struct {
	ID         string                     `json:"id"`
	Signature  string                     `json:"signature"`
	Path       string                     `json:"path"`
	Line       int                        `json:"line"`
	Role       string                     `json:"role"`
	Impact     string                     `json:"impact"`
	Module     *string                    `json:"module"`
	Extensions map[string]json.RawMessage `json:"extensions,omitempty"`
}
type Relation struct {
	Caller string `json:"caller"`
	Callee string `json:"callee"`
	Kind   string `json:"kind"`
}
type Boundary struct {
	Caller     *string `json:"caller"`
	Path       string  `json:"path"`
	Line       int     `json:"line"`
	Expression string  `json:"expression"`
	Reason     string  `json:"reason"`
}
type Snapshot struct {
	Nodes              []Symbol   `json:"nodes"`
	Edges              []Relation `json:"edges"`
	Boundaries         []Boundary `json:"boundaries"`
	TotalBoundaryCount int        `json:"totalBoundaryCount"`
	UnresolvedChanges  []string   `json:"unresolvedChanges"`
	Notes              []string   `json:"notes"`
}
type Execution struct {
	AfterSHA       string                     `json:"afterSha"`
	Status         string                     `json:"status"`
	Command        []string                   `json:"command"`
	ExitCode       *int                       `json:"exitCode"`
	DurationMillis int64                      `json:"durationMillis"`
	Log            *string                    `json:"log"`
	Notes          []string                   `json:"notes"`
	Extensions     map[string]json.RawMessage `json:"extensions,omitempty"`
}
type Artifact struct {
	Source   string `json:"source"`
	Retained string `json:"retained"`
	SHA256   string `json:"sha256"`
}
type Counter struct {
	Missed  int `json:"missed"`
	Covered int `json:"covered"`
}
type TestRoute struct {
	Test               string `json:"test"`
	Kind               string `json:"kind"`
	PassedClassRecords int    `json:"passedClassRecords"`
}
type SymbolEvidence struct {
	Symbol       string      `json:"symbol"`
	Signature    string      `json:"signature"`
	Path         string      `json:"path"`
	Line         int         `json:"line"`
	Role         string      `json:"role"`
	Impact       string      `json:"impact"`
	Status       string      `json:"status"`
	Reason       string      `json:"reason"`
	Instructions *Counter    `json:"instructions"`
	Branches     *Counter    `json:"branches"`
	Routes       []TestRoute `json:"routes"`
}
type DatasetMatch struct {
	Artifact               *Artifact        `json:"artifact"`
	Methods                []SymbolEvidence `json:"methods"`
	Counts                 map[string]int   `json:"counts"`
	Assessed               int              `json:"assessed"`
	WithHits               int              `json:"withHits"`
	MethodExecutionPercent *float64         `json:"methodExecutionPercent"`
}
type Matching struct {
	AfterSHA          string         `json:"afterSha"`
	Datasets          []DatasetMatch `json:"datasets"`
	Removed           []string       `json:"removed"`
	UnresolvedChanges []string       `json:"unresolvedChanges"`
	Notes             []string       `json:"notes"`
}
type Evidence struct {
	AfterSHA      string            `json:"afterSha"`
	CommandStatus string            `json:"commandStatus"`
	TestRecords   map[string]int    `json:"testRecords"`
	Tests         []json.RawMessage `json:"tests"`
	Coverage      []json.RawMessage `json:"coverage"`
	Artifacts     []Artifact        `json:"artifacts"`
	Manifest      *string           `json:"manifest"`
	Notes         []string          `json:"notes"`
}
type AnalyzeReport struct {
	SchemaVersion int                        `json:"schemaVersion"`
	GeneratedAt   string                     `json:"generatedAt"`
	Repository    string                     `json:"repository"`
	BeforeSHA     string                     `json:"beforeSha"`
	AfterSHA      string                     `json:"afterSha"`
	Files         []FileChange               `json:"files"`
	Changes       []Change                   `json:"changes"`
	Before        Snapshot                   `json:"before"`
	After         Snapshot                   `json:"after"`
	Execution     Execution                  `json:"execution"`
	Evidence      Evidence                   `json:"evidence"`
	Matching      Matching                   `json:"matching"`
	Notes         []string                   `json:"notes"`
	Extensions    map[string]json.RawMessage `json:"extensions,omitempty"`
}

func NewDocument(stack, operation string, report any) (Document, error) {
	data, err := json.Marshal(report)
	if err != nil {
		return Document{}, err
	}
	document := Document{Stack: stack, Operation: operation, Data: data}
	return document, document.Validate()
}
