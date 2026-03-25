package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/executor"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/policy"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/sandbox"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/trace"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// --- Request/Response types ---

// CreateSandboxRequest is the request body for creating a sandbox.
type CreateSandboxRequest struct {
	Name       string `json:"name"`
	PolicyFile string `json:"policy_file,omitempty"`
	RootDir    string `json:"root_dir,omitempty"`
}

// ExecActionRequest is the request body for executing an action.
type ExecActionRequest struct {
	Type   string            `json:"type"`
	Params map[string]string `json:"params"`
}

// ValidatePolicyRequest is the request body for validating a policy.
type ValidatePolicyRequest struct {
	Content string `json:"content"`
}

// SandboxResponse is the response body for sandbox info.
type SandboxResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	RootDir   string `json:"root_dir"`
	CreatedAt string `json:"created_at,omitempty"`
}

// --- Auth ---

// GenerateTokenRequest is the request body for generating a token.
type GenerateTokenRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"required"`
}

func (s *Server) handleGenerateToken(c *gin.Context) {
	if !s.config.AuthEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "auth is not enabled"})
		return
	}

	var req GenerateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if req.Role != "admin" && req.Role != "viewer" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be admin or viewer"})
		return
	}

	token, err := GenerateToken(s.config.AuthSecret, req.UserID, req.Role, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generate token: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// --- Health ---

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// --- Sandbox CRUD ---

func (s *Server) handleCreateSandbox(c *gin.Context) {
	if c.Request.Body == nil || c.Request.ContentLength == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request body is required"})
		return
	}

	var req CreateSandboxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	if verrs := ValidateCreateSandbox(req); verrs != nil {
		respondValidationError(c, verrs)
		return
	}

	id := uuid.New().String()
	name := req.Name
	if name == "" {
		name = "sandbox-" + id[:8]
	}

	rootDir := req.RootDir
	if rootDir == "" {
		rootDir = filepath.Join(os.TempDir(), "agent-sandbox", id)
	}

	cfg := sandbox.DefaultConfig()
	cfg.ID = id
	cfg.Name = name
	cfg.RootDir = rootDir
	cfg.TracePath = filepath.Join(rootDir, "traces.db")

	engine := policy.NewEngine()

	if req.PolicyFile != "" {
		p, err := policy.ParseFile(req.PolicyFile)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "load policy: " + err.Error()})
			return
		}
		if err := engine.LoadPolicy(*p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "apply policy: " + err.Error()})
			return
		}
	}

	recorder, err := trace.NewRecorder("")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create recorder: " + err.Error()})
		return
	}

	instance, err := sandbox.NewSandbox(cfg, engine, recorder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create sandbox: " + err.Error()})
		return
	}

	exec := executor.NewExecutor(cfg)

	s.mu.Lock()
	s.sandboxes[id] = &SandboxEntry{
		Instance: instance,
		Config:   cfg,
		Executor: exec,
		Recorder: recorder,
		Engine:   engine,
	}
	s.mu.Unlock()

	c.JSON(http.StatusCreated, SandboxResponse{
		ID:      id,
		Name:    name,
		Status:  string(instance.Status()),
		RootDir: rootDir,
	})
}

func (s *Server) handleListSandboxes(c *gin.Context) {
	statusFilter := c.Query("status")

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]SandboxResponse, 0, len(s.sandboxes))
	for id, entry := range s.sandboxes {
		status := string(entry.Instance.Status())
		if statusFilter != "" && status != statusFilter {
			continue
		}
		result = append(result, SandboxResponse{
			ID:      id,
			Name:    entry.Config.Name,
			Status:  status,
			RootDir: entry.Config.RootDir,
		})
	}

	c.JSON(http.StatusOK, gin.H{"sandboxes": result})
}

func (s *Server) handleGetSandbox(c *gin.Context) {
	entry, ok := s.getSandboxEntry(c)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, SandboxResponse{
		ID:      entry.Config.ID,
		Name:    entry.Config.Name,
		Status:  string(entry.Instance.Status()),
		RootDir: entry.Config.RootDir,
	})
}

func (s *Server) handleStartSandbox(c *gin.Context) {
	entry, ok := s.getSandboxEntry(c)
	if !ok {
		return
	}

	if err := entry.Instance.Start(context.Background()); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":     entry.Config.ID,
		"status": string(entry.Instance.Status()),
	})
}

func (s *Server) handleExecAction(c *gin.Context) {
	entry, ok := s.getSandboxEntry(c)
	if !ok {
		return
	}

	if c.Request.Body == nil || c.Request.ContentLength == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request body is required"})
		return
	}

	var req ExecActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	if verrs := ValidateExecAction(req); verrs != nil {
		respondValidationError(c, verrs)
		return
	}

	action := types.Action{
		ID:        uuid.New().String(),
		Type:      types.ActionType(req.Type),
		Params:    req.Params,
		Timestamp: time.Now(),
	}

	result, err := entry.Instance.Execute(context.Background(), action, entry.Executor.Execute)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Broadcast trace event to WebSocket clients
	s.broadcastTraceEvent(entry.Config.ID, types.TraceEvent{
		ID:        uuid.New().String(),
		SandboxID: entry.Config.ID,
		Type:      types.EventActionExecuted,
		ActionID:  action.ID,
		Timestamp: time.Now(),
		Data: map[string]string{
			"action_type": req.Type,
			"success":     fmt.Sprintf("%t", result.Success),
		},
	})

	c.JSON(http.StatusOK, result)
}

func (s *Server) handleStopSandbox(c *gin.Context) {
	entry, ok := s.getSandboxEntry(c)
	if !ok {
		return
	}

	if err := entry.Instance.Stop(context.Background()); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":     entry.Config.ID,
		"status": string(entry.Instance.Status()),
	})
}

func (s *Server) handleDestroySandbox(c *gin.Context) {
	id := c.Param("id")

	s.mu.Lock()
	entry, ok := s.sandboxes[id]
	if !ok {
		s.mu.Unlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "sandbox not found"})
		return
	}

	if entry.Instance.Status() == sandbox.StatusRunning {
		entry.Instance.Stop(context.Background())
	}
	entry.Recorder.Close()
	delete(s.sandboxes, id)
	s.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{"id": id, "destroyed": true})
}

// --- Traces ---

func (s *Server) handleGetTraces(c *gin.Context) {
	entry, ok := s.getSandboxEntry(c)
	if !ok {
		return
	}

	events, err := entry.Instance.GetTraces()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get traces: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}

// --- Replay ---

func (s *Server) handleStartReplay(c *gin.Context) {
	entry, ok := s.getSandboxEntry(c)
	if !ok {
		return
	}

	events, err := entry.Instance.GetTraces()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "load traces: " + err.Error()})
		return
	}

	id := entry.Config.ID
	s.replayMu.Lock()
	s.replays[id] = &ReplaySession{Events: events, Current: 0}
	s.replayMu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"sandbox_id":   id,
		"total_events": len(events),
		"status":       "started",
	})
}

func (s *Server) handleReplayNext(c *gin.Context) {
	id := c.Param("id")

	s.replayMu.Lock()
	session, ok := s.replays[id]
	if !ok {
		s.replayMu.Unlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "no replay session for this sandbox"})
		return
	}

	if session.Current >= len(session.Events) {
		s.replayMu.Unlock()
		c.JSON(http.StatusOK, gin.H{"done": true, "event": nil})
		return
	}

	event := session.Events[session.Current]
	session.Current++
	hasMore := session.Current < len(session.Events)
	s.replayMu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"event":    event,
		"has_more": hasMore,
		"position": session.Current,
		"total":    len(session.Events),
	})
}

// --- Policy ---

func (s *Server) handleValidatePolicy(c *gin.Context) {
	var req ValidatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	p, err := policy.Parse([]byte(req.Content))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"valid":  false,
			"errors": []string{err.Error()},
		})
		return
	}

	if verrs := ValidatePolicy(*p); verrs != nil {
		c.JSON(http.StatusOK, gin.H{
			"valid":  false,
			"errors": verrs.Errors,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":  true,
		"policy": p,
	})
}

// --- WebSocket ---

func (s *Server) handleWebSocket(c *gin.Context) {
	id := c.Param("id")

	s.mu.RLock()
	_, ok := s.sandboxes[id]
	s.mu.RUnlock()
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "sandbox not found"})
		return
	}

	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	s.wsMu.Lock()
	if s.wsClients[id] == nil {
		s.wsClients[id] = make(map[*websocket.Conn]bool)
	}
	s.wsClients[id][conn] = true
	s.wsMu.Unlock()

	// Keep connection alive until client disconnects
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			s.wsMu.Lock()
			delete(s.wsClients[id], conn)
			s.wsMu.Unlock()
			conn.Close()
			return
		}
	}
}

// --- Helpers ---

func (s *Server) getSandboxEntry(c *gin.Context) (*SandboxEntry, bool) {
	id := c.Param("id")
	s.mu.RLock()
	entry, ok := s.sandboxes[id]
	s.mu.RUnlock()
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "sandbox not found"})
		return nil, false
	}
	return entry, true
}
