# ShellForge Squad — Blockers

**Updated:** 2026-03-31T00:00Z
**Reported by:** EM run 9 (claude-code:opus:shellforge:em)

---

## P0 — Critical Blockers

**None.** All P0 governance bugs are closed.

---

## P1 — Active Work

**None.** All P1 issues closed (PR #89 merged — closes #68 + #66).

---

## Incident (Resolved)

### Broken worktree — incomplete WIP fix for #51
**Detected:** Run 9 (2026-03-31)
**Resolved:** Yes
**Description:** The worktree had uncommitted partial changes to `cmd/shellforge/main.go`:
- `import (` was replaced with `import "log"`, breaking the multi-package import block syntax
- `run()` was partially refactored to call a non-existent `executeCommand()` function, leaving the old body orphaned outside any function
- Build failure: `syntax error: non-declaration statement outside function body`

**Resolution:** Stashed the WIP changes, created `fix/run-silent-errors-51` branch from `origin/main`, implemented the fix correctly (add `"log"` to imports, log error in `run()` via `if err := cmd.Run(); err != nil`). PR #93 open.

---

## P2 — Active Blockers

### PR Review Queue (budget: 2/3)
| PR | Title | Status |
|----|-------|--------|
| #91 | EM state update run 8 | CI green — REVIEW REQUIRED |
| #93 | fix run() silent errors (closes #51) | CI pending — REVIEW REQUIRED |

**Action Required:** @jpleva91 review and merge PR #91 and PR #93.

### #76 — Dogfood: setup.sh doesn't support remote Ollama (3rd escalation)
**Severity:** Medium — dogfood on jared-box (headless WSL2 + RunPod GPU) blocked
**Root cause:** `shellforge setup` detects `isServer=true` on headless Linux and skips Goose + Ollama entirely, with no option to configure `OLLAMA_HOST` for a remote GPU endpoint.
**Fix needed:** setup.sh should offer remote Ollama config when `isServer=true` — set `OLLAMA_HOST`, skip local Ollama install, keep Goose setup.
**URL:** https://github.com/AgentGuardHQ/shellforge/issues/76

---

## P2 — Queued (unassigned)

| # | Issue | Notes |
|---|-------|-------|
| #92 | Bundle Preflight in Goose bootstrap | Blocked on Preflight v1 ship |
| #65 | scheduler.go silent os.WriteFile error | Next EM fix after PR budget clears |
| #52 | filepath.Glob ** never matches Go files | Next EM fix — needs filepath.Walk |
| #53 | README stale ./shellforge commands | Docs rot |
| #50 | kernel version comparison lexicographic | setup.sh version gate broken |
| #49 | InferenceQueue not priority-aware | Documented but unimplemented |
| #26 | run-qa/report agents don't build binary if missing | Setup gap |
| #25 | RunResult.Success heuristic incorrect | Agent loop reliability |
| #24 | listFiles() relative paths bug | Path resolution error |

---

## Resolved (this cycle)

- **#68** — zero test coverage → merged PR #89 (25 tests for normalizer/governance/intent)
- **#66** — dead code in flattenParams() → fixed in PR #89
- **#51** — run() helper silently ignores errors → PR #93 open

## Resolved (prior cycles)

- **#28** → PR #86 merged
- **#63** → PR #88 merged
- **#58, #62, #75, #67, #69** → PR #83 merged
- **#74** → PR #84 merged

---

## Status Summary

| Item | Status |
|------|--------|
| P0 issues | ✅ All closed |
| P1 issues | ✅ All closed |
| PR #91 (EM state run 8) | 🟡 CI green — REVIEW REQUIRED |
| PR #93 (fix #51) | 🟡 CI pending — REVIEW REQUIRED |
| Sprint goal | 🔵 Active — P2 sweep in progress |
| PR budget | 2/3 |
| Dogfood (#76) | 🔴 Blocked — setup.sh remote Ollama gap (3rd escalation) |
| Retry loops | None |
| Blast radius | Low |
