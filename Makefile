.PHONY: setup dev up down logs pr-meta-check pr-open session-index session-find session-index-test

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
	@./infra/scripts/pr-metadata-check.sh --title "$(TITLE)" --body-file "$(BODY_FILE)" --base "$(or $(BASE),develop)" $(if $(HEAD),--head "$(HEAD)",)

pr-open:
	@test -n "$(TITLE)" || (echo "TITLE is required"; exit 2)
	@test -n "$(BODY_FILE)" || (echo "BODY_FILE is required"; exit 2)
	@./infra/scripts/pr-open.sh --title "$(TITLE)" --body-file "$(BODY_FILE)" --base "$(or $(BASE),develop)" $(if $(HEAD),--head "$(HEAD)",) $(if $(filter 1,$(DRAFT)),--draft,) $(if $(filter 1,$(AUTO_READY)),--auto-ready,)

# ── Local Codex session lookup ───────────────────────────────────────────────
session-index:
	@test -n "$(PR)" || (echo "PR is required"; exit 2)
	@./infra/scripts/session-index.sh add --pr "$(PR)" $(if $(ISSUE),--issue "$(ISSUE)",) $(if $(WORKTREE),--worktree "$(WORKTREE)",) $(if $(INDEX_FILE),--index-file "$(INDEX_FILE)",)

session-find:
	@test -n "$(PR)" || (echo "PR is required"; exit 2)
	@./infra/scripts/session-index.sh find --pr "$(PR)" $(if $(INDEX_FILE),--index-file "$(INDEX_FILE)",)

session-index-test:
	@./infra/scripts/session-index.test.sh
