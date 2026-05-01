#!/usr/bin/env bash
set -euo pipefail

workflow=".github/workflows/ci.yml"
compose_file="docker-compose.yml"

require_in() {
  local file="$1"
  local pattern="$2"
  local message="$3"

  if ! grep -Fq -- "$pattern" "$file"; then
    echo "missing: $message" >&2
    exit 1
  fi
}

reject_in() {
  local file="$1"
  local pattern="$2"
  local message="$3"

  if grep -Fq -- "$pattern" "$file"; then
    echo "unexpected: $message" >&2
    exit 1
  fi
}

require_in "$compose_file" "image: tachigo-app:latest" "docker compose app service should use the CI-built image tag"
require_in "$workflow" "docker/setup-buildx-action@v3" "backend CI should initialize Docker Buildx"
require_in "$workflow" "docker/build-push-action@v6" "backend CI should build app image through build-push-action"
require_in "$workflow" "context: ./services/api" "backend CI should build from services/api"
require_in "$workflow" "target: dev" "backend CI should build the same dev target used by docker compose tests"
require_in "$workflow" "cache-from: type=gha" "backend CI should restore Docker layers from GitHub Actions cache"
require_in "$workflow" "cache-to: type=gha,mode=max" "backend CI should save Docker layers to GitHub Actions cache"
require_in "$workflow" "load: false" "backend build should validate Docker cache without loading the image"
require_in "$workflow" "tags: tachigo-app:latest" "backend CI should tag the image used by docker compose"
require_in "$workflow" "actions/setup-go@v6" "backend unit tests should use native Go"
require_in "$workflow" "go-version-file: services/api/go.mod" "backend unit tests should read services/api/go.mod"
require_in "$workflow" "working-directory: services/api" "backend unit tests should run from services/api"
require_in "$workflow" "run: go test ./..." "backend unit tests should run natively"
require_in "$workflow" "run: go vet ./..." "backend vet should run natively"
reject_in "$workflow" "actions/download-artifact" "backend unit tests should not download a Docker image artifact"
reject_in "$workflow" "docker load" "backend unit tests should not load a Docker image artifact"
reject_in "$workflow" "backend-image" "backend image artifact roundtrip should be removed"
reject_in "$workflow" "docker compose build app" "backend CI should not rebuild the app image without GHA layer cache"
reject_in "$workflow" "docker compose run --pull never --no-deps --rm app go test ./..." "unit tests should not use the prebuilt backend image"
