# Merge PR

## Steps

### 1. Merge PR to develop
```bash
gh pr merge <pr-number> --squash --delete-branch
```

### 2. Send Feishu Notification
```bash
bash automation/feishu-notify.sh "pr_merged" "PR #<num> merged" "Merged into develop"
```

### 3. Cleanup
```bash
git checkout develop && git pull
```
