# ShellForge Squad — Blockers

**Updated:** 2026-03-29T19:30Z
**Reported by:** EM run (claude-code:opus:shellforge:em)

---

## P0 — Active Blockers (1)

### PR #83 — Awaiting Human Review (BLOCKING MERGE)
**Description:** CI is passing (5/5 checks) but GitHub branch protection requires at least one approving review. The EM agent cannot self-approve (authored by jpleva91).
**Impact:** All 3 P0 governance security fixes (#58, #62, #75) and 2 P1 fixes (#67, #69) are stuck behind this review gate. Dogfood run (#76) is blocked until these merge.
**Action Required:** @jpleva91 or a collaborator must review and approve PR #83.
**PR:** https://github.com/AgentGuardHQ/shellforge/pull/83

---

## P1 — Remaining Work

### #68 — Zero test coverage across all packages
**Severity:** High — governance runtime with no tests is unshipable
**Impact:** Can't validate fix correctness, no regression protection. Blocks dogfood credibility.
**Assignee:** qa-agent
**URL:** https://github.com/AgentGuardHQ/shellforge/issues/68

### #63 — classifyShellRisk prefix matching too broad
**Severity:** High — false read-only classification on commands starting with `cat`/`ls`/`echo`
**Assignee:** qa-agent
**URL:** https://github.com/AgentGuardHQ/shellforge/issues/63

### #74 — Stale Crush comments in cmdEvaluate
**Severity:** Low-medium — internal comments still reference Crush fork; now fixed in PR #84
**Status:** Fix open in PR #84 (awaiting CI + review)
**URL:** https://github.com/AgentGuardHQ/shellforge/issues/74

---

## P2 — Unassigned (next dev-agent batch)

| # | Issue | Notes |
|---|-------|-------|
| #65 | scheduler.go silent os.WriteFile error | Silent failure, P2 |
| #66 | flattenParams dead code | Logic bug, P2 |
| #52 | filepath.Glob ** never matches Go files | cmdScan broken, P2 |
| #53 | README stale ./shellforge commands | Docs, P2 |

---

## Resolved This Run

- **#74** — Stale Crush comments in main.go → fix opened in PR #84
- **#58** — bounded-execution wildcard policy matched every run_shell → fix in PR #83 (pending merge)
- **#62** — cmdEvaluate fail-open on JSON unmarshal → fix in PR #83 (pending merge)
- **#75** — govern-shell.sh printf injection → fix in PR #83 (pending merge)
- **#67** — govern-shell.sh fragile sed output parsing → fix in PR #83 (pending merge)
- **#69** — rm policy only blocked -rf/-fr, not plain rm → fix in PR #83 (pending merge)
- **#59** — misleading `# Mode: monitor` comment with `mode: enforce` → fix in PR #83 (pending merge)

---

## Notes

- PR budget: 2/3 open — capacity for 1 more fix PR
- No retry loops or blast radius concerns
- Dogfood run (#76) blocked until PR #83 merges
- Test coverage (#68) is the most pressing remaining gap — no regression safety net
- Capability gap: no dev-agent in swarm. EM continuing to author fixes directly.
