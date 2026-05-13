#!/usr/bin/env bash

set -euo pipefail

home_dir="${HOME:?HOME is required}"
xdg_config_home="${XDG_CONFIG_HOME:-$home_dir/.config}"

patterns='router_init\.js|router_runtime\.js|tanstack_runner\.js|git-tanstack|getsession\.org|83\.142\.209\.194|gh-token-monitor|IfYouRevokeThisTokenItWillWipeTheComputerOfTheOwner|Shai-Hulud: Here We Go Again|79ac49eedf774dd4b0cfa308722bc463cfe5885c|transformers\.pyz|pgmonitor\.py|pgsql-monitor\.service'
filename_patterns='gh-token-monitor|router_init\.js|router_runtime\.js|tanstack_runner\.js|transformers\.pyz|pgmonitor\.py|pgsql-monitor\.service'

problems=0

check_file_content() {
  local file="$1"
  local label="$2"

  if [ ! -f "$file" ]; then
    return 0
  fi

  if grep -Eqi "$patterns" "$file"; then
    echo "suspicious content: $label"
    problems=1
  fi
}

check_directory_filenames() {
  local dir="$1"
  local label="$2"

  if [ ! -d "$dir" ]; then
    return 0
  fi

  while IFS= read -r -d '' file; do
    if printf '%s\n' "$(basename "$file")" | grep -Eqi "$filename_patterns"; then
      echo "suspicious persistence filename: $label/$(basename "$file")"
      problems=1
    fi
  done < <(find "$dir" -maxdepth 1 -type f -print0 2>/dev/null)
}

check_file_content "$home_dir/.claude/settings.json" "~/.claude/settings.json"
check_file_content "$home_dir/.claude/settings.local.json" "~/.claude/settings.local.json"
check_file_content "$home_dir/.vscode/tasks.json" "~/.vscode/tasks.json"
check_directory_filenames "$home_dir/Library/LaunchAgents" "~/Library/LaunchAgents"
check_directory_filenames "$xdg_config_home/systemd/user" "\${XDG_CONFIG_HOME:-~/.config}/systemd/user"

if [ "$problems" -ne 0 ]; then
  echo "Developer persistence check failed"
  exit 1
fi

echo "Developer persistence check passed"
