# ShellForge Squad — Blockers

**Updated:** 2026-03-30T08:42Z
**Reported by:** EM run 7 (claude-code:opus:shellforge:em)

---

## P0 — Critical Blockers

**None.** All P0 governance bugs are closed.

---

## P1 — Active Work

### PR #89 — Test coverage + dead code fix (awaiting human review)
**Description:** qa-agent opened PR #89 with 25 tests across `normalizer`, `governance`, and `intent` packages, plus the `flattenParams` dead code removal (#66). CI is green (5/5). GitHub branch protection prevents self-approval.
**Action Required:** @jpleva91 review and approve PR #89 — this closes the last P1 (#68 test coverage).
**URL:** https://github.com/AgentGuardHQ/shellforge/pull/89

---

## P2 — Active Blocker

### #76 — Dogfood: setup.sh doesn't support remote Ollama
**Severity:** Medium — dogfood on jared-box (headless WSL2 + RunPod GPU) is blocked
**Root cause:** `shellforge setup` detects `isServer=true` on headless Linux and skips Goose + Ollama entirely, with no option to configure `OLLAMA_HOST` for a remote GPU endpoint.
**Fix needed:** setup.sh should offer remote Ollama config when `isServer=true` — set `OLLAMA_HOST`, skip local Ollama install, keep Goose setup.
**URL:** https://github.com/AgentGuardHQ/shellforge/issues/76

---

## P2 — Queued (unassigned)

| # | Issue | Notes |
|---|-------|-------|
| #65 | scheduler.go silent os.WriteFile error | Silent failure on job persistence |
| #52 | filepath.Glob ** never matches Go files | cmdScan scan feature broken |
| #53 | README stale ./shellforge commands | Docs rot |
| #51 | run() helper silently ignores errors | Silent failure in main.go |
| #50 | kernel version comparison lexicographic | setup.sh version gate broken |
| #49 | InferenceQueue not priority-aware | Documented but unimplemented |
| #26 | run-qa/report agents don't build binary if missing | Setup gap |
| #25 | RunResult.Success heuristic incorrect | Agent loop reliability |
| #24 | listFiles() relative paths bug | Path resolution error |

---

## Resolved (this cycle)

- **#28** — bounded-execution policy timeout silently overridden to 60s → merged in PR #86
- **#63** — classifyShellRisk prefix matching too broad → merged in PR #88
- **#58** — bounded-execution wildcard policy blocked all run_shell → merged in PR #83
- **#62** — cmdEvaluate fail-open on JSON unmarshal → merged in PR #83
- **#75** — govern-shell.sh printf injection → merged in PR #83
- **#67** — govern-shell.sh fragile sed output parsing → merged in PR #83
- **#69** — rm policy only blocked -rf/-fr, not plain rm → merged in PR #83
- **#74** — stale crush references in cmdEvaluate → merged in PR #84
- **#59** — misleading `# Mode: monitor` comment → fixed in PR #83, closed manually

---

## Status Summary

| Item | Status |
|------|--------|
| P0 issues | ✅ All closed |
| P1 #28 (timeout fix) | ✅ Closed — PR #86 merged |
| P1 #63 (classifyShellRisk) | ✅ Closed — PR #88 merged |
| P1 #68 (test coverage) | 🟡 PR #89 open, CI green — REVIEW REQUIRED |
| Sprint goal | ✅ Achieved (pending PR #89 merge) |
| PR budget | 1/3 |
| Dogfood (#76) | 🔴 Blocked — setup.sh remote Ollama gap |
| Retry loops | None |
| Blast radius | Low |
