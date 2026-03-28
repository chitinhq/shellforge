package governance

import (
"fmt"
"os"
"strings"

"gopkg.in/yaml.v3"
)

type Policy struct {
Name        string `yaml:"name"`
Description string `yaml:"description"`
Match       Match  `yaml:"match"`
Action      string `yaml:"action"`
Message     string `yaml:"message"`
Timeout     int    `yaml:"timeout_seconds,omitempty"`
}

type Match struct {
Command     string   `yaml:"command"`
ArgsContain []string `yaml:"args_contain"`
PathNotUnder []string `yaml:"path_not_under"`
}

type Config struct {
Mode      string    `yaml:"mode"`
Policies  []Policy  `yaml:"policies"`
Telemetry Telemetry `yaml:"telemetry"`
}

type Telemetry struct {
Enabled     bool `yaml:"enabled"`
LocalSQLite bool `yaml:"local_sqlite"`
Cloud       bool `yaml:"cloud"`
}

type Decision struct {
Allowed    bool
PolicyName string
Reason     string
Mode       string
}

type Engine struct {
Mode     string
Policies []Policy
}

func NewEngine(configPath string) (*Engine, error) {
data, err := os.ReadFile(configPath)
if err != nil {
return nil, fmt.Errorf("read governance config: %w", err)
}

var cfg Config
if err := yaml.Unmarshal(data, &cfg); err != nil {
return nil, fmt.Errorf("parse governance config: %w", err)
}

mode := cfg.Mode
if mode == "" {
mode = "monitor"
}

return &Engine{
Mode:     mode,
Policies: cfg.Policies,
}, nil
}

// Evaluate checks a tool call against all policies and returns an allow/deny Decision.
// In enforce mode, deny policies block execution. In monitor mode, they log only.
func (e *Engine) Evaluate(tool string, params map[string]string) Decision {
for _, p := range e.Policies {
if e.matches(p, tool, params) {
switch p.Action {
case "deny":
return Decision{
Allowed:    e.Mode != "enforce",
PolicyName: p.Name,
Reason:     p.Message,
Mode:       e.Mode,
}
case "monitor":
return Decision{
Allowed:    true,
PolicyName: p.Name,
Reason:     "[monitor] " + p.Message,
Mode:       e.Mode,
}
}
}
}
return Decision{
Allowed:    true,
PolicyName: "default-allow",
Reason:     "No matching deny policy",
Mode:       e.Mode,
}
}

// GetTimeout returns the first policy-level timeout in seconds, or 300 if none is set.
func (e *Engine) GetTimeout() int {
for _, p := range e.Policies {
if p.Timeout > 0 {
return p.Timeout
}
}
return 300
}

func (e *Engine) matches(p Policy, tool string, params map[string]string) bool {
m := p.Match

// Shell command policies
if tool == "run_shell" && m.Command != "" {
cmd := params["command"]
if m.Command == "*" {
if len(m.ArgsContain) > 0 {
return containsAny(cmd, m.ArgsContain)
}
return p.Timeout > 0
}
if !strings.Contains(cmd, m.Command) {
return false
}
if len(m.ArgsContain) > 0 {
return containsAny(cmd, m.ArgsContain)
}
return true
}

// File write policies
if tool == "write_file" && m.Command == "write_file" {
path := params["path"]
if len(m.PathNotUnder) > 0 {
norm := strings.TrimPrefix(path, "./")
for _, dir := range m.PathNotUnder {
if strings.HasPrefix(norm, dir) {
return false // under allowed dir — no match
}
}
return true // NOT under any allowed dir — matches deny
}
}

return false
}

func containsAny(s string, patterns []string) bool {
for _, p := range patterns {
if strings.Contains(s, p) {
return true
}
}
return false
}
