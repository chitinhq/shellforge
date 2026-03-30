# ShellForge Squad — Blockers

**Updated:** 2026-03-30T00:45Z
**Reported by:** EM run 6 (claude-code:opus:shellforge:em)

---

## P0 — Critical Blockers

**None.** All P0 governance bugs are closed.

---

## P1 — Active Work

### PR #86 — Governance timeout override (awaiting human review)
**Description:** PR #86 removes the hardcoded 60s cap in `runShellWithRTK` and `runShellRaw` that silently overrode the governance engine's timeout value. CI pending; GitHub branch protection prevents self-approval.
**Action Required:** @jpleva91 review and approve PR #86.

### #63 — classifyShellRisk prefix matching too broad
**Severity:** High — false read-only classification on commands starting with `cat`/`ls`/`echo`
**Assignee:** qa-agent
**URL:** https://github.com/AgentGuardHQ/shellforge/issues/63

### #68 — Zero test coverage across all packages
**Severity:** High — governance runtime with no tests is unshipable
**Assignee:** qa-agent
**URL:** https://github.com/AgentGuardHQ/shellforge/issues/68

---

## P2 — Queued (unassigned)

| # | Issue | Notes |
|---|-------|-------|
| #76 | Dogfood: run ShellForge swarm on jared box | P0 governance bugs resolved — can now proceed |
| #65 | scheduler.go silent os.WriteFile error | Silent failure on job persistence |
| #66 | flattenParams dead code | Logic bug, result overwritten before use |
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
| PR #86 (P1 timeout fix) | CI pending — REVIEW REQUIRED |
| PR budget | 1/3 |
| Dogfood (#76) | Governance unblocked — needs human trigger |
| QA-agent (#63, #68) | Active |
| Retry loops | None |
| Blast radius | Low |
