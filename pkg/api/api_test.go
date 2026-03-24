package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateAndGetSandbox(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"id":"test-1"}`
	resp, err := http.Post(ts.URL+"/api/sandboxes", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", resp.StatusCode)
	}

	resp, err = http.Get(ts.URL + "/api/sandboxes/test-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d, want 200", resp.StatusCode)
	}
}

func TestGetNonexistentSandbox(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/sandboxes/nonexistent")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestValidatePolicy(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	valid := `{"name":"test","default_effect":"deny","rules":[{"name":"r1","effect":"allow","actions":["file"]}]}`
	resp, err := http.Post(ts.URL+"/api/policies/validate", "application/json", bytes.NewBufferString(valid))
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("valid policy status = %d, want 200", resp.StatusCode)
	}

	invalid := `{"name":"","default_effect":"deny"}`
	resp, err = http.Post(ts.URL+"/api/policies/validate", "application/json", bytes.NewBufferString(invalid))
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid policy status = %d, want 400", resp.StatusCode)
	}
}

func TestListSandboxes(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/sandboxes")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var sandboxes []json.RawMessage
	json.NewDecoder(resp.Body).Decode(&sandboxes)
	if len(sandboxes) != 0 {
		t.Errorf("expected 0 sandboxes, got %d", len(sandboxes))
	}
}
