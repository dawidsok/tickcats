#!/usr/bin/env bash
# scripts/release.sh — build and publish a tickcats release locally.
#
# Usage:
#   ./scripts/release.sh          # version read from Cargo.toml
#   ./scripts/release.sh v0.6.3   # explicit version (must match Cargo.toml)
#
# Requires: cargo, rustup, cross (+Docker), gh (authenticated), sha256sum/shasum
# Optional: HOMEBREW_TAP_TOKEN env var (falls back to gh auth token)

set -euo pipefail
cd "$(dirname "$0")/.."

# ── version ──────────────────────────────────────────────────────────────────
CARGO_VERSION=$(grep '^version' Cargo.toml | head -1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')
VERSION="${1:-v$CARGO_VERSION}"
VERSION="${VERSION#v}"   # strip leading v for archives
TAG="v$VERSION"

if [[ "$VERSION" != "$CARGO_VERSION" ]]; then
  echo "Error: argument $TAG doesn't match Cargo.toml version $CARGO_VERSION" >&2
  exit 1
fi

# ── prerequisites ─────────────────────────────────────────────────────────────
need() { command -v "$1" >/dev/null 2>&1 || { echo "Error: '$1' not found — $2" >&2; exit 1; }; }
need cargo        "install Rust: https://rustup.rs"
need rustup       "install Rust: https://rustup.rs"
need cargo-zigbuild "cargo install cargo-zigbuild --locked"
need zig          "brew install zig"
need gh           "install GitHub CLI: https://cli.github.com"

if ! gh auth status >/dev/null 2>&1; then
  echo "Error: not logged in to GitHub CLI — run: gh auth login" >&2; exit 1
fi
# sha256sum on macOS ships as shasum

echo "==> Releasing $TAG"
echo ""

# ── targets ───────────────────────────────────────────────────────────────────
# darwin targets built natively (no Docker needed); others via cross.
declare -A TARGETS=(
  [darwin-amd64]="x86_64-apple-darwin"
  [darwin-arm64]="aarch64-apple-darwin"
  [linux-amd64]="x86_64-unknown-linux-gnu"
  [linux-arm64]="aarch64-unknown-linux-gnu"
  [windows-amd64]="x86_64-pc-windows-gnu"
)

# Add any missing rustup targets
for key in "${!TARGETS[@]}"; do
  rustup target add "${TARGETS[$key]}" 2>/dev/null || true
done

# ── build ─────────────────────────────────────────────────────────────────────
DIST="$(pwd)/dist/$TAG"
rm -rf "$DIST" && mkdir -p "$DIST"

for key in darwin-amd64 darwin-arm64 linux-amd64 linux-arm64 windows-amd64; do
  target="${TARGETS[$key]}"
  echo "==> Building $key ($target)"
  cargo zigbuild --release --target "$target"
done

# ── package ───────────────────────────────────────────────────────────────────
echo ""
echo "==> Packaging archives"

for key in darwin-amd64 darwin-arm64 linux-amd64 linux-arm64 windows-amd64; do
  target="${TARGETS[$key]}"
  os="${key%-*}"
  arch="${key#*-}"
  name="tickcats_${TAG}_${os}_${arch}"
  dir="$DIST/$name"
  mkdir -p "$dir/completions"

  if [[ "$key" == windows-* ]]; then
    cp "target/$target/release/tickcats.exe" "$dir/"
  else
    cp "target/$target/release/tickcats" "$dir/"
  fi
  cp LICENSE README.md "$dir/"
  cp completions/tickcats.bash completions/_tickcats.zsh completions/tickcats.fish \
     "$dir/completions/"

  if [[ "$key" == windows-* ]]; then
    (cd "$DIST" && zip -qr "$name.zip" "$name") && echo "  $name.zip"
  else
    tar -czf "$DIST/$name.tar.gz" -C "$DIST" "$name" && echo "  $name.tar.gz"
  fi
  rm -rf "$dir"
done

# ── verify ────────────────────────────────────────────────────────────────────
echo ""
echo "==> Verifying archives"
bash scripts/check-release-archive.sh "$DIST"/*.tar.gz

# ── checksums ─────────────────────────────────────────────────────────────────
echo ""
echo "==> Creating checksums"
(cd "$DIST" && SHA256 * > checksums.txt && cat checksums.txt)

# ── github release ────────────────────────────────────────────────────────────
echo ""
echo "==> Creating GitHub release $TAG"
if gh release view "$TAG" >/dev/null 2>&1; then
  echo "    Release $TAG already exists — uploading assets"
  gh release upload "$TAG" "$DIST"/* --clobber
else
  gh release create "$TAG" "$DIST"/* \
    --title "$TAG" \
    --generate-notes
fi
echo "    https://github.com/dawidsok/tickcats/releases/tag/$TAG"

# ── homebrew tap ──────────────────────────────────────────────────────────────
echo ""
echo "==> Updating Homebrew tap"

TAP_REPO="dawidsok/homebrew-tap"
TOKEN="${HOMEBREW_TAP_TOKEN:-}"
if [[ -z "$TOKEN" ]]; then
  # fall back to the gh CLI token (works if it has repo access to the tap)
  TOKEN=$(gh auth token 2>/dev/null || true)
fi

if [[ -z "$TOKEN" ]]; then
  echo "    Skipped — set HOMEBREW_TAP_TOKEN or ensure gh has repo access to $TAP_REPO"
else
  SHA() { SHA256 "$DIST/tickcats_${TAG}_${1}_${2}.tar.gz" | cut -d' ' -f1; }
  DARWIN_ARM64=$(SHA darwin arm64)
  DARWIN_AMD64=$(SHA darwin amd64)
  LINUX_ARM64=$(SHA  linux  arm64)
  LINUX_AMD64=$(SHA  linux  amd64)

  TAPDIR=$(mktemp -d)
  trap 'rm -rf "$TAPDIR"' EXIT
  git clone "https://x-access-token:${TOKEN}@github.com/${TAP_REPO}.git" "$TAPDIR"

  cat > "$TAPDIR/tickcats.rb" << FORMULA
class Tickcats < Formula
  desc "Keyboard-first local kanban board for solo developers"
  homepage "https://github.com/dawidsok/tickcats"
  license "MIT"
  version "$VERSION"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/dawidsok/tickcats/releases/download/${TAG}/tickcats_${TAG}_darwin_arm64.tar.gz"
      sha256 "$DARWIN_ARM64"
    else
      url "https://github.com/dawidsok/tickcats/releases/download/${TAG}/tickcats_${TAG}_darwin_amd64.tar.gz"
      sha256 "$DARWIN_AMD64"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/dawidsok/tickcats/releases/download/${TAG}/tickcats_${TAG}_linux_arm64.tar.gz"
      sha256 "$LINUX_ARM64"
    else
      url "https://github.com/dawidsok/tickcats/releases/download/${TAG}/tickcats_${TAG}_linux_amd64.tar.gz"
      sha256 "$LINUX_AMD64"
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
  git config user.name "$(git config --global user.name 2>/dev/null || echo 'release')"
  git config user.email "$(git config --global user.email 2>/dev/null || echo 'release@tickcats')"
  git add tickcats.rb
  git diff --staged --quiet || git commit -m "chore: bump tickcats to $TAG"
  git push
  echo "    https://github.com/$TAP_REPO/blob/main/tickcats.rb"
fi

echo ""
echo "==> Done — $TAG released"
