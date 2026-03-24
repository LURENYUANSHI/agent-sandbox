#!/bin/bash
# Dev environment setup script

set -e

echo "Setting up AgentSandbox development environment..."

# Check Go
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed"
    exit 1
fi

# Check Node
if ! command -v node &> /dev/null; then
    echo "Error: Node.js is not installed"
    exit 1
fi

# Install Go dependencies
echo "Installing Go dependencies..."
go mod tidy

# Install web dependencies
echo "Installing web dependencies..."
cd web && npm install && cd ..

echo "Setup complete!"
