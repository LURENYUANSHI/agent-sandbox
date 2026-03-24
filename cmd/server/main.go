package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/api"
)

func main() {
	port := flag.String("port", "8080", "server port")
	staticDir := flag.String("static-dir", "", "path to static web files")
	flag.Parse()

	srv := api.NewServer()

	if *staticDir != "" {
		if info, err := os.Stat(*staticDir); err == nil && info.IsDir() {
			srv.Mux.Handle("GET /", http.FileServer(http.Dir(*staticDir)))
		}
	}

	addr := fmt.Sprintf(":%s", *port)
	log.Printf("Starting agent-sandbox server on %s", addr)
	if err := srv.ListenAndServe(addr); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
