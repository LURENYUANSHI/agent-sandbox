package api

import (
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/sandbox"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// DashboardStatsResponse is the response for GET /api/v1/dashboard/stats.
type DashboardStatsResponse struct {
	ActiveSandboxes int     `json:"active_sandboxes"`
	TotalActions    int     `json:"total_actions"`
	DeniedActions   int     `json:"denied_actions"`
	AvgResponseMs   float64 `json:"avg_response_ms"`
}

// ActivityEventResponse is a single item in the recent activity list.
type ActivityEventResponse struct {
	ID           string  `json:"id"`
	SandboxID    string  `json:"sandbox_id"`
	EventType    string  `json:"event_type"`
	ActionType   string  `json:"action_type,omitempty"`
	ActionDetail string  `json:"action_detail"`
	Effect       string  `json:"effect"`
	DurationMs   float64 `json:"duration_ms"`
	Timestamp    string  `json:"timestamp"`
}

// handleGetDashboardStats returns aggregate statistics for the dashboard.
// @Summary Get dashboard statistics
// @Description Returns aggregate statistics including active sandboxes, total/denied actions, and average response time
// @Tags dashboard
// @Produce json
// @Success 200 {object} DashboardStatsResponse
// @Security BearerAuth
// @Router /dashboard/stats [get]
func (s *Server) handleGetDashboardStats(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var stats DashboardStatsResponse
	var totalDurationMs float64
	var executedCount int

	for _, entry := range s.sandboxes {
		if entry.Instance.Status() == sandbox.StatusRunning {
			stats.ActiveSandboxes++
		}

		events, err := entry.Instance.GetTraces()
		if err != nil {
			continue
		}

		for _, ev := range events {
			switch ev.Type {
			case types.EventActionExecuted:
				stats.TotalActions++
				executedCount++
				totalDurationMs += float64(ev.Duration) / float64(time.Millisecond)
			case types.EventActionDenied:
				stats.TotalActions++
				stats.DeniedActions++
			case types.EventActionFailed:
				stats.TotalActions++
				executedCount++
				totalDurationMs += float64(ev.Duration) / float64(time.Millisecond)
			}
		}
	}

	if executedCount > 0 {
		stats.AvgResponseMs = totalDurationMs / float64(executedCount)
	}

	c.JSON(http.StatusOK, stats)
}

// handleGetRecentActivity returns the most recent trace events across all sandboxes.
// @Summary Get recent activity
// @Description Returns the 20 most recent trace events across all sandboxes, sorted by timestamp descending
// @Tags dashboard
// @Produce json
// @Success 200 {array} ActivityEventResponse
// @Security BearerAuth
// @Router /dashboard/activity [get]
func (s *Server) handleGetRecentActivity(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var allEvents []types.TraceEvent
	for _, entry := range s.sandboxes {
		events, err := entry.Instance.GetTraces()
		if err != nil {
			continue
		}
		allEvents = append(allEvents, events...)
	}

	// Sort by timestamp descending
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].Timestamp.After(allEvents[j].Timestamp)
	})

	// Limit to 20
	if len(allEvents) > 20 {
		allEvents = allEvents[:20]
	}

	result := make([]ActivityEventResponse, 0, len(allEvents))
	for _, ev := range allEvents {
		result = append(result, mapToActivityEvent(ev))
	}

	c.JSON(http.StatusOK, result)
}

func mapToActivityEvent(ev types.TraceEvent) ActivityEventResponse {
	return ActivityEventResponse{
		ID:           ev.ID,
		SandboxID:    ev.SandboxID,
		EventType:    string(ev.Type),
		ActionType:   ev.Data["type"],
		ActionDetail: detailForEvent(ev),
		Effect:       effectForEvent(ev),
		DurationMs:   float64(ev.Duration) / float64(time.Millisecond),
		Timestamp:    ev.Timestamp.Format(time.RFC3339),
	}
}

func effectForEvent(ev types.TraceEvent) string {
	switch ev.Type {
	case types.EventActionDenied, types.EventActionFailed:
		return "deny"
	case types.EventActionAllowed, types.EventActionExecuted:
		return "allow"
	default:
		return "audit"
	}
}

func detailForEvent(ev types.TraceEvent) string {
	switch ev.Type {
	case types.EventSandboxStarted:
		if name := ev.Data["name"]; name != "" {
			return "Sandbox started: " + name
		}
		return "Sandbox started"
	case types.EventSandboxStopped:
		return "Sandbox stopped"
	case types.EventSandboxCreated:
		return "Sandbox created"
	case types.EventActionRequested:
		if t := ev.Data["type"]; t != "" {
			return "Action requested: " + t
		}
		return "Action requested"
	case types.EventPolicyEvaluated:
		if allowed := ev.Data["allowed"]; allowed != "" {
			return "Policy evaluated: " + allowed
		}
		return "Policy evaluated"
	case types.EventActionDenied:
		if reason := ev.Data["reason"]; reason != "" {
			return "Action denied: " + reason
		}
		return "Action denied"
	case types.EventActionExecuted:
		return "Action executed"
	case types.EventActionFailed:
		if errMsg := ev.Data["error"]; errMsg != "" {
			return "Action failed: " + errMsg
		}
		return "Action failed"
	default:
		return string(ev.Type)
	}
}
