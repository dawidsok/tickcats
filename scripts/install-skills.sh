#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
skills_src="$repo_root/skills"

names=(
  "Claude Code"
  "OpenAI Codex CLI"
  "Pi"
  "Shared Agent Skills"
)
paths=(
  "$HOME/.claude/skills"
  "$HOME/.codex/skills"
  "$HOME/.pi/agent/skills"
  "$HOME/.agents/skills"
)

install_to() {
  local target="$1"
  local skill name count=0

  mkdir -p "$target"
  shopt -s nullglob
  for skill in "$skills_src"/*; do
    [[ -d "$skill" && -f "$skill/SKILL.md" ]] || continue
    name="$(basename "$skill")"
    rm -rf "$target/$name"
    cp -R "$skill" "$target/$name"
    count=$((count + 1))
  done
  shopt -u nullglob

  printf 'Installed %d skills to %s\n' "$count" "$target"
}

print_menu() {
  echo "Install TickCats skills for which harness?"
  for i in "${!names[@]}"; do
    printf '  %d) %s -> %s\n' "$((i + 1))" "${names[$i]}" "${paths[$i]}"
  done
  echo "  a) All"
  echo "  q) Quit"
  printf 'Select one or more numbers (comma-separated): '
}

[[ -d "$skills_src" ]] || { echo "skills directory not found: $skills_src" >&2; exit 1; }

print_menu
read -r reply
selected=()

case "$reply" in
  q|Q|quit|Quit|QUIT) exit 0 ;;
  a|A|all|All|ALL) selected=(1 2 3 4) ;;
  *)
    IFS=', ' read -r -a selected <<< "$reply"
    ;;
esac

[[ ${#selected[@]} -gt 0 ]] || { echo "No harness selected" >&2; exit 1; }

for choice in "${selected[@]}"; do
  [[ "$choice" =~ ^[0-9]+$ ]] || { echo "Invalid choice: $choice" >&2; exit 1; }
  (( choice >= 1 && choice <= ${#paths[@]} )) || { echo "Invalid choice: $choice" >&2; exit 1; }
  install_to "${paths[$((choice - 1))]}"
done
