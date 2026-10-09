// Package java adapts the JVM worker to the common analyzer contract.
package java

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/edwardnoaland/scryer/internal/process"
)

type Client struct {
	Classpath string
	Java      string
	Progress  io.Writer
}

func New(progress io.Writer) *Client { return &Client{Progress: progress} }
func (c *Client) Stack() string      { return "java" }
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
	classpath := c.Classpath
	if classpath == "" {
		classpath = os.Getenv("SCRYER_JAVA_CLASSPATH")
	}
	if classpath == "" {
		executable, err := os.Executable()
		if err != nil {
			return contract.Document{}, err
		}
		executable, err = filepath.EvalSymlinks(executable)
		if err != nil {
			return contract.Document{}, err
		}
		directory := filepath.Join(filepath.Dir(executable), "..", "lib", "java")
		jars, _ := filepath.Glob(filepath.Join(directory, "*.jar"))
		if len(jars) == 0 {
			return contract.Document{}, fmt.Errorf("Java analyzer not installed: run scripts/build or set SCRYER_JAVA_CLASSPATH")
		}
		classpath = strings.Join(jars, string(os.PathListSeparator))
	}
	executable := c.Java
	if executable == "" {
		executable = "java"
		home := os.Getenv("SCRYER_ANALYZER_JAVA_HOME")
		if home == "" {
			home = os.Getenv("JAVA_HOME")
		}
		if home != "" {
			executable = filepath.Join(home, "bin", "java")
		}
	}
	// Target build/test timeouts remain stack-specific; the caller owns cancellation of the complete run.
	cmd := process.Command(ctx, "", executable, "-cp", classpath, "com.edwardnoaland.scryer.worker.WorkerKt")
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stderr = c.Progress
	var output bytes.Buffer
	cmd.Stdout = &output
	err = cmd.Run()
	if ctx.Err() != nil {
		return contract.Document{}, ctx.Err()
	}
	if err != nil {
		return contract.Document{}, fmt.Errorf("Java analyzer failed: %w (analyzer requires Java 21; set SCRYER_ANALYZER_JAVA_HOME)", err)
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
