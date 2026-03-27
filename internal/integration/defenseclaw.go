package integration

import (
"encoding/json"
"fmt"
"os/exec"
"strings"
)

// DefenseClaw — Cisco's AI supply chain security framework.
// Scans agent skills, MCP servers, and generates AI Bill of Materials.
// Open source from RSA 2026. Pre-install scanning + runtime monitoring.
type DefenseClaw struct {
enabled bool
binPath string
}

func NewDefenseClaw() *DefenseClaw {
path, err := exec.LookPath("defenseclaw")
if err != nil {
return &DefenseClaw{enabled: false}
}
return &DefenseClaw{enabled: true, binPath: path}
}

func (d *DefenseClaw) Available() bool { return d.enabled }
func (d *DefenseClaw) Name() string    { return "defenseclaw" }

// ScanResult from a DefenseClaw skill/plugin scan.
type ScanResult struct {
Target       string   `json:"target"`
Status       string   `json:"status"` // clean, suspicious, malicious
Findings     []string `json:"findings"`
RiskScore    float64  `json:"risk_score"`
AIBomEntries int      `json:"ai_bom_entries"`
}

// ScanSkills scans all agent skills/plugins in the given directory.
func (d *DefenseClaw) ScanSkills(dir string) (*ScanResult, error) {
if !d.enabled {
return nil, fmt.Errorf("defenseclaw not installed. See: https://github.com/cisco/defenseclaw")
}
cmd := exec.Command(d.binPath, "scan", "--dir", dir, "--format", "json")
out, err := cmd.CombinedOutput()
if err != nil {
return nil, fmt.Errorf("scan failed: %w — %s", err, string(out))
}
var result ScanResult
if err := json.Unmarshal(out, &result); err != nil {
return nil, fmt.Errorf("parse scan result: %w", err)
}
return &result, nil
}

// ScanMCPServer verifies an MCP server connection before allowing agent access.
func (d *DefenseClaw) ScanMCPServer(serverURL string) (*ScanResult, error) {
if !d.enabled {
return nil, fmt.Errorf("defenseclaw not installed")
}
cmd := exec.Command(d.binPath, "scan-mcp", "--url", serverURL, "--format", "json")
out, err := cmd.CombinedOutput()
if err != nil {
return nil, fmt.Errorf("mcp scan failed: %w — %s", err, string(out))
}
var result ScanResult
if err := json.Unmarshal(out, &result); err != nil {
return nil, fmt.Errorf("parse scan result: %w", err)
}
return &result, nil
}

// GenerateBOM creates an AI Bill of Materials for the agent workspace.
func (d *DefenseClaw) GenerateBOM(dir string) (string, error) {
if !d.enabled {
return "", fmt.Errorf("defenseclaw not installed")
}
cmd := exec.Command(d.binPath, "bom", "--dir", dir, "--format", "json")
out, err := cmd.CombinedOutput()
return strings.TrimSpace(string(out)), err
}
