# AI Documentation

AI-facing collaboration guidance lives here unless a tool requires a specific path.

## Contents

- `claude-codex-cheatsheet.md` — quick reference for Claude Code and Codex collaboration.
- `claude-codex-workflow.md` — full workflow guide for low-token Claude Code usage.
- `code-review-refactor.md` — local Claude Code review workflow notes.
- `github-actions-debugging.md` — playbook for PR, CI, scope gate, and auto-ready debugging.
- `token-budget.md` — token budget guidance for AI-assisted work.

## Root-Level Exceptions

These files and directories stay outside `docs/ai/` because tools discover them by convention:

- `CLAUDE.md` — Claude Code entrypoint and repo-specific instructions.
- `AGENTS.md` — Codex/agent entrypoint and repo-specific instructions.
- `.claude/` — Claude Code commands, rules, and shared settings.
- `.codex/` — Codex local configuration.
- `.cursor/` — Cursor rules.

Root-level entrypoints should link to this directory when they need longer-form AI guidance.
