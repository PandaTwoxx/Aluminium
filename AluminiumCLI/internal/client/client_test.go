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

func TestDeleteUserSendsExpectedPayload(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"User deleted successfully."}`))
	}))
	defer server.Close()

	client := NewAPIClient()
	err := client.DeleteUser(server.URL, "alice", "hunter2")
	if err != nil {
		t.Fatalf("DeleteUser returned error: %v", err)
	}

	if got["username"] != "alice" {
		t.Fatalf("expected username alice, got %#v", got["username"])
	}
	if got["password"] != "hunter2" {
		t.Fatalf("expected password hunter2, got %#v", got["password"])
	}
}
