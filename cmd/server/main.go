package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

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

	cfg := api.ServerConfig{
		Port:    port,
		DevMode: devMode,
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
	<-ctx.Done()

	if err := srv.Shutdown(context.Background()); err != nil {
		log.Printf("Shutdown error: %v", err)
		os.Exit(1)
	}
	fmt.Println("Server stopped")
}
