#!/usr/bin/env bash
# ForgeCode Issue Worker — pulls GitHub issues, works them via shellforge agent
# Usage: ./scripts/issue-worker.sh [anthropic|deepseek] [--dry-run]
# Expects API keys in environment (use: set -a && source .env && set +a)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SHELLFORGE="${SCRIPT_DIR}/../shellforge"
PROVIDER="${1:-anthropic}"
DRY_RUN="${2:-}"
WORK_DIR="/tmp/forgecode-work"
COOLDOWN=300  # 5 min between tasks
ATTEMPT_FILE="/tmp/forgecode-attempts.json"  # Track per-issue retry counts
MAX_ATTEMPTS=2  # Max retries before marking stuck

# Active repos to pull issues from
REPOS=(
  chitinhq/chitin
  chitinhq/clawta
  chitinhq/octi
  chitinhq/shellforge
  chitinhq/sentinel
  chitinhq/llmint

)

# Notify helper
notify() {
  local msg="$1"
  echo "[forgecode] $(date '+%H:%M:%S') $msg"
  if [[ -n "${NTFY_TOPIC:-}" ]]; then
    curl -s -H "Title: ForgeCode" -d "$msg" "https://ntfy.sh/${NTFY_TOPIC}" > /dev/null 2>&1 || true
  fi
}

# Attempt tracking — prevents infinite retries on the same issue
get_attempts() {
  local key="$1"
  if [[ -f "$ATTEMPT_FILE" ]]; then
    jq -r --arg k "$key" '.[$k] // 0' "$ATTEMPT_FILE" 2>/dev/null || echo 0
  else
    echo 0
  fi
}

inc_attempts() {
  local key="$1"
  local current
  current=$(get_attempts "$key")
  local new=$((current + 1))
  if [[ -f "$ATTEMPT_FILE" ]]; then
    jq --arg k "$key" --argjson v "$new" '.[$k] = $v' "$ATTEMPT_FILE" > "${ATTEMPT_FILE}.tmp" && mv "${ATTEMPT_FILE}.tmp" "$ATTEMPT_FILE"
  else
    echo "{\"$key\": $new}" > "$ATTEMPT_FILE"
  fi
  echo "$new"
}

clear_attempts() {
  local key="$1"
  if [[ -f "$ATTEMPT_FILE" ]]; then
    jq --arg k "$key" 'del(.[$k])' "$ATTEMPT_FILE" > "${ATTEMPT_FILE}.tmp" && mv "${ATTEMPT_FILE}.tmp" "$ATTEMPT_FILE"
  fi
}

pick_issue() {
  # Find first unclaimed issue across repos
  for repo in "${REPOS[@]}"; do
    local issue
    issue=$(gh issue list -R "$repo" --state open --json number,title,labels \
      --jq '[.[] | select((.labels | map(.name) | index("human-required") | not) and (.labels | map(.name) | index("agent:claimed") | not) and (.labels | map(.name) | index("agent:stuck") | not))] | first // empty | "\(.number)\t\(.title)"' 2>/dev/null || true)

    if [[ -n "$issue" && "$issue" != "null" ]]; then
      local num title
      num=$(echo "$issue" | cut -f1)
      title=$(echo "$issue" | cut -f2-)
      local key="${repo}#${num}"
      local attempts
      attempts=$(get_attempts "$key")
      if (( attempts >= MAX_ATTEMPTS )); then
        notify "Skipping ${key} — ${attempts} failed attempts, marking stuck" >&2
        gh issue edit "$num" -R "$repo" --add-label "agent:stuck" 2>/dev/null || true
        continue
      fi
      echo "${repo}|${num}|${title}"
      return 0
    fi
  done
  return 1
}

work_issue() {
  local repo="$1" num="$2" title="$3"
  local org repo_name clone_dir branch

  org=$(echo "$repo" | cut -d/ -f1)
  repo_name=$(echo "$repo" | cut -d/ -f2)
  clone_dir="${WORK_DIR}/${repo_name}"
  branch="forgecode/issue-${num}"

  notify "Working ${repo}#${num}: ${title}"

  # Claim the issue
  gh issue edit "$num" -R "$repo" --add-label "agent:claimed" 2>/dev/null || true

  # Clone or update repo
  if [[ -d "$clone_dir" ]]; then
    git -C "$clone_dir" checkout main 2>/dev/null || git -C "$clone_dir" checkout master 2>/dev/null
    git -C "$clone_dir" pull --ff-only 2>/dev/null || true
  else
    gh repo clone "$repo" "$clone_dir" 2>/dev/null
  fi

  # Create branch
  git -C "$clone_dir" checkout -b "$branch" 2>/dev/null || git -C "$clone_dir" checkout "$branch" 2>/dev/null

  # Bootstrap worker governance policy (permissive enough to write code)
  cp "${SCRIPT_DIR}/../policies/worker.yaml" "$clone_dir/chitin.yaml" 2>/dev/null || true

  # Read full issue body
  local body
  body=$(gh issue view "$num" -R "$repo" --json body --jq '.body' 2>/dev/null || echo "")

  # Build prompt — directive, not exploratory
  local prompt="TASK: Implement GitHub issue #${num} in ${repo}.

Title: ${title}

Description:
${body}

CRITICAL INSTRUCTIONS:
- You MUST write code using the write_file tool. Do NOT just read and analyze.
- Spend at most 3-4 turns reading code, then START WRITING.
- When calling write_file, you MUST include both 'path' and 'content' parameters.
- Write the implementation files directly. Do not summarize or plan.
- NEVER delete files you have written. If tests fail, fix the code — do not rm it.
- After writing, run 'go build ./...' or equivalent to verify it compiles.
- If you run out of turns without writing code, you have FAILED.

Working directory: ${clone_dir}"

  # Run shellforge agent
  if [[ "$DRY_RUN" == "--dry-run" ]]; then
    echo "[dry-run] Would run: $SHELLFORGE agent --provider $PROVIDER"
    echo "[dry-run] Prompt: $prompt"
    return 0
  fi

  local issue_key="${repo}#${num}"

  cd "$clone_dir"
  timeout 1800 "$SHELLFORGE" agent --provider "$PROVIDER" "$prompt" 2>&1 | tee "/tmp/forgecode-${repo_name}-${num}.log" || {
    local attempt_num
    attempt_num=$(inc_attempts "$issue_key")
    notify "FAILED on ${issue_key} (attempt ${attempt_num}/${MAX_ATTEMPTS}): check /tmp/forgecode-${repo_name}-${num}.log"
    gh issue edit "$num" -R "$repo" --remove-label "agent:claimed" 2>/dev/null || true
    return 1
  }

  # Check if anything changed
  if git -C "$clone_dir" diff --quiet && git -C "$clone_dir" diff --cached --quiet; then
    local attempt_num
    attempt_num=$(inc_attempts "$issue_key")
    notify "No changes for ${issue_key} (attempt ${attempt_num}/${MAX_ATTEMPTS}) — skipping PR"
    gh issue edit "$num" -R "$repo" --remove-label "agent:claimed" 2>/dev/null || true
    return 0
  fi

  # Success — clear attempt counter
  clear_attempts "$issue_key"

  # Avoid double-prefixing when the issue title already carries a
  # conventional-commit prefix (closes chitinhq/shellforge#134).
  case "$title" in
    fix:*|feat:*|chore:*|docs:*|test:*|refactor:*|perf:*|arch:*|build:*|ci:*|style:*|revert:*)
      pr_title="$title"
      commit_subject="address #${num} — ${title}" ;;
    *)
      pr_title="fix: $title"
      commit_subject="fix: address #${num} — ${title}" ;;
  esac

  # Commit and push
  git -C "$clone_dir" add -A
  git -C "$clone_dir" commit -m "${commit_subject}

Co-Authored-By: ForgeCode <forgecode@chitinhq.com>"
  git -C "$clone_dir" push -u origin "$branch" 2>/dev/null

  # Create PR
  gh pr create -R "$repo" \
    --title "$pr_title" \
    --body "Closes #${num}

## Summary
Automated fix by ForgeCode agent (${PROVIDER}).

## Test plan
- [ ] Review changes
- [ ] Verify build passes
- [ ] Verify tests pass" \
    --head "$branch" 2>/dev/null

  notify "PR created for ${repo}#${num}"
  gh issue edit "$num" -R "$repo" --add-label "agent:working" 2>/dev/null || true
}

# Main loop
notify "ForgeCode issue worker starting (provider: ${PROVIDER})"
mkdir -p "$WORK_DIR"

while true; do
  if result=$(pick_issue); then
    repo=$(echo "$result" | cut -d'|' -f1)
    num=$(echo "$result" | cut -d'|' -f2)
    title=$(echo "$result" | cut -d'|' -f3-)

    work_issue "$repo" "$num" "$title" || true

    notify "Cooling down ${COOLDOWN}s..."
    sleep "$COOLDOWN"
  else
    notify "No unclaimed issues. Sleeping 15m..."
    sleep 900
  fi
done
