# Contributing to AgentSandbox

Thank you for your interest in contributing to AgentSandbox! This guide will help you get started.

## Development Setup

### Prerequisites

- **Go** 1.25 or later
- **Node.js** 22 or later (for the web dashboard)
- **Docker** (optional, for container-based testing)
- **Make**

### Getting Started

```bash
# Clone the repository
git clone https://github.com/LURENYUANSHI/agent-sandbox.git
cd agent-sandbox

# Install Go dependencies
go mod download

# Install web dashboard dependencies
cd web && npm install && cd ..

# Run setup script
bash scripts/setup.sh
```

## Branch Naming Conventions

Create branches from `develop` using the following prefixes:

- `feature/` — New features (e.g., `feature/add-network-policy`)
- `bugfix/` — Bug fixes (e.g., `bugfix/fix-trace-export`)
- `docs/` — Documentation changes (e.g., `docs/update-api-reference`)
- `refactor/` — Code refactoring (e.g., `refactor/simplify-executor`)
- `test/` — Test additions or fixes (e.g., `test/add-policy-tests`)

## Commit Message Format

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): description
```

**Types:** `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`

**Scopes:** `sandbox`, `policy`, `trace`, `executor`, `api`, `cli`, `web`, `docker`

**Examples:**

```
feat(sandbox): add Windows job object isolation
fix(trace): prevent duplicate span IDs in export
docs(api): update REST endpoint documentation
test(policy): add table-driven tests for rule evaluation
```

## Running Tests

```bash
# Run all Go tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./pkg/sandbox/...

# Run integration tests
go test ./test/integration/...

# Lint Go code
go vet ./...

# Build the project
make build
```

## Pull Request Process

1. Create a branch from `develop` following the naming conventions above.
2. Make your changes, ensuring tests pass and code compiles.
3. Commit using the conventional commit format.
4. Push your branch and open a PR targeting `develop` (never `main` directly).
5. Fill out the PR template with a summary, related issue, and checklist.
6. Wait for code review and CI checks to pass.

## Code Review Expectations

- PRs require at least one approving review before merge.
- Reviewers will check for correctness, test coverage, and adherence to coding conventions.
- Go code should pass `go vet` and `gofmt` with no issues.
- Core packages (`pkg/sandbox`, `pkg/executor`, `pkg/policy`, `pkg/trace`) should maintain at least 80% test coverage.
- Error handling should wrap errors with context: `fmt.Errorf("...: %w", err)`.
- React code should use functional components, hooks, and TypeScript strict mode.
