# Create Pull Request

## Prerequisites
- [ ] Commits follow conventional format
- [ ] Go tests pass: `go test ./...`
- [ ] Code compiles: `go build ./...`
- [ ] go vet passes: `go vet ./...`
- [ ] Branch synced with develop

## PR Title Format
`[area] Description (closes #issue-id)`

## Steps

### Step 1: Verify Code Quality
```bash
go build ./...
go test ./... -count=1
go vet ./...
```

### Step 2: Create PR
**CRITICAL**: Always set `--base develop`

```bash
gh pr create --draft \
  --title "[sandbox] Description (closes #<issue-id>)" \
  --label "type::feature,status::review,area::sandbox" \
  --base develop
```

### Step 3: Send Feishu Notification
```bash
bash automation/feishu-notify.sh "pr_created" "PR #<num>: <title>" "<link>"
```

## PR-Issue Linking Rules

| Scenario         | Keywords                              |
| ---------------- | ------------------------------------- |
| Partial work     | `Related to #<id>` or `Ref #<id>`    |
| Final completion | `Closes #<id>` / `Fixes` / `Resolves`|

## Completion Checklist
- [ ] PR created targeting develop branch
- [ ] Feishu notification sent
- [ ] PR title contains issue reference
- [ ] PR body includes `Closes #<id>`
