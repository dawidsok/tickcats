#!/usr/bin/env bash
# Verify that release archives contain the required files.
# Usage: scripts/check-release-archive.sh <archive> [<archive>...]
# Returns 0 if all archives pass, 1 otherwise.
set -euo pipefail

REQUIRED_FILES=(
    "LICENSE"
    "README.md"
    "completions/tickcats.bash"
    "completions/_tickcats.zsh"
    "completions/tickcats.fish"
)

pass=0
fail=0

for archive in "$@"; do
    name="$(basename "$archive")"
    tmp="$(mktemp -d)"
    trap "rm -rf '$tmp'" EXIT

    case "$archive" in
        *.tar.gz) tar -xzf "$archive" -C "$tmp" ;;
        *.zip)    unzip -q "$archive" -d "$tmp" ;;
        *)
            echo "SKIP  $name (unknown format)"
            continue
            ;;
    esac

    # The archive should contain exactly one top-level directory
    top="$(ls "$tmp")"
    if [[ "$(echo "$top" | wc -l | tr -d ' ')" -ne 1 ]]; then
        echo "FAIL  $name — expected one top-level directory, got: $top"
        fail=$((fail + 1))
        continue
    fi
    root="$tmp/$top"

    ok=true
    # Check required files
    for f in "${REQUIRED_FILES[@]}"; do
        if [[ ! -f "$root/$f" ]]; then
            echo "FAIL  $name — missing $f"
            ok=false
        fi
    done

    # Check binary (tickcats or tickcats.exe)
    if [[ -f "$root/tickcats" ]]; then
        if [[ ! -x "$root/tickcats" ]]; then
            echo "FAIL  $name — tickcats binary is not executable"
            ok=false
        fi
    elif [[ -f "$root/tickcats.exe" ]]; then
        : # Windows binary; executable bit not applicable
    else
        echo "FAIL  $name — missing tickcats or tickcats.exe binary"
        ok=false
    fi

    # Verify archive name follows the approved template
    # tickcats_<version>_<os>_<arch>.{tar.gz,zip}
    base="${name%.tar.gz}"
    base="${base%.zip}"
    if [[ "$base" =~ ^tickcats_v[0-9]+\.[0-9]+\.[0-9]+[^_]*_(darwin|linux|windows)_(amd64|arm64)$ ]]; then
        : # correct
    else
        echo "FAIL  $name — name does not match tickcats_<version>_<os>_<arch>"
        ok=false
    fi

    if $ok; then
        echo "PASS  $name"
        pass=$((pass + 1))
    else
        fail=$((fail + 1))
    fi
done

echo ""
echo "$pass passed, $fail failed"
[[ $fail -eq 0 ]]
