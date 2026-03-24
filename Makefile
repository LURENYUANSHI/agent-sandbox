.PHONY: build test lint clean run-server run-web

MODULE := github.com/LURENYUANSHI/agent-sandbox
BIN_DIR := bin
CLI_BIN := $(BIN_DIR)/agent-sandbox
SERVER_BIN := $(BIN_DIR)/agent-sandbox-server

build: build-cli build-server

build-cli:
	go build -o $(CLI_BIN) ./cmd/sandbox

build-server:
	go build -o $(SERVER_BIN) ./cmd/server

test:
	go test ./... -v -cover

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
