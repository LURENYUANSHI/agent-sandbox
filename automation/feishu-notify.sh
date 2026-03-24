#!/bin/bash
# ============================================================================
# Feishu (Lark) Bot Notification Helper
# Sends messages to Feishu group via Bot API
# Usage: bash feishu-notify.sh <event_type> <title> <detail>
# ============================================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/secrets.env"

FEISHU_APP_ID="${FEISHU_APP_ID}"
FEISHU_APP_SECRET="${FEISHU_APP_SECRET}"

# Cache token to avoid hitting rate limits
TOKEN_CACHE_FILE="$SCRIPT_DIR/.feishu_token_cache"

# ---- Get tenant access token ----
get_token() {
    # Check cache (token valid for 2 hours, we refresh at 1.5h)
    if [ -f "$TOKEN_CACHE_FILE" ]; then
        local cached_time=$(head -1 "$TOKEN_CACHE_FILE")
        local cached_token=$(tail -1 "$TOKEN_CACHE_FILE")
        local now=$(date +%s)
        local age=$(( now - cached_time ))
        if [ $age -lt 5400 ]; then
            echo "$cached_token"
            return
        fi
    fi

    local response=$(curl -s -X POST "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal" \
        -H "Content-Type: application/json" \
        -d "{\"app_id\": \"$FEISHU_APP_ID\", \"app_secret\": \"$FEISHU_APP_SECRET\"}")

    local token=$(echo "$response" | python -c "import sys,json; print(json.load(sys.stdin).get('tenant_access_token',''))" 2>/dev/null)

    if [ -n "$token" ] && [ "$token" != "None" ]; then
        echo "$(date +%s)" > "$TOKEN_CACHE_FILE"
        echo "$token" >> "$TOKEN_CACHE_FILE"
        echo "$token"
    else
        echo "ERROR: Failed to get Feishu token. Response: $response" >&2
        echo ""
    fi
}

# ---- Send message to chat ----
# First we need to get the bot's chat list, then send to the first group
send_message() {
    local token=$(get_token)
    if [ -z "$token" ]; then
        echo "ERROR: No token available" >&2
        return 1
    fi

    local msg_type=$1  # event type: phase_start, phase_complete, issue_created, pr_created, etc.
    local title=$2
    local detail=$3
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')

    # Map event types to colors/emojis
    local emoji=""
    local color="blue"
    case "$msg_type" in
        phase_start)    emoji="🚀"; color="blue" ;;
        phase_complete) emoji="✅"; color="green" ;;
        phase_failed)   emoji="❌"; color="red" ;;
        phase_recovered) emoji="🔧"; color="orange" ;;
        issue_created)  emoji="📋"; color="purple" ;;
        pr_created)     emoji="🔀"; color="blue" ;;
        pr_merged)      emoji="🎉"; color="green" ;;
        build_start)    emoji="⚡"; color="blue" ;;
        build_complete) emoji="🏁"; color="green" ;;
        test_pass)      emoji="✅"; color="green" ;;
        test_fail)      emoji="❌"; color="red" ;;
        deploy)         emoji="🚢"; color="blue" ;;
        *)              emoji="📌"; color="gray" ;;
    esac

    # Build rich text message card
    local card_json=$(cat <<CARD_EOF
{
    "msg_type": "interactive",
    "card": {
        "config": {
            "wide_screen_mode": true
        },
        "header": {
            "title": {
                "tag": "plain_text",
                "content": "$emoji AgentSandbox: $title"
            },
            "template": "$color"
        },
        "elements": [
            {
                "tag": "div",
                "fields": [
                    {
                        "is_short": true,
                        "text": {
                            "tag": "lark_md",
                            "content": "**Event Type:** $msg_type"
                        }
                    },
                    {
                        "is_short": true,
                        "text": {
                            "tag": "lark_md",
                            "content": "**Time:** $timestamp"
                        }
                    }
                ]
            },
            {
                "tag": "div",
                "text": {
                    "tag": "lark_md",
                    "content": "$detail"
                }
            },
            {
                "tag": "hr"
            },
            {
                "tag": "note",
                "elements": [
                    {
                        "tag": "plain_text",
                        "content": "AgentSandbox Autonomous Dev Pipeline | github.com/LURENYUANSHI/agent-sandbox"
                    }
                ]
            }
        ]
    }
}
CARD_EOF
)

    # Get chat list and send to first available group
    local chat_id_response=$(curl -s -X GET "https://open.feishu.cn/open-apis/im/v1/chats?page_size=1" \
        -H "Authorization: Bearer $token")

    local chat_id=$(echo "$chat_id_response" | python -c "
import sys, json
data = json.load(sys.stdin)
items = data.get('data', {}).get('items', [])
if items:
    print(items[0]['chat_id'])
else:
    print('')
" 2>/dev/null)

    if [ -n "$chat_id" ] && [ "$chat_id" != "" ]; then
        # Send to group chat
        curl -s -X POST "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=chat_id" \
            -H "Authorization: Bearer $token" \
            -H "Content-Type: application/json" \
            -d "{\"receive_id\": \"$chat_id\", \"msg_type\": \"interactive\", \"content\": $(echo "$card_json" | python -c "import sys,json; print(json.dumps(json.loads(sys.stdin.read())['card']))")}" \
            > /dev/null 2>&1
        echo "Sent to chat: $chat_id"
    else
        # Fallback: try sending via webhook if no chat found
        echo "WARN: No chat found. Token may not have chat permissions." >&2
        echo "Chat list response: $chat_id_response" >&2
    fi
}

# ---- Convenience functions ----

notify_phase_start() {
    send_message "phase_start" "Phase Started: $1" "Phase **$1** has started execution.\n\nThis phase will: $2"
}

notify_phase_complete() {
    send_message "phase_complete" "Phase Completed: $1" "Phase **$1** completed successfully.\n\nDuration: $2"
}

notify_phase_failed() {
    send_message "phase_failed" "Phase Failed: $1" "Phase **$1** failed with exit code $2.\n\nAttempting automatic recovery..."
}

notify_issue_created() {
    send_message "issue_created" "Issue Created: #$1" "**$2**\n\n[View on GitHub](https://github.com/LURENYUANSHI/agent-sandbox/issues/$1)"
}

notify_pr_created() {
    send_message "pr_created" "PR Created: #$1" "**$2**\n\n[View on GitHub](https://github.com/LURENYUANSHI/agent-sandbox/pull/$1)"
}

notify_build_complete() {
    send_message "build_complete" "Build Complete" "All phases finished.\n\n$1"
}

# ---- Main: called directly with args ----
if [ $# -ge 2 ]; then
    send_message "$1" "$2" "${3:-No additional details}"
fi
