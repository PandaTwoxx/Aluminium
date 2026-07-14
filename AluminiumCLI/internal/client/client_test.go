package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGenerateTokenSendsEmptyScopesArrayWhenNil(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"abc123"}`))
	}))
	defer server.Close()

	client := NewAPIClient()
	_, err := client.GenerateToken(server.URL, "user", "pass", nil)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	scopes, ok := got["scopes"].([]any)
	if !ok {
		t.Fatalf("expected scopes to be an array, got %#v", got["scopes"])
	}
	if len(scopes) != 0 {
		t.Fatalf("expected empty scopes array, got %v", scopes)
	}
}
