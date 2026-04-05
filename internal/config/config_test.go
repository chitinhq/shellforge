package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	d := Defaults()
	if d.Provider != "ollama" {
		t.Errorf("Provider = %q, want %q", d.Provider, "ollama")
	}
	if d.MaxTurns != 15 {
		t.Errorf("MaxTurns = %d, want %d", d.MaxTurns, 15)
	}
	if d.TimeoutMs != 180000 {
		t.Errorf("TimeoutMs = %d, want %d", d.TimeoutMs, 180000)
	}
	if d.TokenBudget != 3000 {
		t.Errorf("TokenBudget = %d, want %d", d.TokenBudget, 3000)
	}
	if d.Model != "" {
		t.Errorf("Model = %q, want empty", d.Model)
	}
	if d.APIKey != "" {
		t.Errorf("APIKey = %q, want empty", d.APIKey)
	}
}

func TestLoadFileValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := Config{
		Provider: "anthropic",
		Model:    "claude-3-opus",
		MaxTurns: 25,
		Extra:    map[string]string{"foo": "bar"},
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got := loadFile(path)
	if got.Provider != "anthropic" {
		t.Errorf("Provider = %q, want %q", got.Provider, "anthropic")
	}
	if got.Model != "claude-3-opus" {
		t.Errorf("Model = %q, want %q", got.Model, "claude-3-opus")
	}
	if got.MaxTurns != 25 {
		t.Errorf("MaxTurns = %d, want %d", got.MaxTurns, 25)
	}
	if got.Extra["foo"] != "bar" {
		t.Errorf("Extra[foo] = %q, want %q", got.Extra["foo"], "bar")
	}
}

func TestLoadFileStripsAPIKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// Write a config file that contains an api_key (should be stripped)
	data := []byte(`{"api_key": "sk-secret-from-file", "provider": "anthropic"}`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got := loadFile(path)
	if got.APIKey != "" {
		t.Errorf("APIKey = %q, want empty (should be stripped from file)", got.APIKey)
	}
	if got.Provider != "anthropic" {
		t.Errorf("Provider = %q, want %q", got.Provider, "anthropic")
	}
}

func TestLoadFileMissing(t *testing.T) {
	got := loadFile("/nonexistent/path/config.json")
	if got.Provider != "" {
		t.Errorf("Provider = %q, want empty", got.Provider)
	}
	if got.Model != "" {
		t.Errorf("Model = %q, want empty", got.Model)
	}
	if got.MaxTurns != 0 {
		t.Errorf("MaxTurns = %d, want 0", got.MaxTurns)
	}
	if got.Extra != nil {
		t.Errorf("Extra = %v, want nil", got.Extra)
	}
}

func TestMergeNonZeroOverrides(t *testing.T) {
	dst := Config{
		Provider:    "ollama",
		Model:       "llama3",
		MaxTurns:    15,
		TokenBudget: 3000,
	}
	src := Config{
		Provider: "anthropic",
		Model:    "claude-3-opus",
		MaxTurns: 30,
	}

	got := merge(dst, src)
	if got.Provider != "anthropic" {
		t.Errorf("Provider = %q, want %q", got.Provider, "anthropic")
	}
	if got.Model != "claude-3-opus" {
		t.Errorf("Model = %q, want %q", got.Model, "claude-3-opus")
	}
	if got.MaxTurns != 30 {
		t.Errorf("MaxTurns = %d, want %d", got.MaxTurns, 30)
	}
	// TokenBudget was zero in src, should keep dst value
	if got.TokenBudget != 3000 {
		t.Errorf("TokenBudget = %d, want %d", got.TokenBudget, 3000)
	}
}

func TestMergeZeroFieldsPreserved(t *testing.T) {
	dst := Config{
		Provider:    "ollama",
		Model:       "llama3",
		MaxTurns:    15,
		TimeoutMs:   180000,
		TokenBudget: 3000,
		OutputDir:   "/tmp/output",
		SessionDir:  "/tmp/sessions",
	}
	src := Config{} // all zero values

	got := merge(dst, src)
	if got.Provider != "ollama" {
		t.Errorf("Provider = %q, want %q", got.Provider, "ollama")
	}
	if got.Model != "llama3" {
		t.Errorf("Model = %q, want %q", got.Model, "llama3")
	}
	if got.MaxTurns != 15 {
		t.Errorf("MaxTurns = %d, want %d", got.MaxTurns, 15)
	}
	if got.TimeoutMs != 180000 {
		t.Errorf("TimeoutMs = %d, want %d", got.TimeoutMs, 180000)
	}
	if got.TokenBudget != 3000 {
		t.Errorf("TokenBudget = %d, want %d", got.TokenBudget, 3000)
	}
	if got.OutputDir != "/tmp/output" {
		t.Errorf("OutputDir = %q, want %q", got.OutputDir, "/tmp/output")
	}
}

func TestMergeExtraMapsKeyByKey(t *testing.T) {
	dst := Config{
		Extra: map[string]string{"a": "1", "b": "2"},
	}
	src := Config{
		Extra: map[string]string{"b": "override", "c": "3"},
	}

	got := merge(dst, src)
	if got.Extra["a"] != "1" {
		t.Errorf("Extra[a] = %q, want %q", got.Extra["a"], "1")
	}
	if got.Extra["b"] != "override" {
		t.Errorf("Extra[b] = %q, want %q", got.Extra["b"], "override")
	}
	if got.Extra["c"] != "3" {
		t.Errorf("Extra[c] = %q, want %q", got.Extra["c"], "3")
	}
}

func TestMergeExtraIntoNilMap(t *testing.T) {
	dst := Config{}
	src := Config{
		Extra: map[string]string{"key": "val"},
	}

	got := merge(dst, src)
	if got.Extra == nil {
		t.Fatal("Extra is nil, want initialized map")
	}
	if got.Extra["key"] != "val" {
		t.Errorf("Extra[key] = %q, want %q", got.Extra["key"], "val")
	}
}

func TestLoadNoConfigFiles(t *testing.T) {
	// Use a temp dir as CWD with no config files, and override HOME
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origDir)

	// Clear all env vars that Load checks
	t.Setenv("SHELLFORGE_PROVIDER", "")
	t.Setenv("SHELLFORGE_MODEL", "")
	t.Setenv("SHELLFORGE_BASE_URL", "")
	t.Setenv("SHELLFORGE_MAX_TURNS", "")
	t.Setenv("SHELLFORGE_TIMEOUT_MS", "")
	t.Setenv("SHELLFORGE_TOKEN_BUDGET", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("DEEPSEEK_API_KEY", "")

	cfg := Load()
	defaults := Defaults()
	if cfg.Provider != defaults.Provider {
		t.Errorf("Provider = %q, want %q", cfg.Provider, defaults.Provider)
	}
	if cfg.MaxTurns != defaults.MaxTurns {
		t.Errorf("MaxTurns = %d, want %d", cfg.MaxTurns, defaults.MaxTurns)
	}
	if cfg.TimeoutMs != defaults.TimeoutMs {
		t.Errorf("TimeoutMs = %d, want %d", cfg.TimeoutMs, defaults.TimeoutMs)
	}
	if cfg.TokenBudget != defaults.TokenBudget {
		t.Errorf("TokenBudget = %d, want %d", cfg.TokenBudget, defaults.TokenBudget)
	}
}

func TestLoadUserConfigOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// Create user config
	userDir := filepath.Join(dir, ".shellforge")
	if err := os.MkdirAll(userDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	userCfg := Config{Provider: "anthropic", Model: "claude-3-opus"}
	data, _ := json.Marshal(userCfg)
	if err := os.WriteFile(filepath.Join(userDir, "config.json"), data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// CWD with no project/local config
	cwd := filepath.Join(dir, "project")
	if err := os.MkdirAll(cwd, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	origDir, _ := os.Getwd()
	os.Chdir(cwd)
	defer os.Chdir(origDir)

	// Clear env
	t.Setenv("SHELLFORGE_PROVIDER", "")
	t.Setenv("SHELLFORGE_MODEL", "")
	t.Setenv("SHELLFORGE_BASE_URL", "")
	t.Setenv("SHELLFORGE_MAX_TURNS", "")
	t.Setenv("SHELLFORGE_TIMEOUT_MS", "")
	t.Setenv("SHELLFORGE_TOKEN_BUDGET", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("DEEPSEEK_API_KEY", "")

	cfg := Load()
	if cfg.Provider != "anthropic" {
		t.Errorf("Provider = %q, want %q", cfg.Provider, "anthropic")
	}
	if cfg.Model != "claude-3-opus" {
		t.Errorf("Model = %q, want %q", cfg.Model, "claude-3-opus")
	}
	// Defaults should still be present for unset fields
	if cfg.MaxTurns != 15 {
		t.Errorf("MaxTurns = %d, want %d", cfg.MaxTurns, 15)
	}
}

func TestLoadProjectConfigOverridesUserConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// Create user config
	userDir := filepath.Join(dir, ".shellforge")
	if err := os.MkdirAll(userDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	userCfg := Config{Provider: "anthropic", Model: "claude-3-opus", MaxTurns: 20}
	data, _ := json.Marshal(userCfg)
	os.WriteFile(filepath.Join(userDir, "config.json"), data, 0644)

	// Create project config that overrides provider and model
	cwd := filepath.Join(dir, "project")
	projDir := filepath.Join(cwd, ".shellforge")
	if err := os.MkdirAll(projDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	projCfg := Config{Provider: "openai", Model: "gpt-4o"}
	data, _ = json.Marshal(projCfg)
	os.WriteFile(filepath.Join(projDir, "config.json"), data, 0644)

	origDir, _ := os.Getwd()
	os.Chdir(cwd)
	defer os.Chdir(origDir)

	// Clear env
	t.Setenv("SHELLFORGE_PROVIDER", "")
	t.Setenv("SHELLFORGE_MODEL", "")
	t.Setenv("SHELLFORGE_BASE_URL", "")
	t.Setenv("SHELLFORGE_MAX_TURNS", "")
	t.Setenv("SHELLFORGE_TIMEOUT_MS", "")
	t.Setenv("SHELLFORGE_TOKEN_BUDGET", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("DEEPSEEK_API_KEY", "")

	cfg := Load()
	if cfg.Provider != "openai" {
		t.Errorf("Provider = %q, want %q (project should override user)", cfg.Provider, "openai")
	}
	if cfg.Model != "gpt-4o" {
		t.Errorf("Model = %q, want %q (project should override user)", cfg.Model, "gpt-4o")
	}
	// MaxTurns from user config should still be present (project didn't set it)
	if cfg.MaxTurns != 20 {
		t.Errorf("MaxTurns = %d, want %d (from user config)", cfg.MaxTurns, 20)
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	t.Setenv("SHELLFORGE_PROVIDER", "deepseek")
	t.Setenv("SHELLFORGE_MODEL", "deepseek-coder")
	t.Setenv("SHELLFORGE_BASE_URL", "http://localhost:11434")
	t.Setenv("SHELLFORGE_MAX_TURNS", "50")
	t.Setenv("SHELLFORGE_TIMEOUT_MS", "300000")
	t.Setenv("SHELLFORGE_TOKEN_BUDGET", "8000")
	t.Setenv("DEEPSEEK_API_KEY", "sk-deep-123")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")

	cfg := Defaults()
	applyEnv(&cfg)

	if cfg.Provider != "deepseek" {
		t.Errorf("Provider = %q, want %q", cfg.Provider, "deepseek")
	}
	if cfg.Model != "deepseek-coder" {
		t.Errorf("Model = %q, want %q", cfg.Model, "deepseek-coder")
	}
	if cfg.BaseURL != "http://localhost:11434" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "http://localhost:11434")
	}
	if cfg.MaxTurns != 50 {
		t.Errorf("MaxTurns = %d, want %d", cfg.MaxTurns, 50)
	}
	if cfg.TimeoutMs != 300000 {
		t.Errorf("TimeoutMs = %d, want %d", cfg.TimeoutMs, 300000)
	}
	if cfg.TokenBudget != 8000 {
		t.Errorf("TokenBudget = %d, want %d", cfg.TokenBudget, 8000)
	}
	if cfg.APIKey != "sk-deep-123" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "sk-deep-123")
	}
}

func TestAPIKeyFromEnv(t *testing.T) {
	t.Setenv("SHELLFORGE_PROVIDER", "")
	t.Setenv("SHELLFORGE_MODEL", "")
	t.Setenv("SHELLFORGE_BASE_URL", "")
	t.Setenv("SHELLFORGE_MAX_TURNS", "")
	t.Setenv("SHELLFORGE_TIMEOUT_MS", "")
	t.Setenv("SHELLFORGE_TOKEN_BUDGET", "")
	t.Setenv("DEEPSEEK_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")

	// Set provider to anthropic and provide key
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-key")

	cfg := Config{Provider: "anthropic"}
	applyEnv(&cfg)

	if cfg.APIKey != "sk-ant-test-key" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "sk-ant-test-key")
	}
}

func TestAPIKeyFallbackOrder(t *testing.T) {
	// When provider doesn't match any specific key, fallback checks all three
	t.Setenv("SHELLFORGE_PROVIDER", "")
	t.Setenv("SHELLFORGE_MODEL", "")
	t.Setenv("SHELLFORGE_BASE_URL", "")
	t.Setenv("SHELLFORGE_MAX_TURNS", "")
	t.Setenv("SHELLFORGE_TIMEOUT_MS", "")
	t.Setenv("SHELLFORGE_TOKEN_BUDGET", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "sk-openai-fallback")
	t.Setenv("DEEPSEEK_API_KEY", "")

	cfg := Config{Provider: "ollama"} // ollama doesn't match any specific key
	applyEnv(&cfg)

	if cfg.APIKey != "sk-openai-fallback" {
		t.Errorf("APIKey = %q, want %q (fallback)", cfg.APIKey, "sk-openai-fallback")
	}
}
