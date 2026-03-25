package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Executor.MaxReadSizeMB != 10 {
		t.Errorf("expected default max read 10, got %d", cfg.Executor.MaxReadSizeMB)
	}
	if cfg.Executor.MaxWriteSizeMB != 10 {
		t.Errorf("expected default max write 10, got %d", cfg.Executor.MaxWriteSizeMB)
	}
	if cfg.Executor.MaxResponseSizeMB != 5 {
		t.Errorf("expected default max response 5, got %d", cfg.Executor.MaxResponseSizeMB)
	}
	if cfg.Executor.HTTPTimeoutSec != 30 {
		t.Errorf("expected default http timeout 30, got %d", cfg.Executor.HTTPTimeoutSec)
	}
	if cfg.Executor.TCPTimeoutSec != 10 {
		t.Errorf("expected default tcp timeout 10, got %d", cfg.Executor.TCPTimeoutSec)
	}
	if cfg.Policy.MaxFileSizeMB != 100 {
		t.Errorf("expected default max file size 100, got %d", cfg.Policy.MaxFileSizeMB)
	}
	if cfg.Policy.PrivilegedPortLimit != 1024 {
		t.Errorf("expected default port limit 1024, got %d", cfg.Policy.PrivilegedPortLimit)
	}
	if len(cfg.Server.CORSOrigins) != 2 {
		t.Errorf("expected 2 default CORS origins, got %d", len(cfg.Server.CORSOrigins))
	}
}

func TestLoadFromEnv_Defaults(t *testing.T) {
	cfg := LoadFromEnv()

	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Executor.MaxReadSizeMB != 10 {
		t.Errorf("expected default max read 10, got %d", cfg.Executor.MaxReadSizeMB)
	}
	if cfg.Policy.MaxFileSizeMB != 100 {
		t.Errorf("expected default max file size 100, got %d", cfg.Policy.MaxFileSizeMB)
	}
}

func TestLoadFromEnv_Overrides(t *testing.T) {
	t.Setenv("SANDBOX_SERVER_PORT", "9090")
	t.Setenv("SANDBOX_MAX_READ_SIZE_MB", "20")
	t.Setenv("SANDBOX_HTTP_TIMEOUT_SEC", "60")
	t.Setenv("SANDBOX_MAX_FILE_SIZE_MB", "200")
	t.Setenv("SANDBOX_PRIVILEGED_PORT_LIMIT", "2048")
	t.Setenv("SANDBOX_CORS_ORIGINS", "http://example.com,http://other.com")
	t.Setenv("SANDBOX_AUTH_ENABLED", "true")
	t.Setenv("SANDBOX_DEFAULT_ROOT_DIR", "/tmp/sandbox")
	t.Setenv("SANDBOX_TRACE_PATH", "/tmp/traces")

	cfg := LoadFromEnv()

	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
	if !cfg.Server.AuthEnabled {
		t.Error("expected auth enabled")
	}
	if len(cfg.Server.CORSOrigins) != 2 || cfg.Server.CORSOrigins[0] != "http://example.com" {
		t.Errorf("unexpected CORS origins: %v", cfg.Server.CORSOrigins)
	}
	if cfg.Sandbox.DefaultRootDir != "/tmp/sandbox" {
		t.Errorf("expected root dir /tmp/sandbox, got %s", cfg.Sandbox.DefaultRootDir)
	}
	if cfg.Sandbox.TracePath != "/tmp/traces" {
		t.Errorf("expected trace path /tmp/traces, got %s", cfg.Sandbox.TracePath)
	}
	if cfg.Executor.MaxReadSizeMB != 20 {
		t.Errorf("expected max read 20, got %d", cfg.Executor.MaxReadSizeMB)
	}
	if cfg.Executor.HTTPTimeoutSec != 60 {
		t.Errorf("expected http timeout 60, got %d", cfg.Executor.HTTPTimeoutSec)
	}
	if cfg.Policy.MaxFileSizeMB != 200 {
		t.Errorf("expected max file size 200, got %d", cfg.Policy.MaxFileSizeMB)
	}
	if cfg.Policy.PrivilegedPortLimit != 2048 {
		t.Errorf("expected port limit 2048, got %d", cfg.Policy.PrivilegedPortLimit)
	}
}

func TestLoadFromEnv_InvalidValues(t *testing.T) {
	t.Setenv("SANDBOX_SERVER_PORT", "not-a-number")
	t.Setenv("SANDBOX_MAX_READ_SIZE_MB", "bad")

	cfg := LoadFromEnv()

	// Should fall back to defaults
	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080 for invalid env, got %d", cfg.Server.Port)
	}
	if cfg.Executor.MaxReadSizeMB != 10 {
		t.Errorf("expected default max read 10 for invalid env, got %d", cfg.Executor.MaxReadSizeMB)
	}
}

func TestLoadFromFile(t *testing.T) {
	yamlContent := `
server:
  port: 9999
  auth_enabled: true
  cors_origins:
    - "http://custom.com"
executor:
  max_read_size_mb: 50
  http_timeout_sec: 120
policy:
  max_file_size_mb: 500
  privileged_port_limit: 512
`
	dir := t.TempDir()
	path := filepath.Join(dir, "test-config.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	if cfg.Server.Port != 9999 {
		t.Errorf("expected port 9999, got %d", cfg.Server.Port)
	}
	if !cfg.Server.AuthEnabled {
		t.Error("expected auth enabled")
	}
	if len(cfg.Server.CORSOrigins) != 1 || cfg.Server.CORSOrigins[0] != "http://custom.com" {
		t.Errorf("unexpected CORS origins: %v", cfg.Server.CORSOrigins)
	}
	if cfg.Executor.MaxReadSizeMB != 50 {
		t.Errorf("expected max read 50, got %d", cfg.Executor.MaxReadSizeMB)
	}
	if cfg.Executor.HTTPTimeoutSec != 120 {
		t.Errorf("expected http timeout 120, got %d", cfg.Executor.HTTPTimeoutSec)
	}
	// Fields not set in YAML should have defaults
	if cfg.Executor.MaxWriteSizeMB != 10 {
		t.Errorf("expected default max write 10, got %d", cfg.Executor.MaxWriteSizeMB)
	}
	if cfg.Executor.TCPTimeoutSec != 10 {
		t.Errorf("expected default tcp timeout 10, got %d", cfg.Executor.TCPTimeoutSec)
	}
	if cfg.Policy.MaxFileSizeMB != 500 {
		t.Errorf("expected max file size 500, got %d", cfg.Policy.MaxFileSizeMB)
	}
	if cfg.Policy.PrivilegedPortLimit != 512 {
		t.Errorf("expected port limit 512, got %d", cfg.Policy.PrivilegedPortLimit)
	}
}

func TestLoadFromFile_NotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadFromFile_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("server:\n  port: [invalid"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := LoadFromFile(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}
