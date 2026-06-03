.PHONY: setup dev up down logs pr-meta-check pr-open ai-readback ai-docs-build ai-test-backend ai-test-extension ai-test-dashboard ai-pr-check session-index session-find session-index-test supply-chain-check typescript-only-check developer-persistence-check kg-validate kg-export-json

# ── Setup (run once after cloning) ────────────────────────────────────────────
setup:
	@[ -f services/api/.env ]   || cp services/api/.env.example services/api/.env   && echo "created services/api/.env"
	@[ -f apps/extension/.env ] || cp apps/extension/.env.example apps/extension/.env && echo "created apps/extension/.env"
	@[ -f apps/dashboard/.env ] || cp apps/dashboard/.env.example apps/dashboard/.env && echo "created apps/dashboard/.env"
	@git config core.hooksPath infra/githooks

# ── Dev (setup + build + up) ───────────────────────────────────────────────────
dev: setup
	docker compose up --build

# ── Docker Compose ────────────────────────────────────────────────────────────
up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f

# ── PR preflight ──────────────────────────────────────────────────────────────
pr-meta-check:
	@test -n "$(TITLE)" || (echo "TITLE is required"; exit 2)
	@test -n "$(BODY_FILE)" || (echo "BODY_FILE is required"; exit 2)
	@./infra/scripts/pr-metadata-check.sh --title "$(TITLE)" --body-file "$(BODY_FILE)" --base "$(or $(BASE),develop)" $(if $(HEAD),--head "$(HEAD)",) $(if $(filter 1,$(AUTO_READY)),--auto-ready,)

pr-open:
	@test -n "$(TITLE)" || (echo "TITLE is required"; exit 2)
	@test -n "$(BODY_FILE)" || (echo "BODY_FILE is required"; exit 2)
	@./infra/scripts/pr-open.sh --title "$(TITLE)" --body-file "$(BODY_FILE)" --base "$(or $(BASE),develop)" $(if $(HEAD),--head "$(HEAD)",) $(if $(filter 1,$(DRAFT)),--draft,) $(if $(filter 1,$(READY)),--ready,) $(if $(filter 1,$(AUTO_READY)),--auto-ready,)

# ── AI-safe command surface ─────────────────────────────────────────────────
ai-readback:
	@echo "branch: $$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unavailable)"
	@develop_tip="$$(git rev-parse --verify refs/remotes/origin/develop^{commit} 2>/dev/null || true)"; \
	if [ -n "$$develop_tip" ]; then echo "origin/develop: $$develop_tip"; else echo "origin/develop: unavailable"; fi
	@if status="$$(git status --short 2>&1)"; then \
		if [ -z "$$status" ]; then echo "working tree: clean"; else echo "working tree: dirty"; printf '%s\n' "$$status"; fi; \
	else \
		echo "working tree: unavailable"; printf '%s\n' "$$status"; \
	fi
	@echo "pointers: CLAUDE.md docs/ai/ .claude/rules/"

ai-docs-build:
	@if [ ! -f apps/docs/package.json ]; then \
		echo "skipped/unavailable: apps/docs/package.json not found"; \
	elif ! command -v pnpm >/dev/null 2>&1; then \
		echo "skipped/unavailable: pnpm is not installed"; \
	else \
		pnpm build:docs; \
	fi

ai-test-backend:
	@if ! command -v docker >/dev/null 2>&1; then \
		echo "skipped/unavailable: docker is not installed"; \
	elif ! docker compose version >/dev/null 2>&1; then \
		echo "skipped/unavailable: docker compose is not available"; \
	elif ! docker info >/dev/null 2>&1; then \
		echo "skipped/unavailable: docker daemon is not available"; \
	else \
		docker compose run --no-deps --rm app go test ./...; \
	fi

ai-test-extension:
	@if [ ! -f apps/extension/package.json ]; then \
		echo "skipped/unavailable: apps/extension/package.json not found"; \
	elif ! command -v pnpm >/dev/null 2>&1; then \
		echo "skipped/unavailable: pnpm is not installed"; \
	else \
		pnpm --filter ./apps/extension test; \
	fi

ai-test-dashboard:
	@if [ ! -f apps/dashboard/package.json ]; then \
		echo "skipped/unavailable: apps/dashboard/package.json not found"; \
	elif ! command -v pnpm >/dev/null 2>&1; then \
		echo "skipped/unavailable: pnpm is not installed"; \
	else \
		pnpm --filter ./apps/dashboard test; \
	fi

ai-pr-check:
	@if [ -z "$(TITLE)" ] || [ -z "$(BODY_FILE)" ]; then \
		echo "Usage: make ai-pr-check TITLE=\"[chore] ...\" BODY_FILE=/tmp/pr_body.md [BASE=develop] [HEAD=branch] [AUTO_READY=1]"; \
		echo "Runs existing pr-meta-check; no PR or GitHub write operation is performed."; \
	else \
		$(MAKE) pr-meta-check TITLE="$(TITLE)" BODY_FILE="$(BODY_FILE)" BASE="$(or $(BASE),develop)" $(if $(HEAD),HEAD="$(HEAD)",) $(if $(AUTO_READY),AUTO_READY="$(AUTO_READY)",); \
	fi

# ── Local Codex session lookup ───────────────────────────────────────────────
session-index:
	@test -n "$(PR)" || (echo "PR is required"; exit 2)
	@./infra/scripts/session-index.sh add --pr "$(PR)" $(if $(ISSUE),--issue "$(ISSUE)",) $(if $(WORKTREE),--worktree "$(WORKTREE)",) $(if $(INDEX_FILE),--index-file "$(INDEX_FILE)",)

session-find:
	@test -n "$(PR)" || (echo "PR is required"; exit 2)
	@./infra/scripts/session-index.sh find --pr "$(PR)" $(if $(INDEX_FILE),--index-file "$(INDEX_FILE)",)

session-index-test:
	@./infra/scripts/session-index.test.sh

# ── Supply-chain guardrails ──────────────────────────────────────────────────
supply-chain-check:
	@node --experimental-strip-types --no-warnings infra/scripts/check-supply-chain-guardrails.ts

typescript-only-check:
	@bash infra/scripts/check-typescript-only.sh

developer-persistence-check:
	@bash infra/scripts/check-developer-persistence.sh

# ── Repository knowledge graph ───────────────────────────────────────────────
kg-validate:
	@node --experimental-strip-types --no-warnings infra/scripts/kg/validate-kg.ts

kg-export-json:
	@node --experimental-strip-types --no-warnings infra/scripts/kg/export-json.ts
