#!/usr/bin/env bash
# 更新 PR #188 標題與正文以通過 PR Scope Police（需已安裝 gh 並完成 gh auth login，或設定 GH_TOKEN）
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
GH_BIN="$(command -v gh 2>/dev/null || true)"
if [[ -z "${GH_BIN}" && -x /opt/homebrew/bin/gh ]]; then
  GH_BIN=/opt/homebrew/bin/gh
fi
if [[ -z "${GH_BIN}" && -x /usr/local/bin/gh ]]; then
  GH_BIN=/usr/local/bin/gh
fi
if [[ -z "${GH_BIN}" ]]; then
  echo '找不到 gh，請安裝 GitHub CLI 後再執行。' >&2
  exit 1
fi
exec "${GH_BIN}" pr edit 188 \
  --repo nurockplayer/tachigo \
  --title '[frontend] Extension demo — coupon shop UI' \
  --body-file "${ROOT}/.github/pr-188-github-body.md"
