# ShellForge Squad — Blockers

**Updated:** 2026-03-31T08:30Z
**Reported by:** EM run 11 (claude-code:opus:shellforge:em)

---

## P0 — Critical Blockers

**None.** All P0 governance bugs are closed.

---

## P1 — Active Work

**None.** All P1 issues closed.

---

## P2 — Active Blockers

### PR Review Queue (budget: 3/3)

| PR | Title | CI | Status |
|----|-------|----|--------|
| #93 | fix run() silent errors (closes #51) | ✅ 5/5 | REVIEW REQUIRED |
| #95 | fix scheduler WriteFile silent error (closes #65) | ✅ 5/5 | REVIEW REQUIRED |
| #96 | fix cmdScan Glob→WalkDir (closes #52) | ⏳ pending | REVIEW REQUIRED |

**Action Required:** @jpleva91 review and merge PRs #93, #95, #96 to clear budget for remaining P2 sweep.

### #76 — Dogfood: setup.sh doesn't support remote Ollama (4th escalation)

**Severity:** Medium — dogfood on jared-box (headless WSL2 + RunPod GPU) blocked
**Root cause:** `shellforge setup` detects `isServer=true` on headless Linux and skips Goose + Ollama entirely, with no option to configure `OLLAMA_HOST` for a remote GPU endpoint.
**Fix needed:** setup.sh should offer remote Ollama config when `isServer=true` — set `OLLAMA_HOST`, skip local Ollama install, keep Goose setup.
**URL:** https://github.com/chitinhq/shellforge/issues/76

---

## P2 — Queued (unassigned, after budget clears)

| # | Issue | Notes |
|---|-------|-------|
| #92 | Bundle Preflight in Goose bootstrap | Blocked on Preflight v1 ship |
| #53 | README stale ./shellforge commands | Docs rot |
| #50 | kernel version comparison lexicographic | setup.sh version gate broken |
| #49 | InferenceQueue not priority-aware | Documented but unimplemented |
| #26 | run-qa/report agents don't build binary if missing | Setup gap |
| #25 | RunResult.Success heuristic incorrect | Agent loop reliability |
| #24 | listFiles() relative paths bug | Path resolution error |

---

## Resolved (this cycle — run 11)

- **#52** — filepath.Glob ** never matches Go files → fixed with WalkDir in PR #96
- **PR #94** — stale EM state PR (run 9 state was already on master at 832cb58) → closed

## Resolved (prior cycles)

- **#65** — scheduler.go silent WriteFile error → PR #95 open
- **#51** — run() helper silently ignores errors → PR #93 open
- **#68** — zero test coverage → merged PR #89 (25 tests)
- **#66** — dead code in flattenParams() → merged PR #89
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
| PR #93 (fix #51 run() errors) | 🟡 CI green — REVIEW REQUIRED |
| PR #95 (fix #65 WriteFile) | 🟡 CI green — REVIEW REQUIRED |
| PR #96 (fix #52 Glob→WalkDir) | 🟡 CI pending — REVIEW REQUIRED |
| Sprint goal | 🔵 Active — P2 sweep 3/3 bugs fixed, all in PRs |
| PR budget | 3/3 (full — merge needed before new work) |
| Dogfood (#76) | 🔴 Blocked — setup.sh remote Ollama gap (4th escalation) |
| Retry loops | None |
| Blast radius | Low |
