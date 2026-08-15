#!/usr/bin/env bash
# Install TickCats agent skills to one or more harness skill directories.
#
# Usage:
#   ./scripts/install-skills.sh              # interactive menu
#   ./scripts/install-skills.sh --path <dir> # install directly to <dir>
#   ./scripts/install-skills.sh --local      # install to all local (~/.)-style paths

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
skills_src="$repo_root/skills"

[[ -d "$skills_src" ]] || { echo "skills directory not found: $skills_src" >&2; exit 1; }

# ── harness entries ──────────────────────────────────────────────────────────
# Each entry: "display name|global path|local path"
# Local path is relative to cwd (the project root where you run the script).
entries=(
  "Claude Code|$HOME/.claude/skills|.claude/skills"
  "OpenAI Codex CLI|$HOME/.codex/skills|.codex/skills"
  "Pi|$HOME/.pi/agent/skills|.pi/agent/skills"
  "Shared Agent Skills|$HOME/.agents/skills|.agents/skills"
)

# ── install_to <target_dir> ──────────────────────────────────────────────────
install_to() {
  local target="$1" skill name count=0
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
  printf 'Installed %d skill(s) to %s\n' "$count" "$target"
}

# ── --path <dir> flag ────────────────────────────────────────────────────────
if [[ "${1:-}" == "--path" ]]; then
  [[ -n "${2:-}" ]] || { echo "Usage: $0 --path <dir>" >&2; exit 1; }
  install_to "$2"
  exit 0
fi

# ── interactive menu ─────────────────────────────────────────────────────────
print_menu() {
  echo "Install TickCats skills for which harness?"
  echo ""
  echo "  Global (installs to ~/ paths):"
  local i=1
  for entry in "${entries[@]}"; do
    IFS='|' read -r name global_path _ <<< "$entry"
    printf '  %d) %-22s -> %s\n' "$i" "$name" "$global_path"
    i=$((i + 1))
  done
  echo ""
  echo "  Local (installs relative to current directory):"
  local j=$((${#entries[@]} + 1))
  for entry in "${entries[@]}"; do
    IFS='|' read -r name _ local_path <<< "$entry"
    printf '  %d) %-22s -> %s\n' "$j" "$name (local)" "$local_path"
    j=$((j + 1))
  done
  echo ""
  echo "  a) All global"
  echo "  l) All local"
  echo "  q) Quit"
  echo ""
  printf 'Select (number, comma-separated, a/l/q): '
}

print_menu
read -r reply

n=${#entries[@]}

case "$reply" in
  q|Q|quit|Quit|QUIT) exit 0 ;;
  a|A|all|All|ALL)
    for entry in "${entries[@]}"; do
      IFS='|' read -r _ global_path _ <<< "$entry"
      install_to "$global_path"
    done
    exit 0
    ;;
  l|L|local|Local|LOCAL)
    for entry in "${entries[@]}"; do
      IFS='|' read -r _ _ local_path <<< "$entry"
      install_to "$local_path"
    done
    exit 0
    ;;
esac

IFS=', ' read -r -a selected <<< "$reply"
[[ ${#selected[@]} -gt 0 ]] || { echo "No harness selected" >&2; exit 1; }

for choice in "${selected[@]}"; do
  [[ "$choice" =~ ^[0-9]+$ ]] || { echo "Invalid choice: $choice" >&2; exit 1; }
  total=$(( n * 2 ))
  (( choice >= 1 && choice <= total )) || { echo "Invalid choice: $choice" >&2; exit 1; }
  if (( choice <= n )); then
    IFS='|' read -r _ global_path _ <<< "${entries[$((choice - 1))]}"
    install_to "$global_path"
  else
    IFS='|' read -r _ _ local_path <<< "${entries[$((choice - n - 1))]}"
    install_to "$local_path"
  fi
done
