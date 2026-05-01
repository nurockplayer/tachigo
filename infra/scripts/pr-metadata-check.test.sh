#!/usr/bin/env bash

set -euo pipefail

root_dir="$(cd "$(dirname "$0")/../.." && pwd)"
head_branch="tmp-pr-metadata-check-test"

tmpdir="$(mktemp -d)"
trap 'git branch -D "$head_branch" >/dev/null 2>&1 || true; rm -rf "$tmpdir"' EXIT

resolve_base_ref() {
  if git show-ref --verify --quiet refs/remotes/origin/develop; then
    printf '%s\n' "origin/develop"
    return 0
  fi

  if git show-ref --verify --quiet refs/heads/develop; then
    echo "warning: refs/remotes/origin/develop not found; falling back to local develop" >&2
    printf '%s\n' "develop"
    return 0
  fi

  echo "warning: refs/remotes/origin/develop and local develop not found; falling back to HEAD" >&2
  printf '%s\n' "HEAD"
}

base_ref="$(resolve_base_ref)"
git branch -f "$head_branch" "$base_ref" >/dev/null 2>&1

run_case() {
  local name="$1"
  local title="$2"
  local depends_line="$3"
  local extra_body="${4:-}"
  local fake_gh_state="${5:-}"
  local expected_exit="${6:-0}"
  local body_file="$tmpdir/$name.md"
  local fakebin="$tmpdir/$name-bin"

  mkdir -p "$fakebin"
  cat > "$fakebin/gh" <<EOF
#!/usr/bin/env bash
set -euo pipefail
if [ "\${1:-}" = "auth" ] && [ "\${2:-}" = "status" ]; then
  exit 0
fi
if [ "\${1:-}" = "pr" ] && [ "\${2:-}" = "view" ]; then
  if [ -n "${fake_gh_state}" ]; then
    printf '%s\n' "${fake_gh_state}"
    exit 0
  fi
  exit 1
fi
exit 1
EOF
  chmod +x "$fakebin/gh"

  cat > "$body_file" <<EOF
refs #123

## Scope 對齊

- Source of truth：test
- $depends_line
- Backend contract already in develop:
  - [ ] yes
  - [x] no
- If no, this PR is:
  - [x] stacked on dependency branch
  - [ ] intentionally blocked until dependency merges
- 本 PR 明確不做：
  - no-op
$extra_body
EOF

  set +e
  PATH="$fakebin:$PATH" "$root_dir/infra/scripts/pr-metadata-check.sh" \
    --title "$title" \
    --body-file "$body_file" \
    --base develop \
    --head "$head_branch" \
    >/dev/null 2>"$tmpdir/$name.stderr"
  exit_code=$?
  set -e

  if [ "$exit_code" -ne "$expected_exit" ]; then
    echo "expected exit $expected_exit for case $name but got $exit_code" >&2
    cat "$tmpdir/$name.stderr" >&2
    exit 1
  fi
}

run_case fullwidth_no_space "[discussion] metadata parser regression" "Depends on PR：none"
run_case halfwidth_no_space "[discussion] metadata parser regression" "Depends on PR:none"
run_case closed_dependency "[frontend] metadata parser regression" "Depends on PR:#123" "" "CLOSED" 1
run_case open_dependency "[frontend] metadata parser regression" "Depends on PR:#123" "" "OPEN" 0
run_case invalid_dependency_value "[discussion] metadata parser regression" "Depends on PR：foobar" "" "" 1

# [chore] × product surface: must block (exit=1)
git branch -f "$head_branch" "$base_ref" >/dev/null 2>&1
_surface_wt="$tmpdir/surface-wt"
git worktree add -q "$_surface_wt" "$head_branch"
mkdir -p "$_surface_wt/services/api"
printf '// surface test\n' > "$_surface_wt/services/api/_pr_meta_surface_test.go"
git -C "$_surface_wt" add services/api/_pr_meta_surface_test.go
git -C "$_surface_wt" commit -q -m "test: chore touches backend"
git worktree remove -f "$_surface_wt"
run_case chore_product_surface "[chore] update ci config" "Depends on PR：none" "" "" 1
git branch -f "$head_branch" "$base_ref" >/dev/null 2>&1

# apps/dashboard/ and apps/extension/ are one frontend product surface.
_frontend_wt="$tmpdir/frontend-wt"
git worktree add -q "$_frontend_wt" "$head_branch"
mkdir -p "$_frontend_wt/apps/dashboard" "$_frontend_wt/apps/extension"
printf '// dashboard surface test\n' > "$_frontend_wt/apps/dashboard/_pr_meta_surface_test.ts"
printf '// extension surface test\n' > "$_frontend_wt/apps/extension/_pr_meta_surface_test.ts"
git -C "$_frontend_wt" add apps/dashboard/_pr_meta_surface_test.ts apps/extension/_pr_meta_surface_test.ts
git -C "$_frontend_wt" commit -q -m "test: frontend apps share surface"
git worktree remove -f "$_frontend_wt"
run_case frontend_apps_single_surface "[frontend] update both frontend apps" "Depends on PR：none"
git branch -f "$head_branch" "$base_ref" >/dev/null 2>&1

# ── checked_item robustness fixtures ─────────────────────────────────────────

run_case_body() {
  local name="$1" title="$2" body="$3" expected_exit="${4:-0}"
  local body_file="$tmpdir/$name.md"
  local fakebin="$tmpdir/$name-bin"
  mkdir -p "$fakebin"
  cat > "$fakebin/gh" <<'GHEOF'
#!/usr/bin/env bash
set -euo pipefail
if [ "${1:-}" = "auth" ] && [ "${2:-}" = "status" ]; then exit 0; fi
exit 1
GHEOF
  chmod +x "$fakebin/gh"
  printf '%s' "$body" > "$body_file"
  set +e
  PATH="$fakebin:$PATH" "$root_dir/infra/scripts/pr-metadata-check.sh" \
    --title "$title" --body-file "$body_file" --base develop --head "$head_branch" \
    >/dev/null 2>"$tmpdir/$name.stderr"
  local exit_code=$?
  set -e
  if [ "$exit_code" -ne "$expected_exit" ]; then
    echo "expected exit $expected_exit for case $name but got $exit_code" >&2
    cat "$tmpdir/$name.stderr" >&2
    exit 1
  fi
}

# Blank line after section header must not prevent finding the checkbox.
run_case_body "blank_after_section_header" "[backend] blank after header fixture" \
'refs #123

## Scope 對齊

- Source of truth：test
- Depends on PR：none
- Backend contract already in develop:

  - [x] yes
  - [ ] no
- 本 PR 明確不做：
  - no-op
'

# Blank lines between checkboxes must not cause early exit.
run_case_body "blank_between_checkboxes" "[frontend] blank between checkboxes fixture" \
'refs #123

## Scope 對齊

- Source of truth：test
- Depends on PR：none
- Backend contract already in develop:
  - [ ] yes

  - [x] no
- If no, this PR is:
  - [x] stacked on dependency branch
  - [ ] intentionally blocked until dependency merges
- 本 PR 明確不做：
  - no-op
'

# Section with extra text in name must not false-positive match.
# "Backend contract already in develop (notes):" should NOT count as the section.
run_case_body "similar_section_no_false_positive" "[backend] similar section name fixture" \
'refs #123

## Scope 對齊

- Source of truth：test
- Depends on PR：none
- Backend contract already in develop (notes):
  - [x] yes
  - [ ] no
- Backend contract already in develop:
  - [ ] yes
  - [x] no
- If no, this PR is:
  - [x] stacked on dependency branch
  - [ ] intentionally blocked until dependency merges
- 本 PR 明確不做：
  - no-op
'

# Both yes and no checked at the same time is a conflict — must be rejected.
run_case_body "yes_no_conflict" "[backend] yes no conflict fixture" \
'refs #123

## Scope 對齊

- Source of truth：test
- Depends on PR：none
- Backend contract already in develop:
  - [x] yes
  - [x] no
- 本 PR 明確不做：
  - no-op
' 1

echo "pr-metadata-check regression tests passed"
