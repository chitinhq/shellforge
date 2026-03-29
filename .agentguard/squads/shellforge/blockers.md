# ShellForge Squad — Blockers

**Updated:** 2026-03-29T10:00Z
**Reported by:** EM run (claude-code:opus:shellforge:em)

---

## P0 — Active Blockers (3)

### #58 — bounded-execution policy denies ALL run_shell calls in enforce mode
**Severity:** Critical — enforcement mode is non-functional
**Impact:** Any agent running under `bounded-execution` policy cannot execute shell commands at all. Blocks dogfood run (#76) and makes core governance a no-op in production.
**Assignee:** qa-agent (analysis) — needs dev-agent for fix
**URL:** https://github.com/AgentGuardHQ/shellforge/issues/58

---

### #62 — cmdEvaluate silently ignores JSON unmarshal error — governance bypass
**Severity:** Critical — security hole (fail-open pattern)
**Impact:** Malformed JSON payload causes silent error swallow — governance bypassed entirely. Go zero-value semantics: unpopulated struct → deny=false → allow. Exploitable by adversarial agent.
**Assignee:** security-scanner (analysis) — needs dev-agent for fix
**URL:** https://github.com/AgentGuardHQ/shellforge/issues/62

---

### #75 — govern-shell.sh: unescaped $COMMAND in printf silently defaults to allow
**Severity:** Critical — security hole in shell governance hook
**Impact:** Command strings with printf format specifiers (`%s`, `%n`) corrupt JSON payload; hook silently defaults to `allow`. Exploitable via shell-level injection.
**Assignee:** security-scanner (analysis) — needs dev-agent for fix
**URL:** https://github.com/AgentGuardHQ/shellforge/issues/75
**Fix:** Use `printf '%s'` quoting or switch to `jq -n --arg` for JSON construction.

---

## Capability Gap — No Dev Agent in Swarm

**Added:** 2026-03-29T10:00Z
**Severity:** High — limits squad's ability to ship fixes autonomously

Current agents (qa-agent, security-scanner, report-agent) produce analysis only — no agent can write code or open PRs. PR budget is 0/3 (fully available), meaning capacity exists for 3 parallel fix PRs but no agent to author them.

**Action needed:** Add `dev-agent` to agents.yaml, or dispatch feature-dev agent manually for P0 fixes.

---

## Notes

- PR budget: 0/3 open — capacity available to fix all three P0s in parallel once dev-agent exists
- No retry loops or blast radius concerns this run
- Dogfood run (#76, P2) is hard-blocked until at minimum #58 is resolved
- #77 triaged as P3 research this run — not urgent vs P0 security correctness
