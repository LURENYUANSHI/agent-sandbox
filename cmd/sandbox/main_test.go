package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// buildRootCmd creates a fresh root command tree identical to main().
func buildRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "agent-sandbox",
		Short: "AI Agent Security Sandbox",
		Long:  "Runtime sandbox for AI agents with policy enforcement, trace recording, and replay.",
	}
	root.PersistentFlags().String("output", "text", "Output format: text or json")
	root.AddCommand(
		createCmd(),
		startCmd(),
		execCmd(),
		stopCmd(),
		listCmd(),
		traceCmd(),
		replayCmd(),
		policyCmd(),
		versionCmd(),
	)
	return root
}

// runCLI executes a CLI command and returns the captured output.
func runCLI(args ...string) (string, error) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := buildRootCmd()
	cmd.SetArgs(args)
	cmd.SetOut(w)
	cmd.SetErr(w)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		buf.ReadFrom(r)
		close(done)
	}()

	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout
	<-done

	return buf.String(), err
}

func resetRegistry() {
	sandboxRegistry.entries = make(map[string]*sandboxEntry)
}

func projectRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}

func TestCLIVersion(t *testing.T) {
	output, err := runCLI("version")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "agent-sandbox version") {
		t.Errorf("expected version header, got: %s", output)
	}
	if !strings.Contains(output, version) {
		t.Errorf("expected version %s in output: %s", version, output)
	}
}

func TestCLIHelp(t *testing.T) {
	output, err := runCLI("--help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, sub := range []string{"create", "start", "stop", "exec", "list", "trace", "replay", "policy", "version"} {
		if !strings.Contains(output, sub) {
			t.Errorf("help output missing subcommand %q", sub)
		}
	}
}

func TestCLIUnknownCommand(t *testing.T) {
	_, err := runCLI("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestCLIInvalidSandboxID(t *testing.T) {
	resetRegistry()
	_, err := runCLI("start", "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for invalid sandbox ID")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestCLISandboxLifecycle(t *testing.T) {
	resetRegistry()
	tmpDir := t.TempDir()
	sandboxDir := filepath.Join(tmpDir, "sandbox")

	// Create a permissive policy so exec actions are allowed
	policyYAML := "name: test-permissive\nversion: \"1.0\"\ndescription: Permissive test policy\ndefault_effect: allow\nrules: []\n"
	policyFile := filepath.Join(tmpDir, "policy.yaml")
	if err := os.WriteFile(policyFile, []byte(policyYAML), 0644); err != nil {
		t.Fatalf("write policy file: %v", err)
	}

	// Create sandbox root and a test file inside it
	if err := os.MkdirAll(sandboxDir, 0755); err != nil {
		t.Fatalf("create sandbox dir: %v", err)
	}
	testFile := filepath.Join(sandboxDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello sandbox"), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	var sandboxID string
	uuidRe := regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

	t.Run("create", func(t *testing.T) {
		output, err := runCLI("create", "--name", "test-sandbox", "--root", sandboxDir, "--policy", policyFile)
		if err != nil {
			t.Fatalf("create failed: %v", err)
		}
		if !strings.Contains(output, "Created sandbox") {
			t.Errorf("expected 'Created sandbox' in output: %s", output)
		}
		if !strings.Contains(output, "test-sandbox") {
			t.Errorf("expected sandbox name in output: %s", output)
		}
		match := uuidRe.FindString(output)
		if match == "" {
			t.Fatalf("no UUID found in output: %s", output)
		}
		sandboxID = match
	})

	if sandboxID == "" {
		t.Fatal("no sandbox ID from create, cannot continue")
	}

	t.Run("list", func(t *testing.T) {
		output, err := runCLI("list")
		if err != nil {
			t.Fatalf("list failed: %v", err)
		}
		if !strings.Contains(output, "test-sandbox") {
			t.Errorf("expected sandbox in list: %s", output)
		}
		if !strings.Contains(output, sandboxID) {
			t.Errorf("expected ID in list: %s", output)
		}
	})

	t.Run("list_json", func(t *testing.T) {
		output, err := runCLI("list", "--output", "json")
		if err != nil {
			t.Fatalf("list json failed: %v", err)
		}
		var items []map[string]interface{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &items); err != nil {
			t.Fatalf("invalid JSON: %v\nraw: %s", err, output)
		}
		if len(items) == 0 {
			t.Fatal("expected at least one item in JSON list")
		}
		found := false
		for _, item := range items {
			if item["id"] == sandboxID {
				found = true
			}
		}
		if !found {
			t.Errorf("sandbox %s not in JSON output", sandboxID)
		}
	})

	t.Run("start", func(t *testing.T) {
		output, err := runCLI("start", sandboxID)
		if err != nil {
			t.Fatalf("start failed: %v", err)
		}
		if !strings.Contains(output, "started") {
			t.Errorf("expected 'started' in output: %s", output)
		}
	})

	t.Run("exec", func(t *testing.T) {
		output, err := runCLI("exec", sandboxID, "file.read", "--param", "path=test.txt")
		if err != nil {
			t.Fatalf("exec failed: %v", err)
		}
		if !strings.Contains(output, "Action:") {
			t.Errorf("expected 'Action:' in output: %s", output)
		}
		if !strings.Contains(output, "Success:") {
			t.Errorf("expected 'Success:' in output: %s", output)
		}
	})

	t.Run("trace", func(t *testing.T) {
		output, err := runCLI("trace", sandboxID)
		if err != nil {
			t.Fatalf("trace failed: %v", err)
		}
		if !strings.Contains(output, "TYPE") {
			t.Errorf("expected trace table header: %s", output)
		}
	})

	t.Run("stop", func(t *testing.T) {
		output, err := runCLI("stop", sandboxID)
		if err != nil {
			t.Fatalf("stop failed: %v", err)
		}
		if !strings.Contains(output, "stopped") {
			t.Errorf("expected 'stopped' in output: %s", output)
		}
	})
}

func TestCLIPolicyValidate(t *testing.T) {
	policyPath := filepath.Join(projectRoot(), "configs", "default-policy.yaml")
	if _, err := os.Stat(policyPath); os.IsNotExist(err) {
		t.Skipf("policy file not found: %s", policyPath)
	}

	output, err := runCLI("policy", "validate", policyPath)
	if err != nil {
		t.Fatalf("policy validate failed: %v", err)
	}
	if !strings.Contains(output, "Valid policy") {
		t.Errorf("expected 'Valid policy' in output: %s", output)
	}
}

func TestCLIPolicyValidateInvalid(t *testing.T) {
	output, err := runCLI("policy", "validate", "/nonexistent/policy.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "Invalid policy") {
		t.Errorf("expected 'Invalid policy' in output: %s", output)
	}
}
