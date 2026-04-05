package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

// Config holds ShellForge configuration, loaded from multiple sources.
type Config struct {
	Provider       string            `json:"provider,omitempty"`        // "anthropic", "openai", "ollama"
	Model          string            `json:"model,omitempty"`
	BaseURL        string            `json:"base_url,omitempty"`
	APIKey         string            `json:"api_key,omitempty"`         // only from env, never persisted
	MaxTurns       int               `json:"max_turns,omitempty"`
	TimeoutMs      int               `json:"timeout_ms,omitempty"`
	TokenBudget    int               `json:"token_budget,omitempty"`
	ThinkingBudget int               `json:"thinking_budget,omitempty"`
	GovernancePath string            `json:"governance_path,omitempty"` // path to agentguard.yaml
	OutputDir      string            `json:"output_dir,omitempty"`
	SessionDir     string            `json:"session_dir,omitempty"`
	Extra          map[string]string `json:"extra,omitempty"`           // arbitrary key-value pairs
}

// Defaults returns the default configuration.
func Defaults() Config {
	return Config{
		Provider:    "ollama",
		MaxTurns:    15,
		TimeoutMs:   180000,
		TokenBudget: 3000,
	}
}

// Load reads configuration from the three-level hierarchy and merges them.
// Priority (highest wins): local > project > user > defaults.
//
// Paths:
//   - User:    ~/.shellforge/config.json
//   - Project: .shellforge/config.json (relative to CWD)
//   - Local:   .shellforge.local.json (relative to CWD, gitignored)
//
// Then environment variables override all file-based config:
//   - SHELLFORGE_PROVIDER, SHELLFORGE_MODEL, SHELLFORGE_BASE_URL
//   - DEEPSEEK_API_KEY, OPENAI_API_KEY, ANTHROPIC_API_KEY (provider-specific)
//   - SHELLFORGE_MAX_TURNS, SHELLFORGE_TIMEOUT_MS, SHELLFORGE_TOKEN_BUDGET
func Load() *Config {
	cfg := Defaults()

	// User-level config: ~/.shellforge/config.json
	if home, err := os.UserHomeDir(); err == nil {
		userPath := filepath.Join(home, ".shellforge", "config.json")
		cfg = merge(cfg, loadFile(userPath))
	}

	// Project-level config: .shellforge/config.json (relative to CWD)
	projectPath := filepath.Join(".shellforge", "config.json")
	cfg = merge(cfg, loadFile(projectPath))

	// Local overrides: .shellforge.local.json (relative to CWD, gitignored)
	localPath := ".shellforge.local.json"
	cfg = merge(cfg, loadFile(localPath))

	// Environment variables override everything
	applyEnv(&cfg)

	return &cfg
}

// loadFile reads and unmarshals a JSON config file. Returns zero Config if file doesn't exist.
func loadFile(path string) Config {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, &cfg)
	// Strip api_key from file-loaded config for security
	cfg.APIKey = ""
	return cfg
}

// merge applies overrides from src onto dst. Only non-zero values in src override dst.
func merge(dst, src Config) Config {
	if src.Provider != "" {
		dst.Provider = src.Provider
	}
	if src.Model != "" {
		dst.Model = src.Model
	}
	if src.BaseURL != "" {
		dst.BaseURL = src.BaseURL
	}
	if src.APIKey != "" {
		dst.APIKey = src.APIKey
	}
	if src.MaxTurns != 0 {
		dst.MaxTurns = src.MaxTurns
	}
	if src.TimeoutMs != 0 {
		dst.TimeoutMs = src.TimeoutMs
	}
	if src.TokenBudget != 0 {
		dst.TokenBudget = src.TokenBudget
	}
	if src.ThinkingBudget != 0 {
		dst.ThinkingBudget = src.ThinkingBudget
	}
	if src.GovernancePath != "" {
		dst.GovernancePath = src.GovernancePath
	}
	if src.OutputDir != "" {
		dst.OutputDir = src.OutputDir
	}
	if src.SessionDir != "" {
		dst.SessionDir = src.SessionDir
	}

	// Extra maps are merged key-by-key, not replaced wholesale
	if len(src.Extra) > 0 {
		if dst.Extra == nil {
			dst.Extra = make(map[string]string)
		}
		for k, v := range src.Extra {
			dst.Extra[k] = v
		}
	}

	return dst
}

// applyEnv applies environment variable overrides.
func applyEnv(cfg *Config) {
	if v := os.Getenv("SHELLFORGE_PROVIDER"); v != "" {
		cfg.Provider = v
	}
	if v := os.Getenv("SHELLFORGE_MODEL"); v != "" {
		cfg.Model = v
	}
	if v := os.Getenv("SHELLFORGE_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}

	// Provider-specific API keys (check in order of provider preference)
	switch cfg.Provider {
	case "anthropic":
		if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
			cfg.APIKey = v
		}
	case "openai":
		if v := os.Getenv("OPENAI_API_KEY"); v != "" {
			cfg.APIKey = v
		}
	case "deepseek":
		if v := os.Getenv("DEEPSEEK_API_KEY"); v != "" {
			cfg.APIKey = v
		}
	}
	// Also check generic fallbacks regardless of provider
	if cfg.APIKey == "" {
		for _, key := range []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "DEEPSEEK_API_KEY"} {
			if v := os.Getenv(key); v != "" {
				cfg.APIKey = v
				break
			}
		}
	}

	if v := os.Getenv("SHELLFORGE_MAX_TURNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MaxTurns = n
		}
	}
	if v := os.Getenv("SHELLFORGE_TIMEOUT_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.TimeoutMs = n
		}
	}
	if v := os.Getenv("SHELLFORGE_TOKEN_BUDGET"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.TokenBudget = n
		}
	}
}
