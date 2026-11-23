package contract

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
)

const ImpactOperation = "impact"

type ImpactCollector interface {
	CollectImpact(context.Context, AnalyzeRequest) (Document, error)
}

type ImpactPosition struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type ImpactRange struct {
	File  string         `json:"file"`
	Start ImpactPosition `json:"start"`
	End   ImpactPosition `json:"end"`
}

type ImpactSymbol struct {
	Role          string          `json:"role,omitempty"`
	TestContext   json.RawMessage `json:"testContext,omitempty"`
	ExecutionKind string          `json:"executionKind,omitempty"`
	ID            string          `json:"id"`
	Key           string          `json:"key"`
	Signature     string          `json:"signature"`
	Fingerprint   string          `json:"fingerprint"`
	Source        string          `json:"source"`
	Location      ImpactRange     `json:"location"`
}

type ImpactCall struct {
	Kind           string      `json:"kind,omitempty"`
	Target         string      `json:"targetSymbolId,omitempty"`
	ID             string      `json:"callSiteId"`
	Caller         string      `json:"callerSymbolId"`
	Candidates     []string    `json:"candidateSymbolIds,omitempty"`
	CandidateBasis string      `json:"candidateBasis,omitempty"`
	Candidate      string      `json:"candidateSymbolId"`
	Callee         string      `json:"callee"`
	Resolution     string      `json:"resolution"`
	Site           ImpactRange `json:"site"`
}

type ImpactContext struct {
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
}

type ImpactSnapshot struct {
	Selection  *FileSelectionReceipt `json:"selection,omitempty"`
	Resolution json.RawMessage       `json:"resolution,omitempty"`
	Contexts   []ImpactContext       `json:"contexts"`
	Symbols    []ImpactSymbol        `json:"symbols"`
	Calls      []ImpactCall          `json:"calls"`
	Notes      []string              `json:"notes"`
}

type ImpactFacts struct {
	SchemaVersion int            `json:"schemaVersion"`
	BeforeSHA     string         `json:"beforeSha"`
	AfterSHA      string         `json:"afterSha"`
	Before        ImpactSnapshot `json:"before"`
	After         ImpactSnapshot `json:"after"`
}

func validateImpactSnapshot(raw json.RawMessage) error {
	var snapshot ImpactSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return err
	}
	ids := map[string]bool{}
	for _, symbol := range snapshot.Symbols {
		if symbol.ID == "" || symbol.Key == "" || symbol.Signature == "" || len(symbol.Fingerprint) != 64 || ids[symbol.ID] {
			return fmt.Errorf("invalid or duplicate impact symbol %q", symbol.ID)
		}
		if _, err := hex.DecodeString(symbol.Fingerprint); err != nil {
			return fmt.Errorf("invalid symbol fingerprint")
		}
		if err := validateImpactRange(symbol.Location); err != nil {
			return err
		}
		ids[symbol.ID] = true
	}
	contextPaths := map[string]bool{}
	for _, item := range snapshot.Contexts {
		if !filepath.IsLocal(item.Path) || contextPaths[item.Path] || len(item.Fingerprint) != 64 {
			return fmt.Errorf("invalid source context %q", item.Path)
		}
		if _, err := hex.DecodeString(item.Fingerprint); err != nil {
			return fmt.Errorf("invalid source context fingerprint")
		}
		contextPaths[item.Path] = true
	}
	calls := map[string]bool{}
	for _, call := range snapshot.Calls {
		if call.ID == "" || calls[call.ID] || call.Resolution == "" {
			return fmt.Errorf("invalid or duplicate impact call %q", call.ID)
		}
		if err := validateImpactRange(call.Site); err != nil {
			return err
		}
		calls[call.ID] = true
	}
	return nil
}

func validateImpactRange(location ImpactRange) error {
	if !filepath.IsLocal(location.File) || location.Start.Line < 1 || location.Start.Column < 1 ||
		location.End.Line < location.Start.Line || location.End.Column < 1 ||
		location.End.Line == location.Start.Line && location.End.Column < location.Start.Column {
		return fmt.Errorf("invalid impact source location")
	}
	return nil
}
