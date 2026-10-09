package contract

import (
	"encoding/json"
	"fmt"
)

// Validate rejects incompatible documents rather than presenting missing fields as successful analysis.
func (d Document) Validate() error {
	var report map[string]json.RawMessage
	if err := json.Unmarshal(d.Data, &report); err != nil {
		return fmt.Errorf("invalid report JSON: %w", err)
	}
	var version int
	if err := json.Unmarshal(report["schemaVersion"], &version); err != nil || version != ReportVersion {
		return fmt.Errorf("unsupported report schemaVersion (expected %d)", ReportVersion)
	}
	required := map[string]string{}
	switch d.Operation {
	case "scan":
		required = map[string]string{"root": "string", "builds": "array", "dependencies": "array", "modules": "array", "language": "object", "resolution": "object", "testing": "object", "verification": "object"}
	case "analyze":
		required = map[string]string{"repository": "string", "beforeSha": "string", "afterSha": "string", "changes": "array", "files": "array", "before": "object", "after": "object", "execution": "object", "evidence": "object", "matching": "object", "notes": "array"}
	default:
		return fmt.Errorf("unsupported operation %q", d.Operation)
	}
	for name, kind := range required {
		var value any
		if err := json.Unmarshal(report[name], &value); err != nil {
			return fmt.Errorf("missing/invalid report field %s", name)
		}
		valid := false
		switch kind {
		case "string":
			_, valid = value.(string)
		case "array":
			_, valid = value.([]any)
		case "object":
			_, valid = value.(map[string]any)
		}
		if !valid {
			return fmt.Errorf("invalid report field %s: expected %s", name, kind)
		}
	}
	if d.Operation == "analyze" {
		for name, fields := range map[string]map[string]string{
			"evidence":  {"tests": "array", "coverage": "array", "artifacts": "array", "notes": "array"},
			"matching":  {"datasets": "array", "removed": "array", "unresolvedChanges": "array", "notes": "array"},
			"execution": {"command": "array", "notes": "array"},
		} {
			if err := validateFields(report[name], fields); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
		for _, name := range []string{"before", "after"} {
			if err := validateSnapshot(report[name]); err != nil {
				return fmt.Errorf("%s snapshot: %w", name, err)
			}
		}
		var execution struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(report["execution"], &execution); err != nil {
			return err
		}
		switch execution.Status {
		case "SUCCEEDED", "FAILED", "SKIPPED", "UNAVAILABLE", "TIMED_OUT", "SNAPSHOT_CHANGED":
		default:
			return fmt.Errorf("invalid execution status %q", execution.Status)
		}
	}
	return nil
}

// Snapshot identities are opaque across stacks; graph structure and execution uncertainty are common.
func validateSnapshot(raw json.RawMessage) error {
	var snapshot struct {
		Nodes []struct {
			ID        string `json:"id"`
			Signature string `json:"signature"`
			Role      string `json:"role"`
			Impact    string `json:"impact"`
		} `json:"nodes"`
		Edges []struct {
			Caller string `json:"caller"`
			Callee string `json:"callee"`
			Kind   string `json:"kind"`
		} `json:"edges"`
		Boundaries json.RawMessage `json:"boundaries"`
	}
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return err
	}
	if snapshot.Nodes == nil || snapshot.Edges == nil || len(snapshot.Boundaries) == 0 || snapshot.Boundaries[0] != '[' {
		return fmt.Errorf("snapshot nodes/edges/boundaries must be arrays")
	}
	ids := map[string]bool{}
	for _, node := range snapshot.Nodes {
		if node.ID == "" || node.Signature == "" || node.Impact == "" {
			return fmt.Errorf("missing symbol identity/signature/impact")
		}
		if ids[node.ID] {
			return fmt.Errorf("duplicate symbol identity %s", node.ID)
		}
		if node.Role != "PRODUCTION" && node.Role != "TEST" && node.Role != "UNKNOWN" {
			return fmt.Errorf("invalid source role %q", node.Role)
		}
		ids[node.ID] = true
	}
	for _, edge := range snapshot.Edges {
		if !ids[edge.Caller] || !ids[edge.Callee] || edge.Kind == "" {
			return fmt.Errorf("edge references missing symbols or kind")
		}
	}
	return nil
}

func validateFields(raw json.RawMessage, required map[string]string) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	for name, kind := range required {
		value, ok := fields[name]
		if !ok || len(value) == 0 {
			return fmt.Errorf("missing field %s", name)
		}
		var parsed any
		if err := json.Unmarshal(value, &parsed); err != nil {
			return err
		}
		valid := false
		switch kind {
		case "array":
			_, valid = parsed.([]any)
		case "object":
			_, valid = parsed.(map[string]any)
		}
		if !valid {
			return fmt.Errorf("%s must be %s", name, kind)
		}
	}
	return nil
}
