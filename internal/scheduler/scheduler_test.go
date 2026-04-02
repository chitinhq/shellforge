package scheduler

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestServeConfigInferenceField(t *testing.T) {
	yamlInput := `
max_parallel: 0
log_dir: outputs/logs
model_ram_gb: 8
inference: remote
agents: []
`
	var cfg ServeConfig
	if err := yaml.Unmarshal([]byte(yamlInput), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cfg.Inference != "remote" {
		t.Errorf("inference = %q, want %q", cfg.Inference, "remote")
	}
}

func TestServeConfigInferenceDefaultsEmpty(t *testing.T) {
	yamlInput := `
max_parallel: 0
log_dir: outputs/logs
model_ram_gb: 8
agents: []
`
	var cfg ServeConfig
	if err := yaml.Unmarshal([]byte(yamlInput), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cfg.Inference != "" {
		t.Errorf("inference = %q, want empty", cfg.Inference)
	}
}

func TestNewSchedulerRemoteInferenceDefaultsToFour(t *testing.T) {
	cfg := &ServeConfig{
		MaxParallel: 0,
		Inference:   "remote",
		LogDir:      t.TempDir(),
		ModelRAM:    8,
	}
	var noopRun RunFunc = func(name, system, prompt string, timeoutSec int) error { return nil }
	sched := New(cfg, noopRun)
	if cap(sched.slots) != 4 {
		t.Errorf("remote inference: slots cap = %d, want 4", cap(sched.slots))
	}
}

func TestNewSchedulerRemoteInferenceRespectsExplicitMaxParallel(t *testing.T) {
	cfg := &ServeConfig{
		MaxParallel: 8,
		Inference:   "remote",
		LogDir:      t.TempDir(),
		ModelRAM:    8,
	}
	var noopRun RunFunc = func(name, system, prompt string, timeoutSec int) error { return nil }
	sched := New(cfg, noopRun)
	if cap(sched.slots) != 8 {
		t.Errorf("remote inference with explicit max_parallel: slots cap = %d, want 8", cap(sched.slots))
	}
}

func TestLoadConfigInferenceField(t *testing.T) {
	content := `
max_parallel: 0
log_dir: outputs/logs
model_ram_gb: 8
inference: remote
agents: []
`
	path := t.TempDir() + "/agents.yaml"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Inference != "remote" {
		t.Errorf("loaded inference = %q, want %q", cfg.Inference, "remote")
	}
}
