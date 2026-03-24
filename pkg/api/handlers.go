package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/policy"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/sandbox"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

type createRequest struct {
	ID       string        `json:"id"`
	BasePath string        `json:"base_path,omitempty"`
	Policy   *types.Policy `json:"policy,omitempty"`
}

type execRequest struct {
	Action types.Action `json:"action"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

func (s *Server) handleCreateSandbox(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON: %v", err))
		return
	}

	if req.ID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if _, exists := s.Manager.Get(req.ID); exists {
		writeError(w, http.StatusConflict, "sandbox already exists")
		return
	}

	cfg := &sandbox.Config{
		ID:       req.ID,
		BasePath: req.BasePath,
		Policy:   req.Policy,
	}

	sb, err := sandbox.New(cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.Manager.Add(sb)
	writeJSON(w, http.StatusCreated, sb)
}

func (s *Server) handleListSandboxes(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.Manager.List())
}

func (s *Server) handleGetSandbox(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sb, ok := s.Manager.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "sandbox not found")
		return
	}
	writeJSON(w, http.StatusOK, sb)
}

func (s *Server) handleStartSandbox(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sb, ok := s.Manager.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "sandbox not found")
		return
	}
	if err := sb.Start(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sb)
}

func (s *Server) handleExecAction(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sb, ok := s.Manager.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "sandbox not found")
		return
	}

	var req execRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON: %v", err))
		return
	}

	if req.Action.Type == "" {
		writeError(w, http.StatusBadRequest, "action type is required")
		return
	}

	event, err := sb.Execute(&req.Action)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, event)
}

func (s *Server) handleGetTraces(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sb, ok := s.Manager.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "sandbox not found")
		return
	}

	traces, err := sb.GetTraces()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, traces)
}

func (s *Server) handleStopSandbox(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sb, ok := s.Manager.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "sandbox not found")
		return
	}
	if err := sb.Stop(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sb)
}

func (s *Server) handleDestroySandbox(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sb, ok := s.Manager.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "sandbox not found")
		return
	}
	if err := sb.Destroy(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.Manager.Delete(id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "destroyed"})
}

func (s *Server) handleValidatePolicy(w http.ResponseWriter, r *http.Request) {
	var p types.Policy
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON: %v", err))
		return
	}
	if err := policy.Validate(&p); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "valid"})
}
