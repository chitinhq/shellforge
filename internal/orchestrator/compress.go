package orchestrator

import (
	"github.com/chitinhq/shellforge/internal/canon"
)

// tokenThreshold is the maximum output size (in estimated tokens) that
// passes through without compression. Outputs below this are returned as-is.
const tokenThreshold = 750

// CompressResult compresses a sub-agent output if it exceeds the token threshold.
// Strategy:
//   1. If output < 750 tokens (estimated), return as-is
//   2. Otherwise truncate to the threshold with a marker
func CompressResult(output string) string {
	estimated := estimateTokens(output)
	if estimated <= tokenThreshold {
		return output
	}

	maxChars := tokenThreshold * 4
	if maxChars >= len(output) {
		return output
	}
	return output[:maxChars] + "\n\n[... output truncated — " + itoa(estimated-tokenThreshold) + " tokens omitted]"
}

// CompressShellResult compresses shell command output using canonical tool
// knowledge for structured compression instead of blind truncation.
func CompressShellResult(command, output string) string {
	estimated := estimateTokens(output)
	if estimated <= tokenThreshold {
		return output
	}

	cmd := canon.ParseOne(command)
	return canon.CompressOutput(cmd, output)
}

// estimateTokens provides a rough token count (1 token ~ 4 chars).
func estimateTokens(s string) int {
	return len(s) / 4
}

// itoa converts an int to a string without importing strconv (Go 1.18 compat).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
