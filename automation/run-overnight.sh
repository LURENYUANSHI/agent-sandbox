#!/bin/bash
# ============================================================================
# AgentSandbox - Overnight Autonomous Development Pipeline
# Runs 9 phases of development using Claude Code in non-interactive mode
# Expected duration: ~8 hours (00:00 - 08:00)
# ============================================================================

set -euo pipefail

PROJECT_DIR="/c/Users/Administrator/ai-sandbox"
PHASES_DIR="$PROJECT_DIR/automation/phases"
LOGS_DIR="$PROJECT_DIR/automation/logs"
PROGRESS_FILE="$PROJECT_DIR/automation/progress.json"
CLAUDE_BIN="/c/Users/Administrator/.local/bin/claude"

# Ensure logs directory exists
mkdir -p "$LOGS_DIR"

# Timestamp helper
timestamp() {
    date '+%Y-%m-%d %H:%M:%S'
}

# Log helper
log() {
    echo "[$(timestamp)] $1" | tee -a "$LOGS_DIR/orchestrator.log"
}

# Initialize progress tracking
init_progress() {
    cat > "$PROGRESS_FILE" << 'PROGRESS_EOF'
{
    "start_time": "",
    "phases": {
        "phase1-init": {"status": "pending", "start": "", "end": "", "exit_code": -1},
        "phase2-types": {"status": "pending", "start": "", "end": "", "exit_code": -1},
        "phase3-policy": {"status": "pending", "start": "", "end": "", "exit_code": -1},
        "phase4-trace": {"status": "pending", "start": "", "end": "", "exit_code": -1},
        "phase5-sandbox": {"status": "pending", "start": "", "end": "", "exit_code": -1},
        "phase6-cli-api": {"status": "pending", "start": "", "end": "", "exit_code": -1},
        "phase7-web": {"status": "pending", "start": "", "end": "", "exit_code": -1},
        "phase8-integration": {"status": "pending", "start": "", "end": "", "exit_code": -1},
        "phase9-polish": {"status": "pending", "start": "", "end": "", "exit_code": -1}
    },
    "end_time": "",
    "overall_status": "running"
}
PROGRESS_EOF
    # Set start time
    sed -i "s/\"start_time\": \"\"/\"start_time\": \"$(timestamp)\"/" "$PROGRESS_FILE"
}

# Update phase status in progress file
update_phase() {
    local phase=$1
    local field=$2
    local value=$3
    # Use python for reliable JSON updates
    python -c "
import json
with open('$PROGRESS_FILE', 'r') as f:
    data = json.load(f)
data['phases']['$phase']['$field'] = $value
with open('$PROGRESS_FILE', 'w') as f:
    json.dump(data, f, indent=2)
"
}

# Run a single phase
run_phase() {
    local phase_name=$1
    local phase_file="$PHASES_DIR/${phase_name}.md"
    local phase_log="$LOGS_DIR/${phase_name}.log"

    if [ ! -f "$phase_file" ]; then
        log "ERROR: Phase file not found: $phase_file"
        return 1
    fi

    log "=========================================="
    log "Starting: $phase_name"
    log "=========================================="

    update_phase "$phase_name" "status" '"running"'
    update_phase "$phase_name" "start" "\"$(timestamp)\""

    # Read the phase prompt
    local prompt
    prompt=$(cat "$phase_file")

    # Run Claude Code in non-interactive mode
    local exit_code=0
    "$CLAUDE_BIN" -p "$prompt" \
        --dangerously-skip-permissions \
        --model opus \
        --effort max \
        --verbose \
        > "$phase_log" 2>&1 || exit_code=$?

    update_phase "$phase_name" "end" "\"$(timestamp)\""
    update_phase "$phase_name" "exit_code" "$exit_code"

    if [ $exit_code -eq 0 ]; then
        update_phase "$phase_name" "status" '"completed"'
        log "COMPLETED: $phase_name (exit code: $exit_code)"
    else
        update_phase "$phase_name" "status" '"failed"'
        log "FAILED: $phase_name (exit code: $exit_code)"
        log "Check log: $phase_log"

        # Attempt recovery: run a fix-up prompt
        log "Attempting recovery for $phase_name..."
        local recovery_prompt="You are working on /c/Users/Administrator/ai-sandbox. The previous phase ($phase_name) failed. Read the CLAUDE.md and check git log to see what was done. Read the phase description in automation/phases/${phase_name}.md. Fix any issues and complete remaining work. Run tests to verify. Commit when done."

        "$CLAUDE_BIN" -p "$recovery_prompt" \
            --dangerously-skip-permissions \
            --model opus \
            --effort max \
            > "$LOGS_DIR/${phase_name}-recovery.log" 2>&1 || true

        # Check if recovery succeeded by verifying go build
        if (cd "$PROJECT_DIR" && go build ./... 2>/dev/null); then
            update_phase "$phase_name" "status" '"recovered"'
            log "RECOVERED: $phase_name"
        else
            log "RECOVERY FAILED: $phase_name - continuing to next phase"
        fi
    fi

    return 0  # Always continue to next phase
}

# ============================================================================
# Main execution
# ============================================================================

log "============================================"
log "AgentSandbox Overnight Build Starting"
log "Project: $PROJECT_DIR"
log "============================================"

init_progress

# Phase list in order
PHASES=(
    "phase1-init"
    "phase2-types"
    "phase3-policy"
    "phase4-trace"
    "phase5-sandbox"
    "phase6-cli-api"
    "phase7-web"
    "phase8-integration"
    "phase9-polish"
)

# Check if we should resume from a specific phase
RESUME_FROM=""
if [ -f "$PROGRESS_FILE" ]; then
    # Find last completed phase and resume from next
    for phase in "${PHASES[@]}"; do
        status=$(python -c "
import json
with open('$PROGRESS_FILE') as f:
    data = json.load(f)
print(data['phases']['$phase']['status'])
" 2>/dev/null || echo "pending")

        if [ "$status" = "pending" ] || [ "$status" = "failed" ]; then
            RESUME_FROM="$phase"
            break
        fi
    done
fi

# Run all phases
started=false
for phase in "${PHASES[@]}"; do
    # Skip completed phases if resuming
    if [ -n "$RESUME_FROM" ] && [ "$started" = false ]; then
        if [ "$phase" != "$RESUME_FROM" ]; then
            log "Skipping already completed: $phase"
            continue
        fi
        started=true
    fi

    run_phase "$phase"
done

# Update overall status
python -c "
import json
with open('$PROGRESS_FILE', 'r') as f:
    data = json.load(f)
data['end_time'] = '$(timestamp)'
failed = [k for k, v in data['phases'].items() if v['status'] == 'failed']
data['overall_status'] = 'completed' if not failed else 'completed_with_errors'
data['failed_phases'] = failed
with open('$PROGRESS_FILE', 'w') as f:
    json.dump(data, f, indent=2)
"

log "============================================"
log "AgentSandbox Overnight Build Complete"
log "Check progress: $PROGRESS_FILE"
log "Check logs: $LOGS_DIR/"
log "============================================"
