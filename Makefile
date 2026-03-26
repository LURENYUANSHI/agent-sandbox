.PHONY: build build-cli build-server build-mcp test test-integration lint clean run-server run-web docker-build docker-run

MODULE := github.com/LURENYUANSHI/agent-sandbox
BIN_DIR := bin
CLI_BIN := $(BIN_DIR)/agent-sandbox
SERVER_BIN := $(BIN_DIR)/agent-sandbox-server
MCP_BIN := $(BIN_DIR)/agent-sandbox-mcp

build: build-cli build-server build-mcp

build-cli:
	go build -o $(CLI_BIN) ./cmd/sandbox

build-server:
	go build -o $(SERVER_BIN) ./cmd/server

build-mcp:
	go build -o $(MCP_BIN) ./mcp

test:
	go test ./pkg/... -v -cover

test-integration:
	go test ./test/integration/... -v -cover

lint:
	go vet ./...
	gofmt -l .

clean:
	rm -rf $(BIN_DIR)
	rm -f coverage.out

run-server:
	go run ./cmd/server

run-web:
	cd web && npm run dev

docker-build:
	docker build -f docker/Dockerfile -t agent-sandbox:latest .

docker-run:
	cd docker && docker compose up -d
