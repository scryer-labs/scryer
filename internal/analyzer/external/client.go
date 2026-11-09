// Package external adapts independent engines to the common analyzer contract.
package external

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/edwardnoaland/scryer/internal/engine"
	"github.com/edwardnoaland/scryer/internal/engineabi"
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
	executable, err := c.executable(ctx)
	if err != nil {
		return contract.Document{}, err
	}
	var output bytes.Buffer
	if engineabi.IsLibrary(executable) {
		var response []byte
		response, err = engineabi.Execute(ctx, executable, request.RequestID, data)
		output.Write(response)
	} else {
		cmd := process.Command(ctx, "", executable, "execute")
		cmd.Stdin = bytes.NewReader(data)
		cmd.Stderr = c.Progress
		cmd.Stdout = &output
		err = cmd.Run()
	}

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

// Explicit overrides win, followed by the installed package and development PATH discovery.
func (c *Client) executable(ctx context.Context) (string, error) {
	executable := c.Executable
	if executable == "" {
		key := "SCRYER_ENGINE_" + strings.ToUpper(strings.ReplaceAll(c.EngineID, "-", "_"))
		executable = os.Getenv(key)
	}
	if executable == "" {
		store, err := engine.DefaultStore()
		if err != nil {
			return "", err
		}
		installed, err := store.Find(ctx, c.EngineID)
		if err == nil {
			if installed.Library != "" {
				return installed.Library, nil
			}
			return installed.Executable, nil
		}
		if !errors.Is(err, engine.ErrNotInstalled) {
			return "", fmt.Errorf("installed engine %q: %w", c.EngineID, err)
		}
		executable = "scryer-" + c.EngineID + "-engine"
	}
	if engineabi.IsLibrary(executable) {
		return filepath.Abs(executable)
	}
	resolved, err := exec.LookPath(executable)
	if err != nil {
		return "", fmt.Errorf("engine %q is unavailable: install a local package, provide %s on PATH or set SCRYER_ENGINE_%s: %w", c.EngineID, "scryer-"+c.EngineID+"-engine", strings.ToUpper(strings.ReplaceAll(c.EngineID, "-", "_")), err)
	}
	return filepath.Abs(resolved)
}
