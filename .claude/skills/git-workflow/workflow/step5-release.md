# Release: develop → main

## Steps

### 1. Merge develop into main
```bash
git checkout main && git pull
git merge develop --no-edit
git push origin main
```

### 2. Tag release
```bash
git tag -a v<version> -m "Release v<version>"
git push origin v<version>
```

### 3. Create GitHub Release
```bash
gh release create v<version> --title "v<version>" --generate-notes
```

### 4. Send Feishu Notification
```bash
bash automation/feishu-notify.sh "deploy" "Release v<version>" "Released to main branch"
```

### 5. Return to develop
```bash
git checkout develop
```
