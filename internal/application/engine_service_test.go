package application

import (
	"strings"
	"testing"
)

func Test_should_accept_mock_identity_when_engine_provides_one_complete_response(t *testing.T) {
	identity, err := decodeEngineIdentity([]byte(`{"protocolVersion":1,"id":"java","name":"Java mock","version":"dev","source":"example/java-engine","mock":true}`))
	if err != nil || identity.ID != "java" || identity.Source != "example/java-engine" {
		t.Fatalf("unexpected identity: %+v, %v", identity, err)
	}
}

func Test_should_reject_identity_when_engine_response_is_incomplete_or_not_mock_protocol(t *testing.T) {
	valid := `{"protocolVersion":1,"id":"java","name":"Java mock","version":"dev","source":"example/java-engine","mock":true}`
	for _, response := range []string{
		`not JSON`, `{}`, `null`, valid + `{}`, valid + `trailing text`,
		strings.Replace(valid, `"protocolVersion":1`, `"protocolVersion":2`, 1),
		strings.Replace(valid, `"id":"java"`, `"id":""`, 1),
	} {
		if _, err := decodeEngineIdentity([]byte(response)); err == nil {
			t.Fatalf("accepted invalid identity: %s", response)
		}
	}
}
