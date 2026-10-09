package contract

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDocumentRejectsIncompatibleOrMissingFacts(t *testing.T) {
	good := `{"schemaVersion":1,"root":"/repo","builds":[],"dependencies":[],"modules":[],"language":{},"resolution":{},"testing":{},"verification":{},"futureField":{"value":null}}`
	document := Document{Operation: "scan", Data: json.RawMessage(good)}
	if err := document.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{strings.Replace(good, `"schemaVersion":1`, `"schemaVersion":2`, 1), strings.Replace(good, `"builds":[]`, `"builds":null`, 1), strings.Replace(good, `"root":"/repo"`, `"root":3`, 1)} {
		if err := (Document{Operation: "scan", Data: json.RawMessage(bad)}).Validate(); err == nil {
			t.Fatalf("accepted malformed contract %s", bad)
		}
	}
}

func TestSnapshotRejectsDanglingEdgesAndDuplicateSymbols(t *testing.T) {
	valid := `{"nodes":[{"id":"a","signature":"C#f()","role":"PRODUCTION","impact":"CHANGED"}],"edges":[],"boundaries":[]}`
	if err := validateSnapshot(json.RawMessage(valid)); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{strings.Replace(valid, `"edges":[]`, `"edges":[{"caller":"a","callee":"missing","kind":"DIRECT"}]`, 1), strings.Replace(valid, `"nodes":[`, `"nodes":[{"id":"a","signature":"C#f()","role":"TEST","impact":"CALLER"},`, 1), strings.Replace(valid, `"role":"PRODUCTION"`, `"role":"invented"`, 1)} {
		if err := validateSnapshot(json.RawMessage(bad)); err == nil {
			t.Fatalf("accepted invalid graph: %s", bad)
		}
	}
}
