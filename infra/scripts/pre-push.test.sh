#!/usr/bin/env bash

set -euo pipefail

root_dir="$(cd "$(dirname "$0")/../.." && pwd)"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

repo_dir="$tmpdir/repo"
mkdir -p "$repo_dir"
cd "$repo_dir"

git init -q
git config user.name "Codex Test"
git config user.email "codex@example.com"

mkdir -p apps/extension/src/assets
printf 'seed\n' > README.md
git add README.md
git commit -q -m "seed"

git branch -M develop
git checkout -q -b feature/lfs-scan
git config branch.feature/lfs-scan.merge refs/heads/develop

printf 'not really png\n' > apps/extension/src/assets/hero.png
git add apps/extension/src/assets/hero.png
git commit -q -m "add lfs asset"

printf 'follow-up\n' >> README.md
git add README.md
git commit -q -m "follow-up change"

fakebin="$tmpdir/bin"
mkdir -p "$fakebin"

cat > "$fakebin/git-lfs" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "${LFS_LOG_FILE:?}"
exit 0
EOF
chmod +x "$fakebin/git-lfs"

cat > "$fakebin/gh" <<'EOF'
#!/usr/bin/env bash
exit 1
EOF
chmod +x "$fakebin/gh"

lfs_log="$tmpdir/lfs.log"
stdin_line="refs/heads/feature/lfs-scan $(git rev-parse HEAD) refs/heads/feature/lfs-scan 0000000000000000000000000000000000000000"

set +e
printf '%s\n' "$stdin_line" | PATH="$fakebin:$PATH" LFS_LOG_FILE="$lfs_log" \
  bash "$root_dir/infra/githooks/pre-push" fork https://example.com/fork.git >/dev/null 2>"$tmpdir/pre-push.stderr"
exit_code=$?
set -e

if [ "$exit_code" -ne 0 ]; then
  echo "pre-push hook exited with $exit_code" >&2
  cat "$tmpdir/pre-push.stderr" >&2
  exit 1
fi

if [ ! -s "$lfs_log" ]; then
  echo "expected git lfs pre-push to run for earlier LFS commit on new branch push" >&2
  cat "$tmpdir/pre-push.stderr" >&2
  exit 1
fi

echo "pre-push regression tests passed"
