.PHONY: setup dev up down logs pr-meta-check pr-open

# ── Setup (run once after cloning) ────────────────────────────────────────────
setup:
	@[ -f backend/.env ]   || cp backend/.env.example backend/.env   && echo "created backend/.env"
	@[ -f tachimint/.env ] || cp tachimint/.env.example tachimint/.env && echo "created tachimint/.env"
	@[ -f dashboard/.env ] || cp dashboard/.env.example dashboard/.env && echo "created dashboard/.env"
	@git config core.hooksPath .githooks

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
	@./scripts/pr-metadata-check.sh --title "$(TITLE)" --body-file "$(BODY_FILE)" --base "$(or $(BASE),develop)" $(if $(HEAD),--head "$(HEAD)",)

pr-open:
	@test -n "$(TITLE)" || (echo "TITLE is required"; exit 2)
	@test -n "$(BODY_FILE)" || (echo "BODY_FILE is required"; exit 2)
	@./scripts/pr-open.sh --title "$(TITLE)" --body-file "$(BODY_FILE)" --base "$(or $(BASE),develop)" $(if $(HEAD),--head "$(HEAD)",) $(if $(filter 1,$(DRAFT)),--draft,)
