package tools

import (
"os/exec"
"strings"
"time"

"github.com/chitinhq/shellforge/internal/canon"
)

// rtkAvailable checks if RTK is installed for token-compressed shell output.
var rtkAvailable = func() bool {
_, err := exec.LookPath("rtk")
return err == nil
}()

// runShellWithRTK wraps a shell command through RTK for 60-90% token compression.
// Uses canon to route to specific RTK subcommands for maximum compression.
// Falls back to raw execution + canon-aware compression if RTK isn't available.
func runShellWithRTK(command string, timeoutSec int) Result {
if !rtkAvailable {
return runShellWithCanon(command, timeoutSec)
}

timeout := time.Duration(timeoutSec) * time.Second

// Try canon-aware routing: rtk <tool> <args> instead of rtk sh -c.
// Specific RTK subcommands apply tool-aware filters for better compression.
var cmd *exec.Cmd
if args, specific := canon.RTKCommand(command); specific {
cmd = exec.Command("rtk", args...)
} else {
cmd = exec.Command("rtk", "sh", "-c", command)
}

done := make(chan error, 1)
var out []byte
go func() {
var err error
out, err = cmd.CombinedOutput()
done <- err
}()

select {
case err := <-done:
output := strings.TrimSpace(string(out))
if len(output) > MaxOutput {
output = output[:MaxOutput] + "\n... (truncated)"
}
if output == "" {
output = "(no output)"
}
if err != nil {
return Result{Success: false, Output: output, Error: err.Error()}
}
return Result{Success: true, Output: output}
case <-time.After(timeout):
if cmd.Process != nil {
cmd.Process.Kill()
}
return Result{Success: false, Output: "Command timed out", Error: "timeout"}
}
}

// runShellWithCanon executes a command raw, then applies canon-aware compression.
// Used when RTK is not installed.
func runShellWithCanon(command string, timeoutSec int) Result {
result := runShellRaw(command, timeoutSec)
if result.Success && len(result.Output) > MaxOutput/2 {
cmd := canon.ParseOne(command)
result.Output = canon.CompressOutput(cmd, result.Output)
}
return result
}

func runShellRaw(command string, timeoutSec int) Result {
timeout := time.Duration(timeoutSec) * time.Second
cmd := exec.Command("sh", "-c", command)

done := make(chan error, 1)
var out []byte
go func() {
var err error
out, err = cmd.CombinedOutput()
done <- err
}()

select {
case err := <-done:
output := strings.TrimSpace(string(out))
if len(output) > MaxOutput {
output = output[:MaxOutput] + "\n... (truncated)"
}
if output == "" {
output = "(no output)"
}
if err != nil {
return Result{Success: false, Output: output, Error: err.Error()}
}
return Result{Success: true, Output: output}
case <-time.After(timeout):
if cmd.Process != nil {
cmd.Process.Kill()
}
return Result{Success: false, Output: "Command timed out", Error: "timeout"}
}
}
