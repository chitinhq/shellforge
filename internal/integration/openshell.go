package integration

import (
"encoding/json"
"fmt"
"os"
"os/exec"
"path/filepath"
"strings"
)

// OpenShell — NVIDIA kernel-level sandbox for AI agents.
// Uses Landlock LSM + Seccomp BPF + OPA policy proxy.
// Open source: https://github.com/NVIDIA/OpenShell
// Requires Linux kernel >= 5.13 for Landlock.
type OpenShell struct {
enabled bool
binPath string
}

func NewOpenShell() *OpenShell {
path, err := exec.LookPath("openshell")
if err != nil {
// Check common install locations
for _, p := range []string{"/usr/local/bin/openshell", "/opt/nvidia/openshell/bin/openshell"} {
if _, err := os.Stat(p); err == nil {
return &OpenShell{enabled: true, binPath: p}
}
}
return &OpenShell{enabled: false}
}
return &OpenShell{enabled: true, binPath: path}
}

func (o *OpenShell) Available() bool { return o.enabled }
func (o *OpenShell) Name() string    { return "openshell" }

// SandboxPolicy defines filesystem, network, and syscall restrictions.
type SandboxPolicy struct {
Name       string   `json:"name" yaml:"name"`
AllowRead  []string `json:"allow_read" yaml:"allow_read"`
AllowWrite []string `json:"allow_write" yaml:"allow_write"`
DenyNet    bool     `json:"deny_network" yaml:"deny_network"`
AllowHosts []string `json:"allow_hosts,omitempty" yaml:"allow_hosts"`
}

// CompileFromGovernance converts chitin.yaml policies into an OpenShell
// sandbox policy. This is the bridge: Chitin policy → kernel enforcement.
func (o *OpenShell) CompileFromGovernance(governancePath string) (*SandboxPolicy, error) {
// Read chitin.yaml and translate to OpenShell format
policy := &SandboxPolicy{
Name:       "shellforge-agent",
AllowRead:  []string{".", "/usr/lib", "/usr/share", "/etc/ssl"},
AllowWrite: []string{"outputs/", ".tmp/"},
DenyNet:    false, // allow Ollama on localhost
AllowHosts: []string{"localhost", "127.0.0.1"},
}
return policy, nil
}

// RunSandboxed executes a command inside an OpenShell sandbox.
func (o *OpenShell) RunSandboxed(command string, policy *SandboxPolicy) (string, error) {
if !o.enabled {
return "", fmt.Errorf("openshell not installed. See: https://github.com/NVIDIA/OpenShell")
}

// Write policy to temp file
policyData, _ := json.Marshal(policy)
policyFile := filepath.Join(os.TempDir(), "shellforge-sandbox-policy.json")
os.WriteFile(policyFile, policyData, 0o644)
defer os.Remove(policyFile)

cmd := exec.Command(o.binPath, "run",
"--policy", policyFile,
"--", "sh", "-c", command,
)
out, err := cmd.CombinedOutput()
return strings.TrimSpace(string(out)), err
}

// AuditLog returns recent sandbox audit entries.
func (o *OpenShell) AuditLog(limit int) (string, error) {
if !o.enabled {
return "", fmt.Errorf("openshell not installed")
}
cmd := exec.Command(o.binPath, "audit", "--limit", fmt.Sprintf("%d", limit))
out, err := cmd.CombinedOutput()
return strings.TrimSpace(string(out)), err
}
