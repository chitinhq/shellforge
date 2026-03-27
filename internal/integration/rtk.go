// Package integration provides real integrations with the 8-project ecosystem.
package integration

import (
"fmt"
"os/exec"
"strings"
)

// RTK — Rust Token Killer. Wraps shell commands to compress output 60-90%
// before feeding it back to the LLM. Installed: rtk-ai/tap/rtk (brew).
// Already on this system at /home/jared/.local/bin/rtk v0.31.0.
type RTK struct {
enabled bool
}

func NewRTK() *RTK {
_, err := exec.LookPath("rtk")
return &RTK{enabled: err == nil}
}

func (r *RTK) Available() bool { return r.enabled }
func (r *RTK) Name() string    { return "rtk" }
func (r *RTK) Version() string {
out, err := exec.Command("rtk", "--version").Output()
if err != nil {
return "unknown"
}
return strings.TrimSpace(string(out))
}

// Wrap executes a command through RTK, compressing its output.
// Returns the compressed output that should be fed to the LLM.
func (r *RTK) Wrap(command string) (string, error) {
if !r.enabled {
return "", fmt.Errorf("rtk not installed")
}
// rtk wraps the command: rtk <command>
cmd := exec.Command("rtk", "sh", "-c", command)
out, err := cmd.CombinedOutput()
return string(out), err
}

// Stats returns RTK compression statistics for the current session.
func (r *RTK) Stats() (string, error) {
if !r.enabled {
return "", fmt.Errorf("rtk not installed")
}
out, err := exec.Command("rtk", "stats").CombinedOutput()
return strings.TrimSpace(string(out)), err
}
