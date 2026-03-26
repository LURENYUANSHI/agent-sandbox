#!/bin/bash
# ============================================================================
# AgentSandbox v0.3.0 Iteration Pipeline
# 10 phases: bugfix → auth → ratelimit → dashboard → websocket → config → coverage → cicd → resources → score
# ============================================================================

set -uo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PHASES_DIR="$PROJECT_DIR/automation/phases-v3"
LOGS_DIR="$PROJECT_DIR/automation/logs/v3"
PROGRESS_FILE="$PROJECT_DIR/automation/progress-v3.json"
CLAUDE_BIN="$(which claude)"
FEISHU="$PROJECT_DIR/automation/feishu-notify.sh"
REPO="LURENYUANSHI/agent-sandbox"

mkdir -p "$LOGS_DIR"

timestamp() { date '+%Y-%m-%d %H:%M:%S'; }
log() { echo "[$(timestamp)] $1" >> "$LOGS_DIR/orchestrator.log"; echo "[$(timestamp)] $1" >&2; }
notify() { bash "$FEISHU" "$@" >/dev/null 2>&1 || true; }

export PD="$PROJECT_DIR"

get_desc() {
    case "$1" in
        phase1-templates)     echo "Add GitHub issue/PR templates and contributing guide" ;;
        phase2-swagger)       echo "Add OpenAPI/Swagger API documentation" ;;
        phase3-rbac)          echo "Add role-based access control (admin/operator/viewer)" ;;
        phase4-audit-rotation) echo "Add audit log rotation and improve API coverage" ;;
        phase5-cli-tests)     echo "Add CLI integration tests" ;;
        phase6-demo)          echo "Create interactive demo examples" ;;
        phase7-python-sdk)    echo "Create Python SDK client library" ;;
        phase8-mcp)           echo "Add MCP server for AI agent integration" ;;
        phase9-prometheus)    echo "Add Prometheus metrics endpoint" ;;
        phase10-final)        echo "Final evaluation and v0.3.0 scoring" ;;
    esac
}
}

read_status() {
    python -c "
import json,os
try:
    d=json.load(open(os.path.join(os.environ.get('PD',''),'automation','progress-v3.json')))
    print(d['phases']['$1']['status'])
except: print('pending')
" 2>/dev/null || echo "pending"
}

update_status() {
    python -c "
import json,os
f=os.path.join(os.environ.get('PD',''),'automation','progress-v3.json')
try:
    d=json.load(open(f))
    d['phases']['$1']['$2']=$3
    json.dump(d,open(f,'w'),indent=2)
except: pass
" 2>/dev/null || true
}

run_phase() {
    local phase="$1"
