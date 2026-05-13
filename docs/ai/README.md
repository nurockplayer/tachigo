# AI Documentation

AI-facing collaboration guidance lives here unless a tool requires a specific path.

## Contents

- `claude-codex-cheatsheet.md` — quick reference for Claude Code and Codex collaboration.
- `claude-codex-workflow.md` — full workflow guide for low-token Claude Code usage.
- `autonomous-bootstrap.md` — single AI-facing startup entrypoint for Hybrid AWP with Explicit Fallback Gate and local-only spec-injector flow.
- `codex-autonomous-workflow.md` — autonomous worker profiles, routing rules, review gates, and PR scope contract.
- `code-review-refactor.md` — local Claude Code review workflow notes.
- `github-ssh-443-push.md` — playbook for GitHub SSH over 443 when `git push` is unstable.
- `github-actions-debugging.md` — playbook for PR, CI, scope gate, and auto-ready debugging.
- `supply-chain-security.md` — dependency install, AI-agent package use, and developer-machine persistence guardrails.
- `token-budget.md` — token budget guidance for AI-assisted work.

## Root-Level Exceptions

These files and directories stay outside `docs/ai/` because tools discover them by convention:

- `CLAUDE.md` — Claude Code entrypoint and repo-specific instructions.
- `AGENTS.md` — Codex/agent entrypoint and repo-specific instructions.
- `.claude/` — Claude Code commands, rules, and shared settings.
- `.codex/` — Codex local configuration.
- `.cursor/` — Cursor rules.

Root-level entrypoints should link to this directory when they need longer-form AI guidance.
