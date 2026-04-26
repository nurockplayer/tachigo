#!/usr/bin/env bash

set -euo pipefail

root_dir="$(cd "$(dirname "$0")/.." && pwd)"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

run_case() {
  local name="$1"
  local expected_exit="$2"
  local message_content="$3"
  local message_file="$tmpdir/$name.txt"

  printf '%s\n' "$message_content" > "$message_file"

  set +e
  "$root_dir/scripts/commit-message-check.sh" "$message_file" >"$tmpdir/$name.stdout" 2>"$tmpdir/$name.stderr"
  exit_code=$?
  set -e

  if [ "$exit_code" -ne "$expected_exit" ]; then
    echo "expected exit $expected_exit for case $name but got $exit_code" >&2
    cat "$tmpdir/$name.stderr" >&2
    exit 1
  fi
}

run_case valid_refs 0 "feat: enforce commit message policy

refs #369"

run_case valid_closes 0 "docs: document commit message policy

closes #369"

run_case merge_commit 0 "Merge branch 'develop' into feat/commit-policy"

run_case missing_issue_marker 1 "fix: tighten message validation"

run_case invalid_subject 1 "update validation script

refs #369"

range_repo="$tmpdir/range-repo"
mkdir -p "$range_repo"
cd "$range_repo"

git init -q
git config user.name "Codex Test"
git config user.email "codex@example.com"

printf 'seed\n' > README.md
git add README.md
git commit -q -m "feat: seed branch

refs #1"

git branch -M develop
git checkout -q -b feat/commit-message-policy

printf 'valid\n' >> README.md
git add README.md
git commit -q -m "fix: valid branch commit

refs #369"

git checkout -q develop
printf 'develop side change\n' > DEVELOP.md
git add DEVELOP.md
git commit -q -m "docs: update develop branch

refs #2"

git checkout -q feat/commit-message-policy
git merge --no-ff develop -m "Merge branch 'develop' into feat/commit-message-policy" >/dev/null

set +e
"$root_dir/scripts/check-pr-commit-messages.sh" develop HEAD >"$tmpdir/range-valid.stdout" 2>"$tmpdir/range-valid.stderr"
range_exit=$?
set -e

if [ "$range_exit" -ne 0 ]; then
  echo "expected range validation to pass" >&2
  cat "$tmpdir/range-valid.stderr" >&2
  exit 1
fi

printf 'invalid\n' >> README.md
git add README.md
git commit -q -m "chore: missing issue marker"

set +e
"$root_dir/scripts/check-pr-commit-messages.sh" develop HEAD >"$tmpdir/range-invalid.stdout" 2>"$tmpdir/range-invalid.stderr"
range_exit=$?
set -e

if [ "$range_exit" -eq 0 ]; then
  echo "expected range validation to fail for invalid commit" >&2
  exit 1
fi

echo "commit message regression tests passed"
