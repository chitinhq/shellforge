# Contributing

Thanks for showing up. This doc exists so your first PR lands without a scavenger hunt.

## Setup

```bash
git clone <this-repo>
cd <repo>
# Chitin-platform repos: `chitin init` bootstraps deps
# Go repos: `go build ./...`
# Python repos: `uv sync` or `pip install -e .`
```

If setup takes more than one command and a coffee, that's a bug — file an issue.

## PR naming

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <summary>
```

Types: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `perf`.

Real examples from this org:

- `feat(session): enrich triple with SoulContext`
- `fix(hook): tag governance events with active soul`
- `docs: add claws wrapper to GETTING_STARTED`

Scope is optional but helps reviewers route. Keep the summary under 70 chars.

## PR template

`.github/PULL_REQUEST_TEMPLATE.md` fills in when you open a PR. It's short on purpose. If a section doesn't apply, delete it. If you feel like adding a section, don't — open an issue instead.

## Review flow

1. CI + Copilot review run automatically on open.
2. Address Copilot comments before requesting human review (most are legit).
3. A human reviewer merges. Squash-merge is the default.

## Optional: soul context

If you wrote the PR under a specific cognitive lens (e.g. `hopper`, `feynman`, `turing`), mention it in the Summary. It's a hint for reviewers, not a requirement.

## Questions

Open an issue or ping in the workspace chat. Silent struggles help no one.
