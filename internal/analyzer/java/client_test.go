package java

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/edwardnoaland/scryer/internal/contract"
)

func TestMain(m *testing.M) {
	if mode := os.Getenv("SCRYER_FAKE_WORKER"); mode != "" {
		if mode == "wait" {
			time.Sleep(30 * time.Second)
			os.Exit(0)
		}
		if mode == "exit" {
			os.Exit(7)
		}
		var request contract.Request
		_ = json.NewDecoder(os.Stdin).Decode(&request)
		fmt.Fprintln(os.Stderr, "worker progress")
		response := contract.Response{ProtocolVersion: 1, RequestID: request.RequestID, Stack: "java", Operation: request.Operation, Report: json.RawMessage(`{"schemaVersion":1,"root":"/repo","builds":[],"dependencies":[],"modules":[],"language":{},"resolution":{},"testing":{},"verification":{},"futureField":null}`)}
		switch mode {
		case "version":
			response.ProtocolVersion = 2
		case "id":
			response.RequestID = "different"
		case "error":
			response.Error = &contract.Failure{Code: "STACK_ERROR", Message: "reason"}
		case "invalid":
			response.Report = json.RawMessage(`{"schemaVersion":1}`)
		}
		_ = json.NewEncoder(os.Stdout).Encode(response)
		if mode == "trailing" {
			fmt.Fprintln(os.Stdout, `{"extra":true}`)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}
func fakeClient(t *testing.T, mode string) (*Client, *bytes.Buffer) {
	t.Helper()
	t.Setenv("SCRYER_FAKE_WORKER", mode)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	progress := &bytes.Buffer{}
	return &Client{Java: exe, Classpath: "unused", Progress: progress}, progress
}
func TestProtocolAndProgressAreSeparated(t *testing.T) {
	client, progress := fakeClient(t, "good")
	document, err := client.Scan(context.Background(), contract.ScanRequest{Repository: "/repo"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(document.Data), "worker progress") || !strings.Contains(progress.String(), "worker progress") {
		t.Fatal("progress contaminated protocol")
	}
	if !strings.Contains(string(document.Data), `"futureField":null`) {
		t.Fatal("lost extension/null facts")
	}
}
func TestInvalidResponsesAndWorkerFailuresAreRejected(t *testing.T) {
	for _, mode := range []string{"version", "id", "error", "invalid", "trailing", "exit"} {
		t.Run(mode, func(t *testing.T) {
			client, _ := fakeClient(t, mode)
			if _, err := client.Scan(context.Background(), contract.ScanRequest{Repository: "/repo"}); err == nil {
				t.Fatal("accepted invalid response")
			}
		})
	}
}
func TestCancellationStopsWorker(t *testing.T) {
	client, _ := fakeClient(t, "wait")
	client.Progress = io.Discard
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	if _, err := client.Scan(ctx, contract.ScanRequest{Repository: "/repo"}); err == nil {
		t.Fatal("ignored cancellation")
	}
	if time.Since(started) > 5*time.Second {
		t.Fatal("worker did not stop")
	}
}
