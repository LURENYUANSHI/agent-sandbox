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
)

func main() {
	port := 8080
	if p := os.Getenv("PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}

	devMode := os.Getenv("DEV_MODE") == "true" || os.Getenv("GIN_MODE") == "debug"

	authEnabled := false
	authSecret := ""

	// Parse flags from args
	for i, arg := range os.Args[1:] {
		switch arg {
		case "--auth-enabled":
			authEnabled = true
		case "--auth-secret":
			if i+1 < len(os.Args[1:]) {
				authSecret = os.Args[i+2]
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
		Port:        port,
		DevMode:     devMode,
		AuthEnabled: authEnabled,
		AuthSecret:  authSecret,
	}

	srv := api.NewServer(cfg)

	// Graceful shutdown on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	fmt.Printf("Agent Sandbox API server running on :%d\n", port)

	if authEnabled {
		fmt.Printf("Auth enabled with secret: %s\n", authSecret)

		// Print a sample token for convenience
		token, err := api.GenerateToken(authSecret, "admin", "admin", 24*time.Hour)
		if err == nil {
			fmt.Printf("Sample admin token: %s\n", token)
		}
	}

	<-ctx.Done()

	if err := srv.Shutdown(context.Background()); err != nil {
		log.Printf("Shutdown error: %v", err)
		os.Exit(1)
	}
	fmt.Println("Server stopped")
}
