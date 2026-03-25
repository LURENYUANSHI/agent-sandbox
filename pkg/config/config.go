package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// AppConfig is the top-level application configuration.
type AppConfig struct {
	Server   ServerConfig   `yaml:"server"`
	Sandbox  SandboxConfig  `yaml:"sandbox"`
	Executor ExecutorConfig `yaml:"executor"`
	Policy   PolicyConfig   `yaml:"policy"`
}

// ServerConfig holds API server settings.
type ServerConfig struct {
	Port        int      `yaml:"port"`
	AuthEnabled bool     `yaml:"auth_enabled"`
	CORSOrigins []string `yaml:"cors_origins"`
}

// SandboxConfig holds default sandbox creation parameters.
type SandboxConfig struct {
	DefaultRootDir     string `yaml:"default_root_dir"`
	DefaultTimeoutSec  int    `yaml:"default_timeout_sec"`
	DefaultMaxMemoryMB int    `yaml:"default_max_memory_mb"`
	DefaultMaxDiskMB   int    `yaml:"default_max_disk_mb"`
	DefaultMaxProcs    int    `yaml:"default_max_procs"`
	TracePath          string `yaml:"trace_path"`
}

// ExecutorConfig holds limits for executor operations.
type ExecutorConfig struct {
	MaxReadSizeMB     int `yaml:"max_read_size_mb"`
	MaxWriteSizeMB    int `yaml:"max_write_size_mb"`
	MaxResponseSizeMB int `yaml:"max_response_size_mb"`
	HTTPTimeoutSec    int `yaml:"http_timeout_sec"`
	TCPTimeoutSec     int `yaml:"tcp_timeout_sec"`
}

// PolicyConfig holds policy engine parameters.
type PolicyConfig struct {
	MaxFileSizeMB       int `yaml:"max_file_size_mb"`
	PrivilegedPortLimit int `yaml:"privileged_port_limit"`
}

// Default returns an AppConfig with sensible defaults.
func Default() *AppConfig {
	return &AppConfig{
		Server: ServerConfig{
			Port:        8080,
			AuthEnabled: false,
			CORSOrigins: []string{"http://localhost:3000", "http://localhost:5173"},
		},
		Sandbox: SandboxConfig{
			DefaultTimeoutSec:  300,
			DefaultMaxMemoryMB: 512,
			DefaultMaxDiskMB:   1024,
			DefaultMaxProcs:    10,
		},
		Executor: ExecutorConfig{
			MaxReadSizeMB:     10,
			MaxWriteSizeMB:    10,
			MaxResponseSizeMB: 5,
			HTTPTimeoutSec:    30,
			TCPTimeoutSec:     10,
		},
		Policy: PolicyConfig{
			MaxFileSizeMB:       100,
			PrivilegedPortLimit: 1024,
		},
	}
}

// LoadFromEnv loads configuration from environment variables, falling back to defaults.
func LoadFromEnv() *AppConfig {
	cfg := Default()

	if v := envInt("SANDBOX_SERVER_PORT"); v > 0 {
		cfg.Server.Port = v
	}
	if v := os.Getenv("SANDBOX_AUTH_ENABLED"); v == "true" {
		cfg.Server.AuthEnabled = true
	}
	if v := os.Getenv("SANDBOX_CORS_ORIGINS"); v != "" {
		cfg.Server.CORSOrigins = strings.Split(v, ",")
	}

	if v := os.Getenv("SANDBOX_DEFAULT_ROOT_DIR"); v != "" {
		cfg.Sandbox.DefaultRootDir = v
	}
	if v := envInt("SANDBOX_DEFAULT_TIMEOUT_SEC"); v > 0 {
		cfg.Sandbox.DefaultTimeoutSec = v
	}
	if v := envInt("SANDBOX_DEFAULT_MAX_MEMORY_MB"); v > 0 {
		cfg.Sandbox.DefaultMaxMemoryMB = v
	}
	if v := envInt("SANDBOX_DEFAULT_MAX_DISK_MB"); v > 0 {
		cfg.Sandbox.DefaultMaxDiskMB = v
	}
	if v := envInt("SANDBOX_DEFAULT_MAX_PROCS"); v > 0 {
		cfg.Sandbox.DefaultMaxProcs = v
	}
	if v := os.Getenv("SANDBOX_TRACE_PATH"); v != "" {
		cfg.Sandbox.TracePath = v
	}

	if v := envInt("SANDBOX_MAX_READ_SIZE_MB"); v > 0 {
		cfg.Executor.MaxReadSizeMB = v
	}
	if v := envInt("SANDBOX_MAX_WRITE_SIZE_MB"); v > 0 {
		cfg.Executor.MaxWriteSizeMB = v
	}
	if v := envInt("SANDBOX_MAX_RESPONSE_SIZE_MB"); v > 0 {
		cfg.Executor.MaxResponseSizeMB = v
	}
	if v := envInt("SANDBOX_HTTP_TIMEOUT_SEC"); v > 0 {
		cfg.Executor.HTTPTimeoutSec = v
	}
	if v := envInt("SANDBOX_TCP_TIMEOUT_SEC"); v > 0 {
		cfg.Executor.TCPTimeoutSec = v
	}

	if v := envInt("SANDBOX_MAX_FILE_SIZE_MB"); v > 0 {
		cfg.Policy.MaxFileSizeMB = v
	}
	if v := envInt("SANDBOX_PRIVILEGED_PORT_LIMIT"); v > 0 {
		cfg.Policy.PrivilegedPortLimit = v
	}

	return cfg
}

// LoadFromFile loads configuration from a YAML file, with defaults for unset fields.
func LoadFromFile(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	return cfg, nil
}

func envInt(key string) int {
	v := os.Getenv(key)
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return n
}
