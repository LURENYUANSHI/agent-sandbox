---
name: git-workflow
description: Git workflow automation for GitHub operations including commits, pull requests, issue management with labels, branch management, and conflict resolution. Use when users need help with git operations, GitHub issues, PRs, or status tracking.
---

# Git Workflow — Agent 操作手册

本项目使用 GitHub Flow + 自动化流水线。Agent 的每一步操作都会触发对应的自动化工作流。

## 核心原则

**操作不规范 = 流水线断裂。严格遵循 label、branch、commit 规范。**

## 操作 → 触发 速查

| 你做什么                     | GitHub 自动做什么                                        |
| ---------------------------- | -------------------------------------------------------- |
| `gh issue create` + 正确标签 | Issue 创建 + 飞书通知                                    |
| 提交代码 + 创建 Draft PR     | 飞书通知 + 代码审查                                      |
| PR 合并到 develop            | 自动关闭 issue + 飞书通知                                |
| develop → main               | Release 发布 + 飞书通知                                  |

## 手动工作流

### Issue 开发周期

| Step | 你做什么                         | 触发什么           | Guide                                                                    |
| ---- | -------------------------------- | ------------------ | ------------------------------------------------------------------------ |
| 1    | 创建 Issue（加正确标签）         | 飞书通知           | [workflow/step1-publish-issue.md](workflow/step1-publish-issue.md)       |
| 2    | 写代码 + 创建 Draft PR           | 飞书通知 + Review  | [workflow/step2-create-pr.md](workflow/step2-create-pr.md)               |
| 3    | 处理合并冲突                     | —                  | [workflow/step3-handle-conflicts.md](workflow/step3-handle-conflicts.md) |
| 4    | 合并 PR                          | 自动关闭 Issue     | [workflow/step4-merge.md](workflow/step4-merge.md)                       |

### 其他工作流

| Workflow       | Description         | Guide                                                    |
| -------------- | ------------------- | -------------------------------------------------------- |
| Release        | develop → main 发布 | [workflow/step5-release.md](workflow/step5-release.md)   |

## 关键操作规范

### 创建 Issue

```bash
gh issue create \
  --title "Issue Title" \
  --body "## 描述\n...\n## 验收标准\n- [ ] 条件1\n- [ ] 条件2" \
  --label "type::feature,status::backlog,priority::p1,area::sandbox"
```

**必须包含**:
- `type::*` 标签
- `priority::*` 标签
- `area::*` 标签
- Body 中包含验收标准（`- [ ]` checklist）

### 创建 PR

```bash
gh pr create --draft \
  --title "[area] Description (closes #issue-id)" \
  --label "type::feature,status::review,area::sandbox" \
  --base develop
```

**关键**: `--base develop`（必须）, Body 中 `Closes #N`

## Rules

| Rule        | Reference                                        |
| ----------- | ------------------------------------------------ |
| 标签系统    | [rules/labels.md](rules/labels.md)               |
| Commit 格式 | [rules/commit-format.md](rules/commit-format.md) |
| 分支命名    | [rules/branch-naming.md](rules/branch-naming.md) |

## Branch Flow

```
develop → feature/<id>-<slug> → Draft PR → Review → Merge → develop → (release) → main
```
