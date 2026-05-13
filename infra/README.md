# Infra

Repo-level automation that is not product code lives here.

## Contents

- `scripts/` — local and CI helper scripts for PR metadata checks, commit message checks, setup, and workflow assertions.
- `scripts/check-supply-chain-guardrails.mjs` — local and CI guardrail for dependency install lifecycle scripts, dynamic package execution, and Mini Shai-Hulud indicators.
- `scripts/check-developer-persistence.sh` — local-only check for common Claude / VS Code / LaunchAgent / systemd persistence indicators.
- `githooks/` — git hooks installed by `make setup` through `core.hooksPath`.

## Root-Level Exceptions

Some operational files must remain at the repository root or in GitHub's expected locations:

- `.github/` — GitHub Actions, PR templates, issue templates, and Dependabot configuration must stay in GitHub's conventional path.
- `docker-compose.yml` and `docker-compose.override.yml` — Docker Compose entrypoints used from the repo root.
- `Makefile` — convenience command entrypoint for local development.
- `.gitignore`, `.gitattributes`, `.editorconfig`, and `.claudeignore` — tool-discovered root configuration files.
