package api

import (
	"net/http"
	"sync"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/sandbox"
)

// Manager tracks active sandboxes.
type Manager struct {
	sandboxes map[string]*sandbox.Sandbox
	mu        sync.RWMutex
}

// NewManager creates a sandbox manager.
func NewManager() *Manager {
	return &Manager{sandboxes: make(map[string]*sandbox.Sandbox)}
}

// Get returns a sandbox by ID.
func (m *Manager) Get(id string) (*sandbox.Sandbox, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sb, ok := m.sandboxes[id]
	return sb, ok
}

// Add registers a sandbox.
func (m *Manager) Add(sb *sandbox.Sandbox) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sandboxes[sb.ID] = sb
}

// Delete removes a sandbox.
func (m *Manager) Delete(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sandboxes, id)
}

// List returns all sandboxes.
func (m *Manager) List() []*sandbox.Sandbox {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*sandbox.Sandbox, 0, len(m.sandboxes))
	for _, sb := range m.sandboxes {
		result = append(result, sb)
	}
	return result
}

// Server is the HTTP API server.
type Server struct {
	Manager *Manager
	Mux     *http.ServeMux
}

// NewServer creates an API server with all routes registered.
func NewServer() *Server {
	s := &Server{
		Manager: NewManager(),
		Mux:     http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

// NewServerWithManager creates an API server with an existing manager.
func NewServerWithManager(m *Manager) *Server {
	s := &Server{
		Manager: m,
		Mux:     http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.Mux.HandleFunc("POST /api/sandboxes", withLogging(s.handleCreateSandbox))
	s.Mux.HandleFunc("GET /api/sandboxes", withLogging(s.handleListSandboxes))
	s.Mux.HandleFunc("GET /api/sandboxes/{id}", withLogging(s.handleGetSandbox))
	s.Mux.HandleFunc("POST /api/sandboxes/{id}/start", withLogging(s.handleStartSandbox))
	s.Mux.HandleFunc("POST /api/sandboxes/{id}/exec", withLogging(s.handleExecAction))
	s.Mux.HandleFunc("GET /api/sandboxes/{id}/traces", withLogging(s.handleGetTraces))
	s.Mux.HandleFunc("POST /api/sandboxes/{id}/stop", withLogging(s.handleStopSandbox))
	s.Mux.HandleFunc("DELETE /api/sandboxes/{id}", withLogging(s.handleDestroySandbox))
	s.Mux.HandleFunc("POST /api/policies/validate", withLogging(s.handleValidatePolicy))
}

// Handler returns the HTTP handler for the server.
func (s *Server) Handler() http.Handler {
	return withCORS(s.Mux)
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s.Handler())
}
