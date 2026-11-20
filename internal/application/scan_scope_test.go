package application

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/edwardnoaland/scryer/internal/engine"
)

type scopeAnalyzer struct {
	receipt *contract.FileSelectionReceipt
}

func (a scopeAnalyzer) Stack() string { return "cpp" }
func (a scopeAnalyzer) Scan(_ context.Context, request contract.ScanRequest) (contract.Document, error) {
	data, err := json.Marshal(map[string]any{"schemaVersion": 1, "root": request.Repository, "selection": a.receipt, "facts": map[string]any{"files": map[string]any{"files": []any{}}}})
	return contract.Document{Stack: "cpp", Operation: "scan", Data: data}, err
}
func (a scopeAnalyzer) Analyze(context.Context, contract.AnalyzeRequest) (contract.Document, error) {
	panic("not used")
}
func TestScanRejectsMissingAndMismatchedSelectionReceipts(t *testing.T) {
	reference := &contract.FileSelectionRef{SHA256: "expected", FileCount: 2}
	for _, receipt := range []*contract.FileSelectionReceipt{nil, {SHA256: "wrong", FileCount: 2}, {SHA256: "expected", FileCount: 1}, {SHA256: "expected", FileCount: 2}} {
		service := Service{NewInstalledAnalyzer: func(engine.Installation) contract.Analyzer { return scopeAnalyzer{receipt} }}
		installation := engine.Installation{Capabilities: &engine.Capabilities{}}
		outcomes := service.scanSelected(context.Background(), installation, t.TempDir(), []string{"files"}, nil, "", reference)
		want := "failed"
		if receipt != nil && receipt.SHA256 == reference.SHA256 && receipt.FileCount == reference.FileCount {
			want = "collected"
		}
		if len(outcomes) != 1 || outcomes[0].Status != want {
			t.Fatalf("receipt %+v: %+v", receipt, outcomes)
		}
	}
}
