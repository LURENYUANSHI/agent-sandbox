# Commit Message Format

All commits follow conventional commit format:

```
type(scope): description

[optional body]
```

## Commit Types

- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation updates
- `style` - Code formatting
- `refactor` - Code refactoring
- `perf` - Performance improvements
- `test` - Test additions or updates
- `build` - Build system or dependencies
- `ci` - CI/CD configuration
- `chore` - Other changes

## Scopes

sandbox, policy, trace, executor, api, cli, web, docker

## Examples

- `feat(sandbox): implement process isolation with cgroups`
- `feat(policy): add YAML policy parser with glob matching`
- `fix(trace): resolve SQLite concurrent write issue`
- `test(executor): add filesystem path traversal tests`
- `docs: update README with quick start guide`
- `ci: add GitHub Actions build workflow`
