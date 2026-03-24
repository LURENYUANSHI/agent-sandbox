# Branch Management

## Branch Model

- **main**: Production branch, stable releases only
- **develop**: Integration branch, all PRs merge here
- **feature/bugfix/refactor/...**: Feature branches, created from develop

**CRITICAL**: All feature branches MUST be created from `develop`, NOT from `main`.

## Branch Naming Conventions

```bash
git checkout develop && git pull
git checkout -b feature/<id>-<slug>      # type::feature
git checkout -b bugfix/<id>-<slug>       # type::bug
git checkout -b refactor/<slug>          # type::refactor
git checkout -b chore/<slug>             # type::chore
git checkout -b docs/<slug>              # type::docs
git checkout -b test/<slug>              # type::test
```

## Merge Flow

1. Create feature branch from `develop`
2. Submit Draft PR to `develop`
3. Review + merge to `develop`
4. Periodically merge `develop` → `main` for releases
