// Package scheduler provides memory-aware agent scheduling for ShellForge.
// It serializes agent runs against a single Ollama model, preventing OOM
// from concurrent KV cache allocation.
package scheduler

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

// AgentConfig defines a scheduled agent.
type AgentConfig struct {
	Name     string `yaml:"name"`
	System   string `yaml:"system"`
	Prompt   string `yaml:"prompt"`
	Schedule string `yaml:"schedule"` // cron: "*/30 * * * *" or interval: "5m", "1h"
	Priority int    `yaml:"priority"` // higher = runs first when queued
	Timeout  int    `yaml:"timeout"`  // seconds, default 300
	Enabled  bool   `yaml:"enabled"`
}

// ServeConfig is the top-level agents.yaml format.
type ServeConfig struct {
	MaxParallel int           `yaml:"max_parallel"` // 0 = auto-detect
	LogDir      string        `yaml:"log_dir"`
	ModelRAM    int           `yaml:"model_ram_gb"` // estimated model RAM in GB, default 19
	Agents      []AgentConfig `yaml:"agents"`
}

// RunFunc is called to execute an agent. Matches agent.RunLoop signature pattern.
type RunFunc func(name, system, prompt string, timeoutSec int) error

// Scheduler manages agent execution with concurrency control.
type Scheduler struct {
	config  ServeConfig
	slots   chan struct{}
	stop    chan struct{}
	wg      sync.WaitGroup
	runFunc RunFunc
	logDir  string
}

// LoadConfig reads and parses an agents.yaml file.
func LoadConfig(path string) (*ServeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg ServeConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.LogDir == "" {
		cfg.LogDir = "outputs/logs"
	}
	if cfg.ModelRAM == 0 {
		cfg.ModelRAM = 19 // qwen3:30b Q4 default
	}
	return &cfg, nil
}

// DetectMaxParallel calculates how many agents can run in parallel
// based on available system RAM and estimated model size.
func DetectMaxParallel(modelRAMGB int) int {
	totalBytes := detectTotalRAM()
	if totalBytes == 0 {
		return 1 // can't detect, be safe
	}
	totalGB := int(totalBytes / (1024 * 1024 * 1024))
	// Each parallel KV cache slot ~1.5GB for 4K context at f16.
	// With q8_0 KV quantization: ~0.75GB per slot.
	kvPerSlot := 2 // conservative: 2GB per slot
	freeGB := totalGB - modelRAMGB - 4 // reserve 4GB for OS
	if freeGB < kvPerSlot {
		return 1
	}
	maxSlots := freeGB / kvPerSlot
	if maxSlots > 4 {
		maxSlots = 4 // cap at 4 — diminishing returns on single GPU
	}
	fmt.Printf("[scheduler] detected %dGB RAM, model ~%dGB, free ~%dGB → max_parallel=%d\n",
		totalGB, modelRAMGB, freeGB, maxSlots)
	return maxSlots
}

func detectTotalRAM() uint64 {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err != nil {
			return 0
		}
		val, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
		if err != nil {
			return 0
		}
		return val
	case "linux":
		data, err := os.ReadFile("/proc/meminfo")
		if err != nil {
			return 0
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					kb, err := strconv.ParseUint(fields[1], 10, 64)
					if err == nil {
						return kb * 1024
					}
				}
			}
		}
		return 0
	default:
		return 0
	}
}

// New creates a scheduler from config.
func New(cfg *ServeConfig, run RunFunc) *Scheduler {
	maxP := cfg.MaxParallel
	if maxP <= 0 {
		maxP = DetectMaxParallel(cfg.ModelRAM)
	}
	os.MkdirAll(cfg.LogDir, 0o755)
	return &Scheduler{
		config:  *cfg,
		slots:   make(chan struct{}, maxP),
		stop:    make(chan struct{}),
		runFunc: run,
		logDir:  cfg.LogDir,
	}
}

// Start begins scheduling all enabled agents.
func (s *Scheduler) Start() {
	enabled := 0
	for _, a := range s.config.Agents {
		if !a.Enabled {
			continue
		}
		enabled++
		agent := a // capture
		interval := parseInterval(agent.Schedule)
		if interval == 0 {
			fmt.Printf("[scheduler] ⚠ %s: invalid schedule %q, skipping\n", agent.Name, agent.Schedule)
			continue
		}
		fmt.Printf("[scheduler] ✓ %s: every %s (priority %d, timeout %ds)\n",
			agent.Name, interval, agent.Priority, agent.Timeout)
		s.wg.Add(1)
		go s.runLoop(agent, interval)
	}
	fmt.Printf("[scheduler] started %d agents, max_parallel=%d\n", enabled, cap(s.slots))
}

// Wait blocks until shutdown signal, then drains.
func (s *Scheduler) Wait() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	fmt.Println("\n[scheduler] shutting down — waiting for active agents...")
	close(s.stop)
	s.wg.Wait()
	fmt.Println("[scheduler] all agents stopped")
}

func (s *Scheduler) runLoop(agent AgentConfig, interval time.Duration) {
	defer s.wg.Done()

	// Run immediately on startup, then on interval
	s.executeAgent(agent)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.executeAgent(agent)
		}
	}
}

func (s *Scheduler) executeAgent(agent AgentConfig) {
	// Check if shutting down
	select {
	case <-s.stop:
		return
	default:
	}

	// Acquire slot (blocks if all slots busy)
	select {
	case s.slots <- struct{}{}:
	case <-s.stop:
		return
	}
	defer func() { <-s.slots }()

	timeout := agent.Timeout
	if timeout == 0 {
		timeout = 300
	}

	ts := time.Now().Format("2006-01-02T15-04-05")
	logPath := filepath.Join(s.logDir, fmt.Sprintf("%s-%s.log", agent.Name, ts))

	fmt.Printf("[scheduler] %s started (slot %d/%d)\n",
		agent.Name, len(s.slots), cap(s.slots))

	start := time.Now()
	err := s.runFunc(agent.Name, agent.System, agent.Prompt, timeout)
	elapsed := time.Since(start)

	status := "✓"
	errMsg := ""
	if err != nil {
		status = "✗"
		errMsg = fmt.Sprintf(" — error: %s", err)
	}
	fmt.Printf("[scheduler] %s %s finished (%s)%s\n",
		agent.Name, status, elapsed.Round(time.Second), errMsg)

	// Write run log
	logContent := fmt.Sprintf("[%s] %s %s (%s)%s\n",
		time.Now().Format(time.RFC3339), agent.Name, status, elapsed.Round(time.Second), errMsg)
	if err := os.WriteFile(logPath, []byte(logContent), 0o644); err != nil {
		fmt.Printf("[scheduler] ⚠ %s: failed to write log: %s\n", agent.Name, err)
	}
}

// parseInterval converts schedule strings to durations.
// Supports: "5m", "1h", "30s" (Go duration) and simple cron-like "*/N * * * *".
func parseInterval(schedule string) time.Duration {
	// Try Go duration first: "5m", "1h", "30m"
	if d, err := time.ParseDuration(schedule); err == nil && d > 0 {
		return d
	}

	// Simple cron: "*/N * * * *" → every N minutes
	parts := strings.Fields(schedule)
	if len(parts) == 5 && strings.HasPrefix(parts[0], "*/") {
		n, err := strconv.Atoi(parts[0][2:])
		if err == nil && n > 0 {
			return time.Duration(n) * time.Minute
		}
	}

	// "N * * * *" → every hour at minute N (treat as 1h interval)
	if len(parts) == 5 {
		// Any 5-field cron → default to 1h interval
		// Full cron parsing is a future enhancement
		return 1 * time.Hour
	}

	return 0
}
