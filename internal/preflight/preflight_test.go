package preflight

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInjectGooseHints_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	if err := InjectGooseHints(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, ".goosehints"))
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if !strings.Contains(string(content), "# Preflight Protocol") {
		t.Error("expected Preflight header in created file")
	}
	if !strings.Contains(string(content), "Phase 1: Orient") {
		t.Error("expected Phase 1 content in created file")
	}
	if !strings.Contains(string(content), "Phase 5: Execute") {
		t.Error("expected Phase 5 content in created file")
	}
}

func TestInjectGooseHints_SkipsIfAlreadyPresent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".goosehints")

	existing := "# Preflight Protocol\nalready injected\n"
	if err := os.WriteFile(path, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	if err := InjectGooseHints(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(path)
	// Content should be unchanged.
	if string(content) != existing {
		t.Errorf("expected file unchanged, got: %q", string(content))
	}
}

func TestInjectGooseHints_PrependsToExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".goosehints")

	projectHints := "# Project-specific hints\nAlways use snake_case.\n"
	if err := os.WriteFile(path, []byte(projectHints), 0644); err != nil {
		t.Fatal(err)
	}

	if err := InjectGooseHints(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(path)
	s := string(content)

	// Preflight must come before project hints.
	preflightIdx := strings.Index(s, "# Preflight Protocol")
	projectIdx := strings.Index(s, "# Project-specific hints")
	if preflightIdx < 0 {
		t.Error("expected Preflight Protocol header")
	}
	if projectIdx < 0 {
		t.Error("expected project hints preserved")
	}
	if preflightIdx > projectIdx {
		t.Error("Preflight must precede project-specific hints")
	}

	// Idempotency: calling again must not double-inject.
	if err := InjectGooseHints(dir); err != nil {
		t.Fatalf("second call unexpected error: %v", err)
	}
	content2, _ := os.ReadFile(path)
	if strings.Count(string(content2), "# Preflight Protocol") != 1 {
		t.Error("expected exactly one Preflight Protocol header after double-inject")
	}
}

func TestInjectGooseHints_Idempotent(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 3; i++ {
		if err := InjectGooseHints(dir); err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
	}
	content, _ := os.ReadFile(filepath.Join(dir, ".goosehints"))
	if count := strings.Count(string(content), "# Preflight Protocol"); count != 1 {
		t.Errorf("expected exactly 1 Preflight header after 3 calls, got %d", count)
	}
}
