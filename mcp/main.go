package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	policyFile := flag.String("policy", "", "Path to default YAML policy file")
	version := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *version {
		fmt.Println("agent-sandbox-mcp v1.0.0")
		os.Exit(0)
	}

	server := NewMCPServer(*policyFile)
	if err := server.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %s\n", err)
		os.Exit(1)
	}
}
