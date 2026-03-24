#!/bin/bash
# Quick test: run ONLY Phase 1 through the full pipeline
# Issue → Branch → Code → Test → PR → Review → Merge
# Expected: ~5 minutes

set -euo pipefail

PROJECT_DIR="/c/Users/Administrator/ai-sandbox"
source "$PROJECT_DIR/automation/run-overnight.sh" 2>/dev/null || true

# Override PHASES to only run phase1
cd "$PROJECT_DIR"

PHASES_DIR="$PROJECT_DIR/automation/phases"
LOGS_DIR="$PROJECT_DIR/automation/logs"
PROGRESS_FILE="$PROJECT_DIR/automation/progress.json"
CLAUDE_BIN="/c/Users/Administrator/.local/bin/claude"
FEISHU="$PROJECT_DIR/automation/feishu-notify.sh"
REPO="LURENYUANSHI/agent-sandbox"

mkdir -p "$LOGS_DIR"

echo "[TEST] Starting Phase 1 test run..."
echo "[TEST] This tests the full pipeline: Issue → Branch → Code → Test → PR → Review → Merge"
echo ""

# Just source the functions and run phase1
# We need to inline the key functions here since sourcing the full script would start everything

bash -c "
cd $PROJECT_DIR
# Run only phase1 from the main script by modifying PHASES
sed 's/PHASES=(/PHASES=(\n    \"phase1-init\"\n#/' $PROJECT_DIR/automation/run-overnight.sh | \
sed '/phase2-types/,/phase9-polish/s/^/#/' > /tmp/test-run-phase1.sh
" 2>/dev/null || true

# Simpler approach: just run the overnight script but it will skip completed phases
# So let's reset phase1 status and run
python -c "
import json
data = {
    'start_time': '$(date '+%Y-%m-%d %H:%M:%S')',
    'phases': {
        'phase1-init': {'status':'pending','start':'','end':'','exit_code':-1,'issue':'','pr':'','branch':'','review':'','test_result':''},
        'phase2-types': {'status':'completed','start':'','end':'','exit_code':0,'issue':'','pr':'','branch':'','review':'','test_result':''},
        'phase3-policy': {'status':'completed','start':'','end':'','exit_code':0,'issue':'','pr':'','branch':'','review':'','test_result':''},
        'phase4-trace': {'status':'completed','start':'','end':'','exit_code':0,'issue':'','pr':'','branch':'','review':'','test_result':''},
        'phase5-sandbox': {'status':'completed','start':'','end':'','exit_code':0,'issue':'','pr':'','branch':'','review':'','test_result':''},
        'phase6-cli-api': {'status':'completed','start':'','end':'','exit_code':0,'issue':'','pr':'','branch':'','review':'','test_result':''},
        'phase7-web': {'status':'completed','start':'','end':'','exit_code':0,'issue':'','pr':'','branch':'','review':'','test_result':''},
        'phase8-integration': {'status':'completed','start':'','end':'','exit_code':0,'issue':'','pr':'','branch':'','review':'','test_result':''},
        'phase9-polish': {'status':'completed','start':'','end':'','exit_code':0,'issue':'','pr':'','branch':'','review':'','test_result':''}
    },
    'end_time': '',
    'overall_status': 'running'
}
with open('$PROGRESS_FILE', 'w') as f:
    json.dump(data, f, indent=2)
"

bash "$PROJECT_DIR/automation/run-overnight.sh"
