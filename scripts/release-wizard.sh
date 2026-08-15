#!/usr/bin/env bash
#
# scripts/release-wizard.sh
# Walks through releasing tickcats v0.6.0 to GitHub Releases + Homebrew tap.
# Run from the tickcats repo root.
#
# Everything above the "STAGES" marker is the wizard library — do not edit.

set -euo pipefail

# ──────────────────────────────────────────────────────────────────────────
# Wizard library
# ──────────────────────────────────────────────────────────────────────────

if [[ -t 1 ]] && command -v tput >/dev/null 2>&1 && [[ "$(tput colors 2>/dev/null || echo 0)" -ge 8 ]]; then
  BOLD=$(tput bold); DIM=$(tput dim); RESET=$(tput sgr0)
  BLUE=$(tput setaf 4); GREEN=$(tput setaf 2); YELLOW=$(tput setaf 3); RED=$(tput setaf 1)
else
  BOLD=""; DIM=""; RESET=""; BLUE=""; GREEN=""; YELLOW=""; RED=""
fi

TOTAL_STAGES=0
_STAGE_INDEX=0
ENV_FILE="${ENV_FILE:-.env}"
WRITTEN_ENV=()
WRITTEN_SECRET=()
SKIPPED=()

_clear() { [[ -t 1 ]] || return 0; if command -v tput >/dev/null 2>&1; then tput clear; else printf '\033[2J\033[3J\033[H'; fi; }
banner() { _clear; printf '\n%s%s  %s%s\n' "$BOLD" "$BLUE" "$1" "$RESET"; printf '%s  %s stages%s\n\n' "$DIM" "$TOTAL_STAGES" "$RESET"; printf '%s  You drive the browser; this wizard tells you exactly what to do.\n  Stop any time with Ctrl-C and re-run — idempotent steps are skipped.%s\n' "$DIM" "$RESET"; pause "Ready to start?"; }
stage() { _clear; _STAGE_INDEX=$((_STAGE_INDEX + 1)); printf '\n%s%s▸ Stage %s/%s · %s%s\n' "$BOLD" "$BLUE" "$_STAGE_INDEX" "$TOTAL_STAGES" "$1" "$RESET"; }
say()     { printf '  %s\n' "$1"; }
step()    { printf '  %s•%s %s\n' "$BLUE" "$RESET" "$1"; }
note()    { printf '  %s%s%s\n' "$DIM" "$1" "$RESET"; }
warn()    { printf '  %s⚠ %s%s\n' "$YELLOW" "$1" "$RESET"; }
ok()      { printf '  %s✓ %s%s\n' "$GREEN" "$1" "$RESET"; }
pause()   { printf '  %s%s%s ' "$DIM" "${1:-Press Enter to continue}" "$RESET"; read -r _ || true; }
confirm() { local r=""; printf '  %s? %s [y/N] ' "$YELLOW" "$1"; read -r r || true; [[ "$r" =~ ^[Yy] ]]; }
open_url() {
  local url="$1"
  printf '  %s↗ opening%s %s\n' "$GREEN" "$RESET" "$url"
  { if command -v wslview >/dev/null 2>&1; then wslview "$url"
    elif command -v explorer.exe >/dev/null 2>&1; then explorer.exe "$url"
    elif command -v xdg-open >/dev/null 2>&1; then xdg-open "$url"
    elif command -v open >/dev/null 2>&1; then open "$url"
    else warn "couldn't open browser — visit manually: $url"; fi
  } >/dev/null 2>&1 || warn "couldn't open browser — visit manually: $url"
}
_existing() { [[ -f "$ENV_FILE" ]] || return 1; local l; l=$(grep -E "^${1}=" "$ENV_FILE" | tail -n1) || return 1; printf '%s' "${l#*=}"; }
ask() {
  local key="$1" prompt="$2" current input
  current=$(_existing "$key" || true)
  [[ -n "$current" ]] && printf '  %s%s%s %s[Enter keeps current]%s ' "$BOLD" "$prompt" "$RESET" "$DIM" "$RESET" || printf '  %s%s%s ' "$BOLD" "$prompt" "$RESET"
  read -r input || true
  [[ -z "$input" && -n "$current" ]] && input="$current"
  printf -v "$key" '%s' "$input"
}
ask_secret() {
  local key="$1" prompt="$2" current input
  current=$(_existing "$key" || true)
  [[ -n "$current" ]] && printf '  %s%s%s %s[Enter keeps current]%s ' "$BOLD" "$prompt" "$RESET" "$DIM" "$RESET" || printf '  %s%s%s ' "$BOLD" "$prompt" "$RESET"
  read -rs input || true; printf '\n'
  [[ -z "$input" && -n "$current" ]] && input="$current"
  printf -v "$key" '%s' "$input"
}
write_env() {
  local key="$1" value="$2" tmp
  touch "$ENV_FILE"; tmp=$(mktemp)
  grep -vE "^${key}=" "$ENV_FILE" > "$tmp" || true
  printf '%s=%s\n' "$key" "$value" >> "$tmp"; mv "$tmp" "$ENV_FILE"
  WRITTEN_ENV+=("$key"); ok "wrote $key → $ENV_FILE"
}
set_secret() {
  local name="$1" value="$2"
  if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
    if printf '%s' "$value" | gh secret set "$name" >/dev/null 2>&1; then
      WRITTEN_SECRET+=("$name"); ok "set GitHub secret $name"; return
    fi
  fi
  SKIPPED+=("GitHub secret $name — run: gh secret set $name")
  warn "skipped secret $name (gh not ready) — set it manually after"
}
finish() {
  _clear; printf '\n%s%s  ✓ Done%s\n' "$BOLD" "$GREEN" "$RESET"
  (( ${#WRITTEN_ENV[@]} ))    && note "wrote ${#WRITTEN_ENV[@]} value(s) to $ENV_FILE: ${WRITTEN_ENV[*]}"
  (( ${#WRITTEN_SECRET[@]} )) && note "set ${#WRITTEN_SECRET[@]} GitHub secret(s): ${WRITTEN_SECRET[*]}"
  if (( ${#SKIPPED[@]} )); then
    printf '\n'; warn "still to do by hand:"; for s in "${SKIPPED[@]}"; do note "  - $s"; done
  fi
  printf '\n'
}

# ──────────────────────────────────────────────────────────────────────────
# STAGES
# ──────────────────────────────────────────────────────────────────────────

VERSION="v0.6.0"
BRANCH="plan/go-to-rust-migration"
REPO_OWNER="dawidsok"
TAP_REPO="${REPO_OWNER}/homebrew-tap"
TICKCATS_REPO="${REPO_OWNER}/tickcats"

TOTAL_STAGES=5

banner "tickcats ${VERSION} — GitHub Releases + Homebrew"

# ─── Stage 1: GitHub CLI ──────────────────────────────────────────────────

stage "GitHub CLI — auth check"
say "The wizard uses 'gh' to set secrets and push to GitHub."
say ""

if ! command -v gh >/dev/null 2>&1; then
  warn "'gh' is not installed."
  step "Install it: https://cli.github.com"
  open_url "https://cli.github.com"
  pause "Press Enter once 'gh' is installed and in your PATH"
fi

if ! gh auth status >/dev/null 2>&1; then
  warn "Not logged in to GitHub CLI."
  step "Running: gh auth login"
  gh auth login
fi

ok "gh is authenticated as $(gh api user --jq .login)"

# ─── Stage 2: Homebrew tap repository ────────────────────────────────────

stage "Homebrew tap — ${TAP_REPO}"
say "The release workflow pushes a generated formula to ${TAP_REPO}."
say "Users will install via:  brew install ${REPO_OWNER}/tap/tickcats"
say ""

if gh repo view "${TAP_REPO}" >/dev/null 2>&1; then
  ok "${TAP_REPO} already exists — skipping creation"
else
  say "Creating public repo ${TAP_REPO} …"
  gh repo create "${TAP_REPO}" \
    --public \
    --description "Homebrew tap for TickCats — keyboard-first local kanban" \
    --add-readme
  ok "Created ${TAP_REPO}"

  say "Initialising Formula/ directory …"
  TAPDIR="$(mktemp -d)/homebrew-tap"
  git clone "https://github.com/${TAP_REPO}.git" "$TAPDIR"
  mkdir -p "${TAPDIR}/Formula"
  printf '# Formula files are generated by the tickcats release workflow.\n' \
    > "${TAPDIR}/Formula/.keep"
  cd "$TAPDIR"
  git add Formula/
  git commit -m "chore: add Formula directory"
  git push
  cd - >/dev/null
  rm -rf "$TAPDIR"
  ok "Formula/ directory pushed"
fi

note "Tap URL: https://github.com/${TAP_REPO}"

# ─── Stage 3: HOMEBREW_TAP_TOKEN ─────────────────────────────────────────

stage "HOMEBREW_TAP_TOKEN — GitHub secret"
say "The release workflow needs a token that can push to ${TAP_REPO}."
say "You'll create a Classic token with 'repo' scope (or a Fine-grained"
say "token with Contents write access on ${TAP_REPO})."
say ""

if gh secret list --repo "${TICKCATS_REPO}" 2>/dev/null | grep -q "HOMEBREW_TAP_TOKEN"; then
  ok "HOMEBREW_TAP_TOKEN is already set on ${TICKCATS_REPO} — skipping"
else
  step "Opening GitHub token settings …"
  open_url "https://github.com/settings/tokens/new?scopes=repo&description=tickcats+homebrew+tap"
  say ""
  say "On the token page:"
  step "Note name:  tickcats homebrew tap"
  step "Expiration: No expiration (or 1 year)"
  step "Scopes:     tick the top-level 'repo' checkbox"
  step "Click 'Generate token' — copy the token (shown only once)"
  say ""
  ask_secret HOMEBREW_TAP_TOKEN "Paste the token:"

  if [[ -z "${HOMEBREW_TAP_TOKEN:-}" ]]; then
    warn "No token entered. You can set it later with:"
    note "  gh secret set HOMEBREW_TAP_TOKEN --repo ${TICKCATS_REPO}"
    SKIPPED+=("HOMEBREW_TAP_TOKEN — set it with: gh secret set HOMEBREW_TAP_TOKEN --repo ${TICKCATS_REPO}")
  else
    # Verify the token can push to the tap repo
    if git ls-remote "https://x-access-token:${HOMEBREW_TAP_TOKEN}@github.com/${TAP_REPO}.git" >/dev/null 2>&1; then
      ok "Token verified — can push to ${TAP_REPO}"
    else
      warn "Token check failed. It may still work; proceeding anyway."
    fi
    set_secret HOMEBREW_TAP_TOKEN "$HOMEBREW_TAP_TOKEN"
  fi
fi

# ─── Stage 4: Merge, tag, and push ───────────────────────────────────────

stage "Merge ${BRANCH} → main  ·  tag ${VERSION}  ·  push"
say "This is the point of no return — the release is triggered by pushing"
say "the tag. GitHub Actions will build 5 platform binaries (~10 min)."
say ""

note "Merge:  git checkout main && git merge ${BRANCH}"
note "Tag:    git tag -a ${VERSION} -m 'Release ${VERSION}'"
note "Push:   git push origin main && git push origin ${VERSION}"
say ""

if ! confirm "Ready to merge, tag, and push ${VERSION}?"; then
  warn "Aborted by user. Re-run the wizard when ready."
  exit 0
fi

# Merge
git checkout main
if git merge-base --is-ancestor "${BRANCH}" main 2>/dev/null; then
  ok "${BRANCH} is already merged into main"
else
  git merge --no-ff "${BRANCH}" -m "chore: merge ${BRANCH} for ${VERSION} release"
  ok "Merged ${BRANCH} into main"
fi

# Tag
if git rev-parse "${VERSION}" >/dev/null 2>&1; then
  ok "Tag ${VERSION} already exists — skipping"
else
  git tag -a "${VERSION}" -m "Release ${VERSION} — Rust implementation"
  ok "Created tag ${VERSION}"
fi

# Push
git push origin main
git push origin "${VERSION}"
ok "Pushed main and ${VERSION} to origin"

# ─── Stage 5: Monitor the release ────────────────────────────────────────

stage "Monitor — GitHub Actions"
say "The release workflow is now running. It will:"
say "  1. Build binaries for 5 platforms       (~8 min)"
say "  2. Verify archive contents"
say "  3. Create the GitHub Release with 6 files + checksums.txt"
say "  4. Push Formula/tickcats.rb to ${TAP_REPO}"
say ""
note "Watch the run at:"
open_url "https://github.com/${TICKCATS_REPO}/actions"
say ""
note "When finished, users can install with:"
note "  brew install ${REPO_OWNER}/tap/tickcats"
note "  # or direct download from:"
note "  https://github.com/${TICKCATS_REPO}/releases/tag/${VERSION}"
say ""
pause "Press Enter once the workflow completes to finish the wizard"

# Optionally verify the release was created
if gh release view "${VERSION}" --repo "${TICKCATS_REPO}" >/dev/null 2>&1; then
  ok "GitHub Release ${VERSION} confirmed"
  ASSET_COUNT=$(gh release view "${VERSION}" --repo "${TICKCATS_REPO}" --json assets --jq '.assets | length')
  note "Assets uploaded: ${ASSET_COUNT} (expect 6: 4 × tar.gz, 1 × zip, checksums.txt)"
else
  warn "Release not found yet — the workflow may still be running"
fi

finish
