.PHONY: setup dev up down logs

# ── Setup (run once after cloning) ────────────────────────────────────────────
setup:
	@[ -f backend/.env ]   || cp backend/.env.example backend/.env   && echo "created backend/.env"
	@[ -f tachimint/.env ] || cp tachimint/.env.example tachimint/.env && echo "created tachimint/.env"
	@[ -f dashboard/.env ] || cp dashboard/.env.example dashboard/.env && echo "created dashboard/.env"

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
