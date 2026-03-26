You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox on a feature branch.

## Your Task: Create GitHub Issue/PR templates

### 1. .github/ISSUE_TEMPLATE/bug_report.yml
```yaml
name: Bug Report
description: Report a bug in AgentSandbox
title: "[Bug]: "
labels: ["type::bug", "status::backlog"]
body:
  - type: markdown
    attributes:
      value: Thanks for reporting a bug!
  - type: textarea
    id: description
    attributes:
      label: Description
      description: A clear description of the bug
    validations:
      required: true
  - type: textarea
    id: steps
    attributes:
      label: Steps to Reproduce
      description: Steps to reproduce the behavior
      value: |
        1.
        2.
        3.
    validations:
      required: true
  - type: textarea
    id: expected
    attributes:
      label: Expected Behavior
    validations:
      required: true
  - type: textarea
    id: actual
    attributes:
      label: Actual Behavior
    validations:
      required: true
  - type: dropdown
    id: component
    attributes:
      label: Component
      options:
        - Sandbox Runtime
        - Policy Engine
        - Trace System
        - CLI
        - REST API
        - Web Dashboard
        - Docker
        - Other
  - type: textarea
    id: environment
    attributes:
      label: Environment
      description: OS, Go version, Docker version, etc.
```

### 2. .github/ISSUE_TEMPLATE/feature_request.yml
Similar structure for feature requests with fields: description, use case, proposed solution, alternatives considered.

### 3. .github/PULL_REQUEST_TEMPLATE.md
```markdown
## Summary
<!-- What does this PR do? -->

## Related Issue
<!-- Closes #XX -->

## Changes
-

## Checklist
- [ ] Code compiles (`go build ./...`)
- [ ] Tests pass (`go test ./...`)
- [ ] go vet passes
- [ ] Follows conventional commit format
```

### 4. .github/ISSUE_TEMPLATE/config.yml
```yaml
blank_issues_enabled: true
contact_links:
  - name: Documentation
    url: https://github.com/LURENYUANSHI/agent-sandbox/tree/main/docs
    about: Read the documentation
```

### 5. CONTRIBUTING.md
Write a contributing guide covering:
- Development setup (Go 1.25+, Node 22+)
- Branch naming conventions (feature/, bugfix/, etc.)
- Commit message format (conventional commits)
- How to run tests
- PR process
- Code review expectations

### Verification:
1. All YAML files are valid
2. Commit: `docs: add GitHub issue/PR templates and contributing guide`

DO NOT push to remote, DO NOT create PRs or issues. Only local commits.
