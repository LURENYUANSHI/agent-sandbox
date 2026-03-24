# Handle Merge Conflicts

## Steps

### 1. Update develop and rebase
```bash
git checkout develop && git pull
git checkout feature/<branch>
git rebase develop
```

### 2. Resolve conflicts
- Open conflicted files
- Keep the correct code
- `git add <resolved-file>`
- `git rebase --continue`

### 3. Force push (after rebase only)
```bash
git push --force-with-lease
```

### 4. Verify
```bash
go build ./...
go test ./... -count=1
```
