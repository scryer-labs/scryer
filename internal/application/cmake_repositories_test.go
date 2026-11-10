package application

import (
	"encoding/json"
	"testing"
)

func TestIdentifyCMakeRepositoriesPreservesDeclarationsAndMatchesSSHIdentity(t *testing.T) {
	payload := json.RawMessage(`{"includes":[{"target":"<x>"}],"cmake":{"future":true,"externalDependencies":[
		{"name":"felt","command":"fetchcontent_declare","file":"CMakeLists.txt","line":7,"arguments":["GIT_REPOSITORY","https://github.com/ScriptoriumLab/scriptorium-felt.git","GIT_TAG","v1"]},
		{"name":"local","command":"externalproject_add","arguments":["GIT_REPOSITORY","../local"]},
		{"name":"dynamic","command":"fetchcontent_declare","arguments":["GIT_REPOSITORY","${DEP_URL}"]}
	]}}`)
	enriched, err := identifyCMakeRepositories(payload)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Includes []map[string]string
		CMake    struct {
			Future               bool
			ExternalDependencies []struct {
				File       string
				Line       int
				Arguments  []string
				Repository *struct{ ID string }
			}
		}
	}
	if err := json.Unmarshal(enriched, &result); err != nil {
		t.Fatal(err)
	}
	declarations := result.CMake.ExternalDependencies
	if len(result.Includes) != 1 || !result.CMake.Future || len(declarations) != 3 ||
		declarations[0].File != "CMakeLists.txt" || declarations[0].Line != 7 ||
		declarations[0].Arguments[3] != "v1" || declarations[0].Repository == nil ||
		declarations[0].Repository.ID != "github.com/scriptoriumlab/scriptorium-felt" ||
		declarations[1].Repository != nil || declarations[2].Repository != nil {
		t.Fatalf("unexpected enriched facts: %s", enriched)
	}
}
