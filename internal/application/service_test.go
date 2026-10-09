package application

import (
	"context"
	"testing"

	"github.com/edwardnoaland/scryer/internal/contract"
)

// A future native analyzer can implement the same interface without importing Java or command packages.
type nativeAnalyzer struct{}

func (nativeAnalyzer) Stack() string { return "cpp" }
func (nativeAnalyzer) Scan(_ context.Context, request contract.ScanRequest) (contract.Document, error) {
	return contract.NewDocument("cpp", "scan", contract.ScanReport{SchemaVersion: 1, Root: request.Repository, Builds: []contract.BuildFact{}, Modules: []contract.ModuleFact{}, Dependencies: []contract.DependencyFact{}, Language: contract.LanguageFacts{}, Testing: map[string]any{}, Verification: map[string]any{}, Resolution: map[string]any{}, Notes: []string{"fixture analyzer, not a C++ implementation"}})
}
func (nativeAnalyzer) Analyze(context.Context, contract.AnalyzeRequest) (contract.Document, error) {
	panic("not called")
}
func TestNativeStackCanSupplyContractWithoutJVM(t *testing.T) {
	service := Service{Analyzer: nativeAnalyzer{}}
	document, err := service.Scan(context.Background(), "cpp", contract.ScanRequest{Repository: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if document.Stack != "cpp" {
		t.Fatal("lost stack")
	}
	if _, err := service.Scan(context.Background(), "java", contract.ScanRequest{Repository: t.TempDir()}); err == nil {
		t.Fatal("incorrect analyzer selected")
	}
}
