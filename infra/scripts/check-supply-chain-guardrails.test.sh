#!/usr/bin/env bash

set -euo pipefail

root_dir="$(cd "$(dirname "$0")/../.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

script="$root_dir/infra/scripts/check-supply-chain-guardrails.mjs"

write_package() {
  local repo="$1"
  local package_path="$2"
  local scripts_json="${3:-\"build\":\"vite\"}"

  mkdir -p "$(dirname "$repo/$package_path")"
  cat > "$repo/$package_path" <<EOF
{
  "name": "fixture",
  "private": true,
  "packageManager": "pnpm@10.33.0",
  "scripts": {
    $scripts_json
  }
}
EOF
}

write_lockfile() {
  local repo="$1"
  local lockfile_path="${2:-pnpm-lock.yaml}"

  cat > "$repo/$lockfile_path" <<'EOF'
lockfileVersion: '9.0'

packages:
  '@tanstack/react-query@5.100.6':
    resolution: {integrity: sha512-clean}
EOF
}

run_case() {
  local name="$1"
  local expected_exit="$2"
  local expected_message="$3"
  shift 3

  local repo="$tmp_dir/$name"
  mkdir -p "$repo"
  "$@" "$repo"

  set +e
  node "$script" --root "$repo" > "$tmp_dir/$name.out" 2> "$tmp_dir/$name.err"
  local exit_code=$?
  set -e

  if [ "$exit_code" -ne "$expected_exit" ]; then
    echo "expected exit $expected_exit for $name but got $exit_code" >&2
    cat "$tmp_dir/$name.out" >&2
    cat "$tmp_dir/$name.err" >&2
    exit 1
  fi

  if [ -n "$expected_message" ]; then
    if ! grep -q "$expected_message" "$tmp_dir/$name.out" "$tmp_dir/$name.err"; then
      echo "expected message '$expected_message' for $name" >&2
      cat "$tmp_dir/$name.out" >&2
      cat "$tmp_dir/$name.err" >&2
      exit 1
    fi
  fi
}

clean_fixture() {
  local repo="$1"
  write_package "$repo" package.json
  write_package "$repo" apps/dashboard/package.json
  write_package "$repo" apps/extension/package.json
  write_lockfile "$repo"
}

preinstall_fixture() {
  local repo="$1"
  write_package "$repo" package.json
  write_package "$repo" apps/dashboard/package.json '"preinstall":"npx only-allow pnpm"'
  write_lockfile "$repo"
}

dynamic_exec_fixture() {
  local repo="$1"
  write_package "$repo" package.json '"audit":"pnpm dlx suspicious-tool"'
  write_lockfile "$repo"
}

ioc_fixture() {
  local repo="$1"
  write_package "$repo" package.json
  cat > "$repo/pnpm-lock.yaml" <<'EOF'
lockfileVersion: '9.0'

packages:
  github:tanstack/router#79ac49eedf774dd4b0cfa308722bc463cfe5885c:
    resolution: {tarball: https://example.invalid/router_init.js}
EOF
}

dockerfile_install_without_ignore_scripts_fixture() {
  local repo="$1"
  mkdir -p "$repo/apps/docs"
  cat > "$repo/apps/docs/Dockerfile" <<'EOF'
FROM node:24-alpine
RUN corepack enable
RUN pnpm install --frozen-lockfile
EOF
}

run_case clean 0 "Supply-chain guardrails passed" clean_fixture
run_case preinstall 1 "disallowed lifecycle script" preinstall_fixture
run_case dynamic_exec 1 "disallowed dynamic package execution" dynamic_exec_fixture
run_case ioc 1 "Mini Shai-Hulud indicator" ioc_fixture
run_case dockerfile_install_without_ignore_scripts 1 "Dockerfile pnpm install must use --frozen-lockfile --ignore-scripts" dockerfile_install_without_ignore_scripts_fixture

echo "supply-chain guardrail regression tests passed"
