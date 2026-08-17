#!/usr/bin/env bash
# scripts/release.sh — build and publish a tickcats release locally.
#
# Builds both macOS targets natively (arm64 + amd64).
# Requires: cargo, rustup, gh (authenticated), sha256sum or shasum
#
# Usage:
#   ./scripts/release.sh          # version from Cargo.toml
#   ./scripts/release.sh v0.6.3   # explicit (must match Cargo.toml)

set -euo pipefail
cd "$(dirname "$0")/.."

# ── version ──────────────────────────────────────────────────────────────────
CARGO_VERSION=$(grep '^version' Cargo.toml | head -1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')
VERSION="${1:-v$CARGO_VERSION}"
VERSION="${VERSION#v}"
TAG="v$VERSION"

[[ "$VERSION" == "$CARGO_VERSION" ]] || {
  echo "Error: $TAG doesn't match Cargo.toml version $CARGO_VERSION" >&2; exit 1
}

# ── prereqs ───────────────────────────────────────────────────────────────────
need() { command -v "$1" >/dev/null 2>&1 || { echo "Error: '$1' not found — $2" >&2; exit 1; }; }
need cargo  "install Rust: https://rustup.rs"
need rustup "install Rust: https://rustup.rs"
need gh     "install GitHub CLI: https://cli.github.com"

gh auth status >/dev/null 2>&1 || { echo "Error: run 'gh auth login' first" >&2; exit 1; }

SHA256() { if command -v sha256sum >/dev/null 2>&1; then sha256sum "$@"; else shasum -a 256 "$@"; fi; }

echo "==> Releasing $TAG (macOS arm64 + amd64)"
echo ""

# ── targets ───────────────────────────────────────────────────────────────────
rustup target add aarch64-apple-darwin x86_64-apple-darwin 2>/dev/null || true

# ── build ─────────────────────────────────────────────────────────────────────
DIST="dist/$TAG"
rm -rf "$DIST" && mkdir -p "$DIST"

for target in aarch64-apple-darwin x86_64-apple-darwin; do
  echo "==> cargo build --release --target $target"
  cargo build --release --target "$target"
done

# ── package ───────────────────────────────────────────────────────────────────
echo ""
echo "==> Packaging"

pack() {
  local target="$1" os="$2" arch="$3"
  local name="tickcats_${TAG}_${os}_${arch}"
  local dir="$DIST/$name"
  mkdir -p "$dir/completions"
  cp "target/$target/release/tickcats" "$dir/"
  cp LICENSE README.md "$dir/"
  cp completions/tickcats.bash completions/_tickcats.zsh completions/tickcats.fish "$dir/completions/"
  tar -czf "$DIST/$name.tar.gz" -C "$DIST" "$name"
  rm -rf "$dir"
  echo "  $name.tar.gz"
}

pack aarch64-apple-darwin darwin arm64
pack x86_64-apple-darwin  darwin amd64

# ── verify ────────────────────────────────────────────────────────────────────
echo ""
bash scripts/check-release-archive.sh "$DIST"/*.tar.gz

# ── checksums ─────────────────────────────────────────────────────────────────
echo ""
echo "==> Checksums"
(cd "$DIST" && SHA256 * > checksums.txt && cat checksums.txt)

# ── github release ────────────────────────────────────────────────────────────
echo ""
echo "==> GitHub release $TAG"
if gh release view "$TAG" >/dev/null 2>&1; then
  gh release upload "$TAG" "$DIST"/* --clobber
  echo "    uploaded to existing release"
else
  gh release create "$TAG" "$DIST"/* --title "$TAG" --generate-notes
fi
echo "    https://github.com/dawidsok/tickcats/releases/tag/$TAG"

# ── homebrew tap ──────────────────────────────────────────────────────────────
echo ""
echo "==> Homebrew tap"

TOKEN="${HOMEBREW_TAP_TOKEN:-$(gh auth token 2>/dev/null || true)}"
if [[ -z "$TOKEN" ]]; then
  echo "    Skipped — set HOMEBREW_TAP_TOKEN or ensure gh has repo access to dawidsok/homebrew-tap"
else
  TAPDIR=$(mktemp -d)
  trap 'rm -rf "$TAPDIR"' EXIT
  git clone "https://x-access-token:${TOKEN}@github.com/dawidsok/homebrew-tap.git" "$TAPDIR" -q
  SHA() { SHA256 "$DIST/tickcats_${TAG}_${1}_${2}.tar.gz" | cut -d' ' -f1; }

  cat > "$TAPDIR/tickcats.rb" << FORMULA
class Tickcats < Formula
  desc "Keyboard-first local kanban board for solo developers"
  homepage "https://github.com/dawidsok/tickcats"
  license "MIT"
  version "$VERSION"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/dawidsok/tickcats/releases/download/${TAG}/tickcats_${TAG}_darwin_arm64.tar.gz"
      sha256 "$(SHA darwin arm64)"
    else
      url "https://github.com/dawidsok/tickcats/releases/download/${TAG}/tickcats_${TAG}_darwin_amd64.tar.gz"
      sha256 "$(SHA darwin amd64)"
    end
  end

  def install
    bin.install "tickcats"
    bash_completion.install "completions/tickcats.bash" => "tickcats"
    zsh_completion.install "completions/_tickcats.zsh" => "_tickcats"
    fish_completion.install "completions/tickcats.fish"
  end

  test do
    system "#{bin}/tickcats", "--path", "#{testpath}/.tickcats", "init"
    assert_predicate testpath/".tickcats/backlog", :directory?
  end
end
FORMULA

  cd "$TAPDIR"
  git config user.name "$(git config --global user.name 2>/dev/null || echo release)"
  git config user.email "$(git config --global user.email 2>/dev/null || echo release@tickcats)"
  git add tickcats.rb
  git diff --staged --quiet || git commit -m "chore: bump tickcats to $TAG"
  git push -q
  echo "    https://github.com/dawidsok/homebrew-tap/blob/main/tickcats.rb"
fi

echo ""
echo "==> Done — $TAG"
