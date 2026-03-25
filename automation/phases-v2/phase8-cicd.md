You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox on a feature branch.

## Your Task: Add GitHub Actions CI/CD pipeline

### 1. .github/workflows/ci.yml
Create CI workflow that runs on push to develop/main and on PRs:
```yaml
name: CI
on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [develop]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - name: Build
        run: go build ./...
      - name: Vet
        run: go vet ./...
      - name: Test
        run: go test ./... -v -count=1 -cover -coverprofile=coverage.out
      - name: Check coverage
        run: |
          go tool cover -func=coverage.out | tail -1
      - name: Upload coverage
        uses: actions/upload-artifact@v4
        with:
          name: coverage
          path: coverage.out

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - name: Check formatting
        run: |
          unformatted=$(gofmt -l .)
          if [ -n "$unformatted" ]; then
            echo "Unformatted files:"
            echo "$unformatted"
            exit 1
          fi

  web:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '22'
      - name: Install dependencies
        working-directory: web
        run: npm ci
      - name: Build
        working-directory: web
        run: npm run build

  docker:
    runs-on: ubuntu-latest
    needs: [test, web]
    if: github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4
      - name: Build Docker image
        run: docker build -f docker/Dockerfile -t agent-sandbox:latest .
```

### 2. .github/workflows/release.yml
Create release workflow triggered by tags:
```yaml
name: Release
on:
  push:
    tags: ['v*']

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - name: Build binaries
        run: |
          GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o agent-sandbox-linux-amd64 ./cmd/sandbox/
          GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o agent-sandbox-server-linux-amd64 ./cmd/server/
      - name: Create Release
        uses: softprops/action-gh-release@v2
        with:
          files: |
            agent-sandbox-linux-amd64
            agent-sandbox-server-linux-amd64
          generate_release_notes: true
```

### 3. Add badges to README.md
Add CI badge at the top of README.md:
```markdown
[![CI](https://github.com/LURENYUANSHI/agent-sandbox/actions/workflows/ci.yml/badge.svg)](https://github.com/LURENYUANSHI/agent-sandbox/actions/workflows/ci.yml)
```

### 4. Update Dockerfile with HEALTHCHECK
Add to docker/Dockerfile before ENTRYPOINT:
```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/api/v1/health || exit 1
```

### Verification:
1. Verify YAML syntax is valid
2. `go build ./...`
3. Commit: `ci: add GitHub Actions CI/CD pipeline with test, lint, web build, Docker, and release workflows`

DO NOT push to remote, DO NOT create PRs or issues. Only local commits.
