package tools

import (
"os/exec"
"strings"
"time"
)

// rtkAvailable checks if RTK is installed for token-compressed shell output.
var rtkAvailable = func() bool {
_, err := exec.LookPath("rtk")
return err == nil
}()

// runShellWithRTK wraps a shell command through RTK for 60-90% token compression.
// Falls back to raw execution if RTK isn't available.
func runShellWithRTK(command string, timeoutSec int) Result {
if !rtkAvailable {
return runShellRaw(command, timeoutSec)
}

timeout := time.Duration(timeoutSec) * time.Second
if timeout > 60*time.Second {
timeout = 60 * time.Second
}

// RTK wraps the command: rtk sh -c "command"
cmd := exec.Command("rtk", "sh", "-c", command)

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

func runShellRaw(command string, timeoutSec int) Result {
timeout := time.Duration(timeoutSec) * time.Second
if timeout > 60*time.Second {
timeout = 60 * time.Second
}
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
