package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestDashboardStatsEmpty(t *testing.T) {
	s := newTestServer()
	w := doRequest(s, "GET", "/api/v1/dashboard/stats", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var stats DashboardStatsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if stats.ActiveSandboxes != 0 {
		t.Errorf("expected 0 active sandboxes, got %d", stats.ActiveSandboxes)
	}
	if stats.TotalActions != 0 {
		t.Errorf("expected 0 total actions, got %d", stats.TotalActions)
	}
	if stats.DeniedActions != 0 {
		t.Errorf("expected 0 denied actions, got %d", stats.DeniedActions)
	}
	if stats.AvgResponseMs != 0 {
		t.Errorf("expected 0 avg response ms, got %f", stats.AvgResponseMs)
	}
}

func TestDashboardStatsActiveSandboxes(t *testing.T) {
	s := newTestServer()

	// Create and start two sandboxes
	r1 := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "a"})
	id1 := parseBody(r1)["id"].(string)
	doRequest(s, "POST", "/api/v1/sandboxes/"+id1+"/start", nil)

	r2 := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "b"})
	id2 := parseBody(r2)["id"].(string)
	doRequest(s, "POST", "/api/v1/sandboxes/"+id2+"/start", nil)

	// Third sandbox not started
	doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "c"})

	w := doRequest(s, "GET", "/api/v1/dashboard/stats", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var stats DashboardStatsResponse
	json.Unmarshal(w.Body.Bytes(), &stats)

	if stats.ActiveSandboxes != 2 {
		t.Errorf("expected 2 active sandboxes, got %d", stats.ActiveSandboxes)
	}
}

func TestDashboardStatsCountsActions(t *testing.T) {
	s := newTestServer()

	r := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "action-test"})
	id := parseBody(r)["id"].(string)
	doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/start", nil)

	// Load allow-all policy and execute an action
	s.mu.RLock()
	entry := s.sandboxes[id]
	s.mu.RUnlock()
	entry.Engine.LoadPolicy(allowAllPolicy())

	doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/exec", map[string]interface{}{
		"type":   "file.write",
		"params": map[string]string{"path": "test.txt", "content": "hello"},
	})

	w := doRequest(s, "GET", "/api/v1/dashboard/stats", nil)
	var stats DashboardStatsResponse
	json.Unmarshal(w.Body.Bytes(), &stats)

	if stats.TotalActions < 1 {
		t.Errorf("expected at least 1 total action, got %d", stats.TotalActions)
	}
	if stats.AvgResponseMs < 0 {
		t.Errorf("expected non-negative avg response ms, got %f", stats.AvgResponseMs)
	}
}

func TestDashboardActivityEmpty(t *testing.T) {
	s := newTestServer()
	w := doRequest(s, "GET", "/api/v1/dashboard/activity", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result []ActivityEventResponse
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected 0 events, got %d", len(result))
	}
}

func TestDashboardActivityReturnsEvents(t *testing.T) {
	s := newTestServer()

	r := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "activity-test"})
	id := parseBody(r)["id"].(string)
	doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/start", nil)

	w := doRequest(s, "GET", "/api/v1/dashboard/activity", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result []ActivityEventResponse
	json.Unmarshal(w.Body.Bytes(), &result)

	if len(result) == 0 {
		t.Fatal("expected at least one activity event after start")
	}

	// Verify fields are populated
	for _, ev := range result {
		if ev.ID == "" {
			t.Error("expected non-empty ID")
		}
		if ev.Timestamp == "" {
			t.Error("expected non-empty timestamp")
		}
		if ev.EventType == "" {
			t.Error("expected non-empty event_type")
		}
		if ev.Effect == "" {
			t.Error("expected non-empty effect")
		}
	}
}

func TestDashboardActivityLimit(t *testing.T) {
	s := newTestServer()

	// Create multiple sandboxes to generate many events
	for i := 0; i < 25; i++ {
		r := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "limit-test"})
		id := parseBody(r)["id"].(string)
		doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/start", nil)
	}

	w := doRequest(s, "GET", "/api/v1/dashboard/activity", nil)
	var result []ActivityEventResponse
	json.Unmarshal(w.Body.Bytes(), &result)

	if len(result) > 20 {
		t.Errorf("expected at most 20 events, got %d", len(result))
	}
}
