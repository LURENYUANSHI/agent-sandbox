#!/bin/bash
# Quick start script - run the overnight build immediately
# Usage: bash /c/Users/Administrator/ai-sandbox/automation/start-now.sh

echo "Starting AgentSandbox overnight build NOW..."
echo "Logs: /c/Users/Administrator/ai-sandbox/automation/logs/"
echo "Progress: /c/Users/Administrator/ai-sandbox/automation/progress.json"
echo ""

nohup bash /c/Users/Administrator/ai-sandbox/automation/run-overnight.sh \
    > /c/Users/Administrator/ai-sandbox/automation/logs/nohup.log 2>&1 &

echo "Build started in background (PID: $!)"
echo "To monitor: tail -f /c/Users/Administrator/ai-sandbox/automation/logs/orchestrator.log"
echo "To check progress: cat /c/Users/Administrator/ai-sandbox/automation/progress.json"
