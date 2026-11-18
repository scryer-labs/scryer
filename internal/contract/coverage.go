package contract

import (
	"context"
	"encoding/json"
)

const CoverageOperation = "coverage"

type CoverageRequest struct {
	Repository   string   `json:"repository"`
	Artifact     string   `json:"artifact,omitempty"`
	TestBinaries []string `json:"testBinaries,omitempty"`
	TestTargets  []string `json:"testTargets,omitempty"`
}

type CoverageCollector interface {
	CollectCoverage(context.Context, CoverageRequest) (Document, error)
}

type CoverageReport struct {
	SchemaVersion int             `json:"schemaVersion"`
	Root          string          `json:"root"`
	Engine        string          `json:"engine,omitempty"`
	Status        string          `json:"status"`
	Message       string          `json:"message,omitempty"`
	Runs          []CoverageRun   `json:"runs,omitempty"`
	Coverage      json.RawMessage `json:"coverage,omitempty"`
}

type CoverageRun struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Profiles int    `json:"profiles"`
	Message  string `json:"message,omitempty"`
}

type CoverageEvidence struct {
	Format      string         `json:"format"`
	Version     string         `json:"version"`
	Artifact    string         `json:"artifact"`
	SHA256      string         `json:"sha256"`
	SourceMatch string         `json:"sourceMatch"`
	SourceSHA   string         `json:"sourceSha,omitempty"`
	Granularity string         `json:"granularity"`
	Files       []CoverageFile `json:"files"`
	Notes       []string       `json:"notes"`
}

type CoverageCount struct {
	Covered uint64 `json:"covered"`
	Count   uint64 `json:"count"`
}

type CoverageRegion struct {
	Start ImpactPosition `json:"start"`
	End   ImpactPosition `json:"end"`
	Count uint64         `json:"count"`
}

type CoverageFile struct {
	Regions []CoverageRegion `json:"regions,omitempty"`
	Path    string           `json:"path"`
	Summary struct {
		Lines     *CoverageCount `json:"lines"`
		Functions *CoverageCount `json:"functions"`
		Regions   *CoverageCount `json:"regions"`
	} `json:"summary"`
	LLVMSegments json.RawMessage `json:"llvmSegments,omitempty"`
}
