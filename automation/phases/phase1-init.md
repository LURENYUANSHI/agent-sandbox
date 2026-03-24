You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox.

## Repository Info
- **GitHub Repo**: https://github.com/LURENYUANSHI/agent-sandbox
- **Remote**: origin → LURENYUANSHI/agent-sandbox
- **Branch Strategy**: main (production) + develop (integration), all work on feature branches from develop
- **Current working directory**: /c/Users/Administrator/ai-sandbox
- **Feishu notifications**: bash automation/feishu-notify.sh "<event>" "<title>" "<detail>"
- After completing your work, send a Feishu notification: bash automation/feishu-notify.sh "phase_complete" "Phase N completed" "description"



Read CLAUDE.md and docs/development-plan.md first to understand the full project.

## Your Task: Phase 1 - Project Initialization

Complete ALL of the following:

1. Initialize a git repository if not already done
2. Create go.mod with module `github.com/LURENYUANSHI/agent-sandbox` using Go 1.25
3. Create ALL directories listed in CLAUDE.md project structure
4. Create a Makefile with targets: build, test, lint, clean, run-server, run-web
5. Create LICENSE file with Apache 2.0 license
6. Write a basic README.md (will be expanded in Phase 9)
7. Initialize web/ directory:
   - Run `npm create vite@latest . -- --template react-ts` in web/ directory
   - Install tailwindcss, @tailwindcss/vite
   - Configure tailwind in vite.config.ts
   - Add CSS import for tailwind
8. Create docker/Dockerfile skeleton (multi-stage: go build + node build + runtime)
9. Create docker/docker-compose.yaml skeleton
10. Create placeholder files for all packages listed in CLAUDE.md (empty .go files with package declaration)
11. Git add and commit with message "feat: initialize project skeleton with Go, React, Docker setup"

IMPORTANT:
- Do NOT skip any step
- Verify each step worked before moving on
- If a command fails, debug and fix it
- End by running `git log --oneline` to confirm the commit
