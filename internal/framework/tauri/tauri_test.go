package tauri

import (
	"encoding/json"
	"testing"

	"github.com/edwardnoaland/scryer/internal/contract"
)

func TestAggregateMatchesOnlySameAppWithCompleteEvidence(t *testing.T) {
	outcome := func(engine string, apps []contract.TauriApp) contract.FactOutcome {
		t.Helper()
		data, err := json.Marshal(map[string]contract.TauriIPC{"tauri": {Apps: apps}})
		if err != nil {
			t.Fatal(err)
		}
		return contract.FactOutcome{Fact: "ipc", Engine: engine, Status: "collected", Data: data}
	}
	endpoint := func(kind, role, name, file, symbol string, line uint) contract.TauriEndpoint {
		return contract.TauriEndpoint{Kind: kind, Role: role, Name: name, File: file, Symbol: symbol, Line: line, Column: 1}
	}
	const config = "src-tauri/tauri.conf.json"
	invoke := endpoint("command", "invoke", "select", "src/page.svelte", "", 8)
	listen := endpoint("event", "listen", "update", "src/page.svelte", "", 10)
	declaration := endpoint("command", "declaration", "select", "src-tauri/src/lib.rs", "select", 1)
	registration := endpoint("command", "registration", "select", "src-tauri/src/lib.rs", "select", 8)
	emit := endpoint("event", "emit", "update", "src-tauri/src/lib.rs", "", 18)
	frontend := outcome("ts-js", []contract.TauriApp{
		{Config: config, Endpoints: []contract.TauriEndpoint{invoke, listen, listen}},
		{Config: "other/src-tauri/tauri.conf.json", Endpoints: []contract.TauriEndpoint{invoke, listen}},
	})
	partial := Aggregate([]contract.FactOutcome{frontend})
	if len(partial.Apps) != 2 || len(partial.Apps[0].Endpoints) != 2 || len(partial.Apps[0].Links) != 0 {
		t.Fatalf("unexpected partial report: %+v", partial)
	}
	backend := outcome("rust", []contract.TauriApp{{Config: config, Endpoints: []contract.TauriEndpoint{declaration, registration, emit}}})
	result := Aggregate([]contract.FactOutcome{frontend, backend})
	links := result.Apps[0].Links
	if len(links) != 2 || len(result.Apps[1].Links) != 0 {
		t.Fatalf("unexpected matches: %+v", result)
	}
	if links[0].Kind != "command" || links[0].From.Engine != "ts-js" || links[0].To.Engine != "rust" || links[0].Registration == nil {
		t.Fatalf("incorrect command evidence: %+v", links[0])
	}
	if links[1].Kind != "event" || links[1].From.Engine != "rust" || links[1].To.Engine != "ts-js" {
		t.Fatalf("incorrect event direction: %+v", links[1])
	}

	other := declaration
	other.File = "src-tauri/src/other.rs"
	ambiguous := outcome("rust", []contract.TauriApp{{Config: config, Endpoints: []contract.TauriEndpoint{declaration, other, registration}}})
	if links := Aggregate([]contract.FactOutcome{frontend, ambiguous}).Apps[0].Links; len(links) != 0 {
		t.Fatalf("ambiguous declarations produced links: %+v", links)
	}
}
