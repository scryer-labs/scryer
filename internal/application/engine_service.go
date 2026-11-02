package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/edwardnoaland/scryer/internal/process"
)

// EngineIdentity is the independent engine identity handshake.
type EngineIdentity struct {
	ProtocolVersion int    `json:"protocolVersion"`
	ID              string `json:"id"`
	Name            string `json:"name"`
	Version         string `json:"version"`
	Source          string `json:"source"`
	Mock            bool   `json:"mock"`
}

func DescribeEngine(ctx context.Context, executable string, diagnostics io.Writer) (EngineIdentity, string, error) {
	absolute, err := filepath.Abs(executable)
	if err != nil {
		return EngineIdentity{}, "", err
	}
	absolute, err = filepath.EvalSymlinks(absolute)
	if err != nil {
		return EngineIdentity{}, "", fmt.Errorf("engine executable: %w", err)
	}
	bounded, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	command := process.Command(bounded, filepath.Dir(absolute), absolute, "describe")
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = diagnostics
	if err := command.Run(); err != nil {
		if bounded.Err() != nil {
			return EngineIdentity{}, "", bounded.Err()
		}
		return EngineIdentity{}, "", fmt.Errorf("engine describe failed: %w", err)
	}
	identity, err := decodeEngineIdentity(output.Bytes())
	return identity, absolute, err
}

func decodeEngineIdentity(data []byte) (EngineIdentity, error) {
	if len(data) > 64*1024 {
		return EngineIdentity{}, fmt.Errorf("engine identity exceeds 64 KiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	var identity EngineIdentity
	if err := decoder.Decode(&identity); err != nil {
		return EngineIdentity{}, fmt.Errorf("invalid engine identity: %w", err)
	}
	if decoder.Decode(new(any)) != io.EOF {
		return EngineIdentity{}, fmt.Errorf("engine must return exactly one JSON identity")
	}
	if identity.ProtocolVersion != 1 {
		return EngineIdentity{}, fmt.Errorf("expected engine protocolVersion 1")
	}
	for _, value := range []string{identity.ID, identity.Name, identity.Version, identity.Source} {
		if strings.TrimSpace(value) == "" {
			return EngineIdentity{}, fmt.Errorf("engine identity requires id, name, version and source")
		}
	}
	return identity, nil
}
