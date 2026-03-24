# Create GitHub Issue

## Prerequisites
- On `develop` branch

## Steps

### 1. Create GitHub Issue

```bash
gh issue create \
  --title "<title>" \
  --body "## 描述\n...\n## 验收标准\n- [ ] 条件1\n- [ ] 条件2" \
  --label "type::feature,status::backlog,priority::p1,area::<area>"
```

**Required labels**:
- `type::*` label
- `priority::*` label
- `area::*` label
- Acceptance criteria with `- [ ]` checklist items in body

### 2. Send Feishu Notification

```bash
bash automation/feishu-notify.sh "issue_created" "Issue #<num>: <title>" "<description>"
```

### 3. Create Branch and Start Development

```bash
git checkout develop && git pull
git checkout -b feature/<issue-id>-<slug>
git push -u origin feature/<issue-id>-<slug>
```

## Completion Checklist
- [ ] GitHub issue created with correct labels
- [ ] Feishu notification sent
- [ ] Feature branch created and pushed
