// Package contract defines the language-neutral analyzer boundary. It has no CLI or process dependencies.
package contract

import (
	"context"
	"encoding/json"
)

const ProtocolVersion = 1
const ReportVersion = 1

type ScanRequest struct {
	Repository string `json:"repository"`
	Static     bool   `json:"static"`
	RemoteList bool   `json:"remoteList"`
	BuildTool  string `json:"buildTool,omitempty"`
}
type AnalyzeRequest struct {
	Repository  string `json:"repository"`
	Before      string `json:"before"`
	After       string `json:"after"`
	BeforeRoot  string `json:"beforeRoot"`
	AfterRoot   string `json:"afterRoot"`
	SkipTests   bool   `json:"skipTests"`
	TestCommand string `json:"testCommand,omitempty"`
	BuildTool   string `json:"buildTool,omitempty"`
}

// Analyzer implementations supply facts; presentation is owned by Go.
type Analyzer interface {
	Stack() string
	Scan(context.Context, ScanRequest) (Document, error)
	Analyze(context.Context, AnalyzeRequest) (Document, error)
}

// Document retains all report fields, including unknown values and stack-specific provenance.
// Required v1 shapes are checked at the adapter boundary; JSON exports preserve the supplied facts.
type Document struct {
	Stack     string
	Operation string
	Data      json.RawMessage
}
type Request struct {
	ProtocolVersion int             `json:"protocolVersion"`
	RequestID       string          `json:"requestId"`
	Stack           string          `json:"stack"`
	Operation       string          `json:"operation"`
	Scan            *ScanRequest    `json:"scan,omitempty"`
	Analyze         *AnalyzeRequest `json:"analyze,omitempty"`
}
type Response struct {
	ProtocolVersion int             `json:"protocolVersion"`
	RequestID       string          `json:"requestId"`
	Stack           string          `json:"stack"`
	Operation       string          `json:"operation"`
	Report          json.RawMessage `json:"report"`
	Error           *Failure        `json:"error"`
}
type Failure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
