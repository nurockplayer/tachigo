#!/usr/bin/env bash

set -euo pipefail

root_dir="$(cd "$(dirname "$0")/../.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

script="$root_dir/infra/scripts/check-typescript-only.sh"

init_repo() {
  local repo="$1"
  mkdir -p "$repo"
  git -C "$repo" init -q
}

track_file() {
  local repo="$1"
  local file_path="$2"
  local content="${3:-// fixture}"

  mkdir -p "$(dirname "$repo/$file_path")"
  printf '%s\n' "$content" > "$repo/$file_path"
  git -C "$repo" add "$file_path"
}

write_untracked_file() {
  local repo="$1"
  local file_path="$2"

  mkdir -p "$(dirname "$repo/$file_path")"
  printf '%s\n' '// untracked fixture' > "$repo/$file_path"
}

run_case() {
  local name="$1"
  local expected_exit="$2"
  local expected_message="${3:-}"
  shift 3

  local repo="$tmp_dir/$name"
  init_repo "$repo"
  "$@" "$repo"

  set +e
  bash "$script" --root "$repo" > "$tmp_dir/$name.out" 2> "$tmp_dir/$name.err"
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

clean_typescript_fixture() {
  local repo="$1"
  track_file "$repo" src/index.ts "export const ok = true"
  track_file "$repo" tools/check.ts "console.log('ok')"
}

source_javascript_fixture() {
  local repo="$1"
  track_file "$repo" apps/extension/src/legacy.js "export const legacy = true"
}

config_javascript_fixture() {
  local repo="$1"
  track_file "$repo" apps/dashboard/vite.config.js "export default {}"
}

tooling_javascript_fixture() {
  local repo="$1"
  track_file "$repo" infra/scripts/legacy-tool.mjs "console.log('legacy')"
}

test_javascript_fixture() {
  local repo="$1"
  track_file "$repo" packages/api-client/src/index.test.cjs "module.exports = {}"
}

untracked_javascript_fixture() {
  local repo="$1"
  track_file "$repo" src/index.ts "export const ok = true"
  write_untracked_file "$repo" src/not-tracked.js
}

allowed_fixture_data() {
  local repo="$1"
  track_file "$repo" infra/scripts/fixtures/input.js "module.exports = {}"
}

allowed_generated_output() {
  local repo="$1"
  track_file "$repo" apps/dashboard/dist/bundle.js "(()=>{})();"
}

allowed_archived_plan() {
  local repo="$1"
  track_file "$repo" plans/archived/old-script.js "// archived historical note"
}

run_case clean_typescript 0 "TypeScript-only guardrail passed" clean_typescript_fixture
run_case source_javascript 1 "apps/extension/src/legacy.js" source_javascript_fixture
run_case source_javascript_guidance 1 "Use .ts/.tsx instead" source_javascript_fixture
run_case source_javascript_node_guidance 1 "node --experimental-strip-types --no-warnings" source_javascript_fixture
run_case config_javascript 1 "apps/dashboard/vite.config.js" config_javascript_fixture
run_case tooling_javascript 1 "infra/scripts/legacy-tool.mjs" tooling_javascript_fixture
run_case test_javascript 1 "packages/api-client/src/index.test.cjs" test_javascript_fixture
run_case untracked_javascript 0 "TypeScript-only guardrail passed" untracked_javascript_fixture
run_case allowed_fixture_data 0 "TypeScript-only guardrail passed" allowed_fixture_data
run_case allowed_generated_output 0 "TypeScript-only guardrail passed" allowed_generated_output
run_case allowed_archived_plan 0 "TypeScript-only guardrail passed" allowed_archived_plan

echo "TypeScript-only guardrail regression tests passed"
