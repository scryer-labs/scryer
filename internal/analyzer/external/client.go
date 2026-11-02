// Package external adapts independent engines to the common analyzer contract.
package external

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/edwardnoaland/scryer/internal/process"
)

type Client struct {
	Executable string
	EngineID   string
	Progress   io.Writer
}

func New(engineID string, progress io.Writer) *Client {
	return &Client{EngineID: engineID, Progress: progress}
}
func (c *Client) Stack() string { return c.EngineID }
func (c *Client) Scan(ctx context.Context, r contract.ScanRequest) (contract.Document, error) {
	return c.call(ctx, contract.Request{Operation: "scan", Scan: &r})
}
func (c *Client) Analyze(ctx context.Context, r contract.AnalyzeRequest) (contract.Document, error) {
	return c.call(ctx, contract.Request{Operation: "analyze", Analyze: &r})
}
func (c *Client) call(ctx context.Context, request contract.Request) (contract.Document, error) {
	request.ProtocolVersion = contract.ProtocolVersion
	request.Stack = c.Stack()
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return contract.Document{}, err
	}
	request.RequestID = hex.EncodeToString(id)
	data, err := json.Marshal(request)
	if err != nil {
		return contract.Document{}, err
	}
	executable, err := c.executable()
	if err != nil {
		return contract.Document{}, err
	}
	cmd := process.Command(ctx, "", executable, "execute")
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stderr = c.Progress
	var output bytes.Buffer
	cmd.Stdout = &output
	err = cmd.Run()
	if ctx.Err() != nil {
		return contract.Document{}, ctx.Err()
	}
	if err != nil {
		return contract.Document{}, fmt.Errorf("engine execution failed: %w", err)
	}
	decoder := json.NewDecoder(&output)
	var response contract.Response
	if err := decoder.Decode(&response); err != nil {
		return contract.Document{}, fmt.Errorf("invalid analyzer response: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return contract.Document{}, fmt.Errorf("analyzer emitted more than one response")
	}
	if response.ProtocolVersion != contract.ProtocolVersion || response.RequestID != request.RequestID || response.Stack != request.Stack || response.Operation != request.Operation {
		return contract.Document{}, fmt.Errorf("analyzer response protocol/request identity mismatch")
	}
	if response.Error != nil {
		return contract.Document{}, fmt.Errorf("%s: %s", response.Error.Code, response.Error.Message)
	}
	document := contract.Document{Stack: c.Stack(), Operation: request.Operation, Data: response.Report}
	if err := document.Validate(); err != nil {
		return contract.Document{}, err
	}
	if request.Analyze != nil {
		var snapshots struct {
			Before string `json:"beforeSha"`
			After  string `json:"afterSha"`
		}
		if err := json.Unmarshal(document.Data, &snapshots); err != nil {
			return contract.Document{}, err
		}
		if snapshots.Before != request.Analyze.Before || snapshots.After != request.Analyze.After {
			return contract.Document{}, fmt.Errorf("analyzer report snapshot identity mismatch")
		}
	}
	return document, nil
}

var _ contract.Analyzer = (*Client)(nil)

// Explicit development overrides and PATH discovery do not persist an installation.
func (c *Client) executable() (string, error) {
	executable := c.Executable
	if executable == "" {
		key := "SCRYER_ENGINE_" + strings.ToUpper(strings.ReplaceAll(c.EngineID, "-", "_"))
		executable = os.Getenv(key)
	}
	if executable == "" {
		executable = "scryer-" + c.EngineID + "-engine"
	}
	resolved, err := exec.LookPath(executable)
	if err != nil {
		return "", fmt.Errorf("engine %q is unavailable: provide %s on PATH or set SCRYER_ENGINE_%s: %w", c.EngineID, "scryer-"+c.EngineID+"-engine", strings.ToUpper(strings.ReplaceAll(c.EngineID, "-", "_")), err)
	}
	return filepath.Abs(resolved)
}
