package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/executor"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/policy"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/sandbox"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/trace"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// ServerConfig holds configuration for the API server.
type ServerConfig struct {
	Port        int
	DevMode     bool
	AuthEnabled bool
	AuthSecret  string
	RateLimitRPS   float64 // Requests per second (0 = disabled)
	RateLimitBurst int     // Burst size for rate limiter
}

// SandboxEntry tracks a running sandbox and its associated resources.
type SandboxEntry struct {
	Instance *sandbox.Instance
	Config   sandbox.Config
	Executor *executor.Executor
	Recorder *trace.Recorder
	Engine   *policy.Engine
}

// ReplaySession tracks an active replay session.
type ReplaySession struct {
	Events  []types.TraceEvent
	Current int
}

// Server is the API server that manages sandboxes.
type Server struct {
	config     ServerConfig
	router     *gin.Engine
	httpServer *http.Server

	mu        sync.RWMutex
	sandboxes map[string]*SandboxEntry

	replayMu sync.Mutex
	replays  map[string]*ReplaySession

	upgrader websocket.Upgrader

	wsMu      sync.Mutex
	wsClients map[string]map[*websocket.Conn]bool
}

// NewServer creates a new API server.
func NewServer(config ServerConfig) *Server {
	if !config.DevMode {
		gin.SetMode(gin.ReleaseMode)
	}

	s := &Server{
		config:    config,
		sandboxes: make(map[string]*SandboxEntry),
		replays:   make(map[string]*ReplaySession),
		wsClients: make(map[string]map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}

	s.router = gin.New()
	s.setupRoutes()

	return s
}

func (s *Server) setupRoutes() {
	s.router.Use(Recovery())
	s.router.Use(RequestID())
	s.router.Use(Logger())
	s.router.Use(CORS())
	if s.config.RateLimitRPS > 0 {
		s.router.Use(NewRateLimiter(s.config.RateLimitRPS, s.config.RateLimitBurst))
	}
	s.router.Use(NewAuthMiddleware(s.config.AuthSecret, s.config.AuthEnabled))

	v1 := s.router.Group("/api/v1")
	{
		v1.GET("/health", s.handleHealth)
		v1.POST("/auth/token", s.handleGenerateToken)

		v1.POST("/sandboxes", s.handleCreateSandbox)
		v1.GET("/sandboxes", s.handleListSandboxes)
		v1.GET("/sandboxes/:id", s.handleGetSandbox)
		v1.POST("/sandboxes/:id/start", s.handleStartSandbox)
		v1.POST("/sandboxes/:id/exec", s.handleExecAction)
		v1.POST("/sandboxes/:id/stop", s.handleStopSandbox)
		v1.DELETE("/sandboxes/:id", s.handleDestroySandbox)
		v1.GET("/sandboxes/:id/traces", s.handleGetTraces)
		v1.POST("/sandboxes/:id/replay", s.handleStartReplay)
		v1.GET("/sandboxes/:id/replay/next", s.handleReplayNext)
		v1.POST("/policies/validate", s.handleValidatePolicy)
		v1.GET("/sandboxes/:id/ws", s.handleWebSocket)
	}
}

// Start begins serving HTTP requests.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	log.Printf("API server starting on %s", addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server listen: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down API server...")

	// Close all WebSocket connections
	s.wsMu.Lock()
	for _, clients := range s.wsClients {
		for conn := range clients {
			conn.Close()
		}
	}
	s.wsMu.Unlock()

	// Stop all running sandboxes
	s.mu.Lock()
	for _, entry := range s.sandboxes {
		if entry.Instance.Status() == sandbox.StatusRunning {
			entry.Instance.Stop(ctx)
		}
		entry.Recorder.Close()
	}
	s.mu.Unlock()

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(shutdownCtx)
}

// Router returns the gin engine for testing.
func (s *Server) Router() *gin.Engine {
	return s.router
}

// broadcastTraceEvent sends a trace event to all WebSocket clients for a sandbox.
func (s *Server) broadcastTraceEvent(sandboxID string, event types.TraceEvent) {
	s.wsMu.Lock()
	clients, ok := s.wsClients[sandboxID]
	if !ok {
		s.wsMu.Unlock()
		return
	}
	// Copy the client set to avoid holding the lock during writes
	conns := make([]*websocket.Conn, 0, len(clients))
	for conn := range clients {
		conns = append(conns, conn)
	}
	s.wsMu.Unlock()

	for _, conn := range conns {
		if err := conn.WriteJSON(event); err != nil {
			s.wsMu.Lock()
			delete(clients, conn)
			s.wsMu.Unlock()
			conn.Close()
		}
	}
}
