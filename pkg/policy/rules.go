package policy

import (
	"fmt"
	"net"
	"path"
	"strconv"
	"strings"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// BuiltinRule represents a safety rule that is always enforced regardless of user policy.
type BuiltinRule struct {
	ID   string
	Name string
	// Check returns a deny decision and true if the rule blocks the action,
	// or a zero-value decision and false if the rule does not apply.
	Check func(action types.Action) (types.PolicyDecision, bool)
}

// DefaultBuiltinRules returns the standard set of built-in safety rules.
func DefaultBuiltinRules() []BuiltinRule {
	const defaultMaxFileSize int64 = 100 * 1024 * 1024 // 100 MB

	return []BuiltinRule{
		noDeleteRoot(),
		noKillInit(),
		noDangerousCommands(),
		noPrivilegedPorts(),
		maxFileSizeLimit(defaultMaxFileSize),
		pathTraversalProtection(),
	}
}

func builtinDeny(ruleID, reason string) types.PolicyDecision {
	return types.PolicyDecision{
		Effect: types.EffectDeny,
		Reason: fmt.Sprintf("[builtin:%s] %s", ruleID, reason),
	}
}

// --- Individual rule implementations ---

// systemPaths that must never be deleted.
var systemPaths = []string{
	"/", "/etc", "/usr", "/bin", "/sbin", "/boot",
	"/sys", "/proc", "/dev", "/var", "/lib", "/root", "/home",
}

func noDeleteRoot() BuiltinRule {
	return BuiltinRule{
		ID:   "no-delete-root",
		Name: "Deny deletion of root and system paths",
		Check: func(action types.Action) (types.PolicyDecision, bool) {
			if action.Type != types.ActionFileDelete {
				return types.PolicyDecision{}, false
			}
			cleaned := path.Clean(action.Resource)
			for _, sp := range systemPaths {
				if cleaned == sp {
					return builtinDeny("no-delete-root",
						fmt.Sprintf("cannot delete system path %s", cleaned)), true
				}
			}
			return types.PolicyDecision{}, false
		},
	}
}

func noKillInit() BuiltinRule {
	return BuiltinRule{
		ID:   "no-kill-init",
		Name: "Deny killing PID 1",
		Check: func(action types.Action) (types.PolicyDecision, bool) {
			if action.Type != types.ActionProcKill {
				return types.PolicyDecision{}, false
			}
			if strings.TrimSpace(action.Resource) == "1" {
				return builtinDeny("no-kill-init", "cannot kill PID 1 (init)"), true
			}
			return types.PolicyDecision{}, false
		},
	}
}

// dangerousSubstrings are command fragments that are always blocked.
var dangerousSubstrings = []string{
	"mkfs",
	"dd if=/dev/zero",
	"dd if=/dev/random",
	":(){ :|:& };:",
}

func noDangerousCommands() BuiltinRule {
	return BuiltinRule{
		ID:   "no-dangerous-commands",
		Name: "Deny dangerous shell commands",
		Check: func(action types.Action) (types.PolicyDecision, bool) {
			if action.Type != types.ActionShellExec && action.Type != types.ActionProcExec {
				return types.PolicyDecision{}, false
			}
			cmd := action.Resource

			// Check for known dangerous substrings.
			for _, ds := range dangerousSubstrings {
				if strings.Contains(cmd, ds) {
					return builtinDeny("no-dangerous-commands",
						fmt.Sprintf("command contains dangerous pattern: %s", ds)), true
				}
			}

			// Check for destructive rm targeting root / system paths.
			if isDestructiveRm(cmd) {
				return builtinDeny("no-dangerous-commands",
					"rm with force+recursive targeting system path"), true
			}

			return types.PolicyDecision{}, false
		},
	}
}

func isDestructiveRm(cmd string) bool {
	fields := strings.Fields(cmd)
	if len(fields) == 0 || fields[0] != "rm" {
		return false
	}

	hasForceRecursive := false
	for _, f := range fields {
		if len(f) > 1 && f[0] == '-' && strings.ContainsRune(f, 'r') && strings.ContainsRune(f, 'f') {
			hasForceRecursive = true
			break
		}
	}
	if !hasForceRecursive {
		return false
	}

	dangerous := map[string]bool{
		"/": true, "/*": true,
		"/etc": true, "/usr": true, "/bin": true, "/sbin": true,
		"/boot": true, "/sys": true, "/proc": true, "/dev": true,
		"/var": true, "/lib": true, "/root": true, "/home": true,
	}
	for _, f := range fields {
		if dangerous[f] {
			return true
		}
	}
	return false
}

func noPrivilegedPorts() BuiltinRule {
	return BuiltinRule{
		ID:   "no-privileged-ports",
		Name: "Deny listening on privileged ports (< 1024)",
		Check: func(action types.Action) (types.PolicyDecision, bool) {
			if action.Type != types.ActionNetListen {
				return types.PolicyDecision{}, false
			}
			port, ok := parsePort(action.Resource)
			if !ok {
				return types.PolicyDecision{}, false
			}
			if port > 0 && port < 1024 {
				return builtinDeny("no-privileged-ports",
					fmt.Sprintf("cannot listen on privileged port %d", port)), true
			}
			return types.PolicyDecision{}, false
		},
	}
}

func parsePort(resource string) (int, bool) {
	// Try as plain number.
	if port, err := strconv.Atoi(resource); err == nil {
		return port, true
	}
	// Try as host:port.
	_, portStr, err := net.SplitHostPort(resource)
	if err != nil {
		return 0, false
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, false
	}
	return port, true
}

func maxFileSizeLimit(maxSize int64) BuiltinRule {
	return BuiltinRule{
		ID:   "max-file-size",
		Name: "Deny file writes exceeding size limit",
		Check: func(action types.Action) (types.PolicyDecision, bool) {
			if action.Type != types.ActionFileWrite {
				return types.PolicyDecision{}, false
			}
			size, ok := metadataInt64(action.Metadata, "size")
			if !ok {
				return types.PolicyDecision{}, false
			}
			if size > maxSize {
				return builtinDeny("max-file-size",
					fmt.Sprintf("file size %d exceeds limit %d", size, maxSize)), true
			}
			return types.PolicyDecision{}, false
		},
	}
}

func metadataInt64(m map[string]interface{}, key string) (int64, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch s := v.(type) {
	case int64:
		return s, true
	case int:
		return int64(s), true
	case float64:
		return int64(s), true
	default:
		return 0, false
	}
}

func pathTraversalProtection() BuiltinRule {
	return BuiltinRule{
		ID:   "path-traversal-protection",
		Name: "Deny path traversal attempts",
		Check: func(action types.Action) (types.PolicyDecision, bool) {
			r := action.Resource
			if strings.Contains(r, "../") || strings.Contains(r, "..\\") ||
				r == ".." || strings.HasSuffix(r, "/..") {
				return builtinDeny("path-traversal-protection",
					"resource path contains traversal sequence (..)"), true
			}
			return types.PolicyDecision{}, false
		},
	}
}
