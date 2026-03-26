#!/bin/bash
# One-command demo: run the interactive AgentSandbox demo
set -e

echo "========================================"
echo "  AgentSandbox — One-Click Demo"
echo "========================================"
echo ""
echo "This demo creates a sandboxed environment, runs a series of"
echo "agent actions through the policy engine, and displays the"
echo "execution trace."
echo ""

cd "$(dirname "$0")/.."

echo "Building demo..."
go build -o /tmp/sandbox-demo ./examples/demo/
echo ""

echo "Running demo..."
echo "----------------------------------------"
/tmp/sandbox-demo

echo "========================================"
echo "  Demo complete!"
echo ""
echo "  Try the other examples:"
echo "    go run examples/coding-agent/main.go"
echo "    go run examples/web-scraper/main.go"
echo "========================================"
