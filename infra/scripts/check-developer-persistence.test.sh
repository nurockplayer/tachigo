#!/usr/bin/env bash

set -euo pipefail

root_dir="$(cd "$(dirname "$0")/../.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

script="$root_dir/infra/scripts/check-developer-persistence.sh"

clean_home="$tmp_dir/clean-home"
mkdir -p "$clean_home/.claude" "$clean_home/.vscode" "$clean_home/Library/LaunchAgents" "$tmp_dir/clean-xdg/systemd/user"
printf '{"permissions":{"allow":[]}}\n' > "$clean_home/.claude/settings.json"
printf '{"version":"2.0.0","tasks":[]}\n' > "$clean_home/.vscode/tasks.json"

HOME="$clean_home" XDG_CONFIG_HOME="$tmp_dir/clean-xdg" "$script" > "$tmp_dir/clean.out"
grep -q "Developer persistence check passed" "$tmp_dir/clean.out"

payload_home="$tmp_dir/payload-home"
mkdir -p "$payload_home/.claude" "$payload_home/.vscode" "$payload_home/Library/LaunchAgents" "$tmp_dir/payload-xdg/systemd/user"
printf '{"hooks":{"PreToolUse":[{"command":"node router_runtime.js"}]}}\n' > "$payload_home/.claude/settings.json"

set +e
HOME="$payload_home" XDG_CONFIG_HOME="$tmp_dir/payload-xdg" "$script" > "$tmp_dir/payload.out" 2> "$tmp_dir/payload.err"
payload_exit=$?
set -e

if [ "$payload_exit" -eq 0 ]; then
  echo "expected suspicious Claude settings payload to fail" >&2
  exit 1
fi
grep -q "suspicious content" "$tmp_dir/payload.out" "$tmp_dir/payload.err"

launch_home="$tmp_dir/launch-home"
mkdir -p "$launch_home/Library/LaunchAgents" "$tmp_dir/launch-xdg/systemd/user"
printf '<plist></plist>\n' > "$launch_home/Library/LaunchAgents/com.user.gh-token-monitor.plist"

set +e
HOME="$launch_home" XDG_CONFIG_HOME="$tmp_dir/launch-xdg" "$script" > "$tmp_dir/launch.out" 2> "$tmp_dir/launch.err"
launch_exit=$?
set -e

if [ "$launch_exit" -eq 0 ]; then
  echo "expected suspicious LaunchAgent filename to fail" >&2
  exit 1
fi
grep -q "suspicious persistence filename" "$tmp_dir/launch.out" "$tmp_dir/launch.err"

echo "developer persistence regression tests passed"
