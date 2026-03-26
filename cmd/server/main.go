package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/api"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/config"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/trace"
)

// @title AgentSandbox API
// @version 0.3.0
// @description AI Agent Security Sandbox - Runtime isolation, policy enforcement, trace recording & replay
// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// Load application config from env (or YAML file via --config flag)
	var appCfg *config.AppConfig
	configFile := ""
	for i, arg := range os.Args[1:] {
		if arg == "--config" && i+1 < len(os.Args[1:]) {
			configFile = os.Args[i+2]
		}
	}

	if configFile != "" {
		var err error
		appCfg, err = config.LoadFromFile(configFile)
		if err != nil {
			log.Fatalf("Failed to load config file: %v", err)
		}
	} else {
		appCfg = config.LoadFromEnv()
	}

	devMode := os.Getenv("DEV_MODE") == "true" || os.Getenv("GIN_MODE") == "debug"

	authEnabled := appCfg.Server.AuthEnabled
	authSecret := ""
	auditRetentionDays := 90

	// Parse flags from args
	for i, arg := range os.Args[1:] {
		switch arg {
		case "--auth-enabled":
			authEnabled = true
		case "--auth-secret":
			if i+1 < len(os.Args[1:]) {
				authSecret = os.Args[i+2]
			}
		case "--audit-retention-days":
			if i+1 < len(os.Args[1:]) {
				if d, err := strconv.Atoi(os.Args[i+2]); err == nil && d > 0 {
					auditRetentionDays = d
				}
			}
		}
	}

	// Auto-generate secret if auth is enabled but no secret provided
	if authEnabled && authSecret == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			log.Fatalf("Failed to generate auth secret: %v", err)
		}
		authSecret = hex.EncodeToString(b)
	}

	cfg := api.ServerConfig{
		Port:         appCfg.Server.Port,
		DevMode:      devMode,
		AuthEnabled:  authEnabled,
		AuthSecret:   authSecret,
		CORSOrigins:  appCfg.Server.CORSOrigins,
		ExecConfig:   appCfg.Executor,
		PolicyConfig: appCfg.Policy,
	}

	srv := api.NewServer(cfg)

	// Initialize audit log rotation
	var auditLogger *trace.AuditLogger
	if cfg.AuditDBPath != "" {
		var err error
		auditLogger, err = trace.NewAuditLogger(cfg.AuditDBPath)
		if err != nil {
			log.Printf("WARNING: failed to initialize audit logger for rotation: %v", err)
		} else {
			auditLogger.SetRetentionDays(auditRetentionDays)
			auditLogger.StartAutoRotation(24 * time.Hour)
			log.Printf("Audit log rotation enabled: retention=%d days", auditRetentionDays)
		}
	}

	// Graceful shutdown on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	fmt.Printf("Agent Sandbox API server running on :%d\n", appCfg.Server.Port)

	if authEnabled {
		fmt.Printf("Auth enabled with secret: %s\n", authSecret)

		// Print a sample token for convenience
		token, err := api.GenerateToken(authSecret, "admin", "admin", 24*time.Hour)
		if err == nil {
			fmt.Printf("Sample admin token: %s\n", token)
		}
	}

	<-ctx.Done()

	if auditLogger != nil {
		auditLogger.StopAutoRotation()
		auditLogger.Close()
	}

	if err := srv.Shutdown(context.Background()); err != nil {
		log.Printf("Shutdown error: %v", err)
		os.Exit(1)
	}
	fmt.Println("Server stopped")
}
