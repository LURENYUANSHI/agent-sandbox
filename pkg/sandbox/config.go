package sandbox

import (
	"fmt"
)

// Config defines the sandbox environment parameters.
type Config struct {
	ID             string            `json:"id" yaml:"id"`
	Name           string            `json:"name" yaml:"name"`
	RootDir        string            `json:"root_dir" yaml:"root_dir"`
	PolicyFile     string            `json:"policy_file" yaml:"policy_file"`
	MaxMemoryMB    int               `json:"max_memory_mb" yaml:"max_memory_mb"`
	MaxCPUPercent  int               `json:"max_cpu_percent" yaml:"max_cpu_percent"`
	MaxDiskMB      int               `json:"max_disk_mb" yaml:"max_disk_mb"`
	MaxProcesses   int               `json:"max_processes" yaml:"max_processes"`
	TimeoutSeconds int               `json:"timeout_seconds" yaml:"timeout_seconds"`
	NetworkEnabled bool              `json:"network_enabled" yaml:"network_enabled"`
	AllowedPaths   []string          `json:"allowed_paths" yaml:"allowed_paths"`
	DeniedPaths    []string          `json:"denied_paths" yaml:"denied_paths"`
	Environment    map[string]string `json:"environment" yaml:"environment"`
	TraceEnabled   bool              `json:"trace_enabled" yaml:"trace_enabled"`
	TracePath      string            `json:"trace_path" yaml:"trace_path"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxMemoryMB:    512,
		MaxCPUPercent:  50,
		MaxDiskMB:      1024,
		MaxProcesses:   10,
		TimeoutSeconds: 30,
		NetworkEnabled: false,
		TraceEnabled:   true,
		Environment:    make(map[string]string),
	}
}

// Validate checks that the configuration is usable.
func (c *Config) Validate() error {
	if c.RootDir == "" {
		return fmt.Errorf("root_dir is required")
	}
	if c.MaxMemoryMB <= 0 {
		return fmt.Errorf("max_memory_mb must be positive, got %d", c.MaxMemoryMB)
	}
	if c.MaxCPUPercent <= 0 || c.MaxCPUPercent > 100 {
		return fmt.Errorf("max_cpu_percent must be 1-100, got %d", c.MaxCPUPercent)
	}
	if c.TimeoutSeconds <= 0 {
		return fmt.Errorf("timeout_seconds must be positive, got %d", c.TimeoutSeconds)
	}
	if c.MaxProcesses <= 0 {
		return fmt.Errorf("max_processes must be positive, got %d", c.MaxProcesses)
	}
	if c.TraceEnabled && c.TracePath == "" {
		return fmt.Errorf("trace_path is required when trace_enabled is true")
	}
	return nil
}
