// Package preflight bundles the Preflight design-before-you-build protocol
// (AgentGuardHQ/preflight v1) for injection into Goose agent bootstraps.
package preflight

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"
)

//go:embed goosehints.txt
var gooseHintsContent string

// InjectGooseHints writes the Preflight protocol into .goosehints in workDir.
//
// Behaviour:
//   - If .goosehints already contains the Preflight header, injection is skipped
//     (idempotent — safe to call on every Goose bootstrap).
//   - If .goosehints exists without Preflight, the protocol is prepended so it
//     takes precedence over project-level hints.
//   - If .goosehints does not exist, it is created with the protocol content.
//
// Returns an error only if the file cannot be read or written.
func InjectGooseHints(workDir string) error {
	path := filepath.Join(workDir, ".goosehints")

	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Already injected — skip.
	if strings.Contains(string(existing), "# Preflight Protocol") {
		return nil
	}

	var content string
	if len(existing) > 0 {
		content = gooseHintsContent + "\n---\n\n" + string(existing)
	} else {
		content = gooseHintsContent
	}

	return os.WriteFile(path, []byte(content), 0644)
}
