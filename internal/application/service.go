// Package application coordinates public use cases without knowing CLI or Java protocol details.
package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/edwardnoaland/scryer/internal/repository"
)

type Service struct{ Analyzer contract.Analyzer }

func (s *Service) selectStack(stack string) error {
	if stack == "" {
		stack = "java"
	}
	if s.Analyzer == nil || stack != s.Analyzer.Stack() {
		supported := "none"
		if s.Analyzer != nil {
			supported = s.Analyzer.Stack()
		}
		return fmt.Errorf("unsupported stack %q; currently supported: %s", stack, supported)
	}
	return nil
}
func (s *Service) Scan(ctx context.Context, stack string, request contract.ScanRequest) (contract.Document, error) {
	if err := s.selectStack(stack); err != nil {
		return contract.Document{}, err
	}
	absolute, err := filepath.Abs(request.Repository)
	if err != nil {
		return contract.Document{}, err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return contract.Document{}, err
	}
	if !info.IsDir() {
		return contract.Document{}, fmt.Errorf("not a directory: %s", absolute)
	}
	request.Repository = absolute
	return s.Analyzer.Scan(ctx, request)
}
func (s *Service) Analyze(ctx context.Context, stack, directory string, request contract.AnalyzeRequest) (document contract.Document, err error) {
	if err := s.selectStack(stack); err != nil {
		return contract.Document{}, err
	}
	snapshots, err := repository.Open(ctx, directory, request.Before, request.After)
	if err != nil {
		return contract.Document{}, err
	}
	defer func() { err = errors.Join(err, snapshots.Close()) }()
	request.Repository = snapshots.Repository
	request.Before = snapshots.Before
	request.After = snapshots.After
	request.ComparisonRepository = snapshots.ComparisonRepository
	request.BeforeRoot = snapshots.BeforeRoot
	request.AfterRoot = snapshots.AfterRoot
	document, err = s.Analyzer.Analyze(ctx, request)
	if err != nil || snapshots.WorkingBase == "" {
		return document, err
	}
	var data map[string]json.RawMessage
	if err = json.Unmarshal(document.Data, &data); err != nil {
		return document, err
	}
	data["afterSource"], err = json.Marshal(map[string]string{
		"kind": "WORKING_TREE", "ref": ".", "baseHead": snapshots.WorkingBase, "snapshotSha": snapshots.After,
	})
	if err != nil {
		return document, err
	}
	var notes []string
	if err = json.Unmarshal(data["notes"], &notes); err != nil {
		return document, err
	}
	notes = append(notes, "After is an isolated working-tree snapshot (staged, unstaged and non-ignored untracked files); its SHA is a private snapshot commit, not a commit in the original repository. Ignored untracked files are excluded.")
	data["notes"], err = json.Marshal(notes)
	if err != nil {
		return document, err
	}
	document.Data, err = json.Marshal(data)
	return document, err
}
