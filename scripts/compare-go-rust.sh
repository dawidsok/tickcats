#!/usr/bin/env bash
set -uo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
scenario_file="$repo_root/tests/contracts/scenarios.tsv"
expected_root="$repo_root/tests/contracts/expected"
expected_gaps_file="$repo_root/tests/contracts/expected-gaps.txt"
allow_gaps=false
self_test=false

usage() {
  echo "usage: $0 [--allow-gaps] [--expected-gaps <file>] [--scenarios <file>] [--self-test]" >&2
}

while (($#)); do
  case "$1" in
    --allow-gaps) allow_gaps=true; shift ;;
    --expected-gaps)
      [[ $# -ge 2 ]] || { usage; exit 2; }
      expected_gaps_file="$2"; shift 2
      ;;
    --scenarios)
      [[ $# -ge 2 ]] || { usage; exit 2; }
      scenario_file="$2"; shift 2
      ;;
    --self-test) self_test=true; shift ;;
    *) usage; exit 2 ;;
  esac
done

snapshot_tree() {
  local root="$1" output="$2"
  (
    cd "$root" || exit 1
    find . -mindepth 1 -print | LC_ALL=C sort | while IFS= read -r path; do
      if [[ -L "$path" ]]; then
        printf 'L\t%s\t%s\n' "$path" "$(readlink "$path")"
      elif [[ -d "$path" ]]; then
        printf 'D\t%s\n' "$path"
      elif [[ -f "$path" ]]; then
        read -r checksum size _ < <(cksum "$path")
        printf 'F\t%s\t%s\t%s\n' "$path" "$checksum" "$size"
      fi
    done
  ) >"$output"
}

normalize_stream() {
  local board="$1" input="$2" output="$3"
  python3 - "$board" "$input" "$output" <<'PY'
from pathlib import Path
import os
import sys
board, source, target = sys.argv[1:]
data = Path(source).read_bytes()
for value in {board, os.path.normpath(board), str(Path(board).resolve())}:
    data = data.replace(value.encode(), b"{BOARD}")
Path(target).write_bytes(data)
PY
}

run_command() {
  local binary="$1" board="$2" workspace="$3" output_dir="$4"
  shift 4
  local -a args=("$@")
  local i
  for i in "${!args[@]}"; do
    [[ "${args[$i]}" == "{board}" ]] && args[$i]="$board"
  done

  mkdir -p "$output_dir"
  snapshot_tree "$workspace" "$output_dir/tree.before"
  "$binary" "${args[@]}" >"$output_dir/stdout.raw" 2>"$output_dir/stderr.raw"
  printf '%s\n' "$?" >"$output_dir/status"
  normalize_stream "$board" "$output_dir/stdout.raw" "$output_dir/stdout"
  normalize_stream "$board" "$output_dir/stderr.raw" "$output_dir/stderr"
  snapshot_tree "$workspace" "$output_dir/tree.after"
}

compare_file() {
  local label="$1" left="$2" right="$3"
  if cmp -s "$left" "$right"; then
    return 0
  fi
  echo "--- mismatch: $label" >&2
  diff -u "$left" "$right" >&2 || true
  return 1
}

check_filesystem_assertions() {
  local scenario="$1" assertions="$2" workspace="$3"
  local failed=0
  while IFS=$'\t' read -r operation path value || [[ -n "$operation$path$value" ]]; do
    [[ -z "$operation" || "$operation" == \#* ]] && continue
    local assertion_failed=0
    case "$operation" in
      dir) [[ -d "$workspace/$path" ]] || assertion_failed=1 ;;
      file) [[ -f "$workspace/$path" ]] || assertion_failed=1 ;;
      not-exists) [[ ! -e "$workspace/$path" ]] || assertion_failed=1 ;;
      contains) [[ -f "$workspace/$path" ]] && grep -Fq "$value" "$workspace/$path" || assertion_failed=1 ;;
      glob-count|glob-contains)
        local -a matches=()
        shopt -s nullglob
        matches=("$workspace"/$path)
        shopt -u nullglob
        if [[ "$operation" == "glob-count" ]]; then
          [[ "${#matches[@]}" -eq "$value" ]] || assertion_failed=1
        else
          local found=false match
          if ((${#matches[@]} > 0)); then
            for match in "${matches[@]}"; do
              if grep -Fq "$value" "$match"; then found=true; break; fi
            done
          fi
          [[ "$found" == true ]] || assertion_failed=1
        fi
        ;;
      *) echo "unknown filesystem assertion '$operation' in $scenario" >&2; return 2 ;;
    esac
    if ((assertion_failed > 0)); then
      failed=1
      echo "failed filesystem assertion for $scenario: $operation $path $value" >&2
    fi
  done <"$assertions"
  return "$failed"
}

run_scenarios() {
  local scenarios="$1" go_bin="$2" rust_bin="$3" expected="$4" work="$5"
  local failures=0 total=0 unexpected=0 stale=0
  local actual_gaps="$work/actual-gaps.txt"
  mkdir -p "$work"
  : >"$actual_gaps"

  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ -z "$line" || "$line" == \#* ]] && continue
    local -a fields
    IFS=$'\t' read -r -a fields <<<"$line"
    if ((${#fields[@]} < 5)); then
      echo "invalid scenario row: $line" >&2
      return 2
    fi

    local name="${fields[0]}" fixture="${fields[1]}" policy="${fields[2]}" filesystem="${fields[3]}"
    local fixture_path="$repo_root/tests/fixtures/$fixture"
    local scenario_dir="$work/$name"
    local go_workspace="$scenario_dir/go-workspace" rust_workspace="$scenario_dir/rust-workspace"
    local go_board="$go_workspace/board" rust_board="$rust_workspace/board"
    local -a args=("${fields[@]:4}")
    ((total += 1))

    mkdir -p "$go_workspace" "$rust_workspace"
    if [[ "$fixture" != "-" ]]; then
      [[ -d "$fixture_path" ]] || { echo "missing fixture: $fixture_path" >&2; return 2; }
      mkdir -p "$go_board" "$rust_board"
      cp -R "$fixture_path/." "$go_board/"
      cp -R "$fixture_path/." "$rust_board/"
    fi

    local failed=false
    case "$policy" in
      parity)
        run_command "$go_bin" "$go_board" "$go_workspace" "$scenario_dir/go" "${args[@]}"
        run_command "$rust_bin" "$rust_board" "$rust_workspace" "$scenario_dir/rust" "${args[@]}"
        compare_file "$name stdout" "$scenario_dir/go/stdout" "$scenario_dir/rust/stdout" || failed=true
        compare_file "$name stderr" "$scenario_dir/go/stderr" "$scenario_dir/rust/stderr" || failed=true
        compare_file "$name status" "$scenario_dir/go/status" "$scenario_dir/rust/status" || failed=true
        compare_file "$name filesystem" "$scenario_dir/go/tree.after" "$scenario_dir/rust/tree.after" || failed=true
        ;;
      intent)
        run_command "$rust_bin" "$rust_board" "$rust_workspace" "$scenario_dir/rust" "${args[@]}"
        local expected_dir="$expected/$name"
        for artifact in stdout stderr status; do
          [[ -f "$expected_dir/$artifact" ]] || { echo "missing expectation: $expected_dir/$artifact" >&2; return 2; }
          compare_file "$name $artifact" "$expected_dir/$artifact" "$scenario_dir/rust/$artifact" || failed=true
        done
        if [[ -f "$expected_dir/tree" ]]; then
          compare_file "$name filesystem" "$expected_dir/tree" "$scenario_dir/rust/tree.after" || failed=true
        fi
        if [[ -f "$expected_dir/filesystem.tsv" ]]; then
          check_filesystem_assertions "$name" "$expected_dir/filesystem.tsv" "$rust_workspace" || failed=true
        fi
        ;;
      *) echo "unknown policy '$policy' in $name" >&2; return 2 ;;
    esac

    if [[ "$filesystem" == "readonly" ]]; then
      compare_file "$name Rust read-only tree" "$scenario_dir/rust/tree.before" "$scenario_dir/rust/tree.after" || failed=true
      if [[ "$policy" == "parity" ]]; then
        compare_file "$name Go read-only tree" "$scenario_dir/go/tree.before" "$scenario_dir/go/tree.after" || failed=true
      fi
    elif [[ "$filesystem" != "mutation" ]]; then
      echo "unknown filesystem policy '$filesystem' in $name" >&2
      return 2
    fi

    if [[ "$failed" == true ]]; then
      ((failures += 1))
      echo "$name" >>"$actual_gaps"
      if [[ "$allow_gaps" == true ]] && ! grep -Fxq "$name" "$expected_gaps_file"; then
        echo "UNEXPECTED GAP $name" >&2
        unexpected=1
      else
        echo "GAP $name" >&2
      fi
    else
      echo "PASS $name"
    fi
  done <"$scenarios"

  if [[ "$allow_gaps" == true ]]; then
    [[ -f "$expected_gaps_file" ]] || { echo "missing expected-gap allowlist: $expected_gaps_file" >&2; return 2; }
    while IFS= read -r name || [[ -n "$name" ]]; do
      [[ -z "$name" || "$name" == \#* ]] && continue
      if ! grep -Fxq "$name" "$actual_gaps"; then
        echo "STALE EXPECTED GAP $name" >&2
        stale=1
      fi
    done <"$expected_gaps_file"
    if ((unexpected > 0 || stale > 0)); then
      return 1
    fi
    echo "expected gap set matched ($failures/$total)"
    return 0
  fi

  if ((failures > 0)); then
    echo "$failures/$total contract scenario(s) differ" >&2
    return 1
  fi
  echo "$total contract scenario(s) passed"
}

self_test_harness() {
  local root="$1"
  mkdir -p "$root/tests/fixtures/self/board" "$root/expected/intent"
  printf 'seed\n' >"$root/tests/fixtures/self/board/seed.txt"
  cat >"$root/fake" <<'SH'
#!/usr/bin/env bash
set -u
board=""
command=""
while (($#)); do
  case "$1" in
    --path) board="$2"; shift 2 ;;
    *) command="$1"; shift ;;
  esac
done
case "$command" in
  mutate)
    printf 'changed\n' >"$board/generated.txt"
    printf 'board/\n' >"$board/../.gitignore"
    echo same
    echo mutation-note >&2
    exit 7
    ;;
  intent)
    echo "$board/result"
    echo intent-note >&2
    exit 3
    ;;
  *) echo "unexpected command" >&2; exit 2 ;;
esac
SH
  chmod +x "$root/fake"
  cat >"$root/scenarios.tsv" <<'EOF'
parity	self/board	parity	mutation	--path	{board}	mutate
intent	self/board	intent	readonly	--path	{board}	intent
EOF
  printf '{BOARD}/result\n' >"$root/expected/intent/stdout"
  printf 'intent-note\n' >"$root/expected/intent/stderr"
  printf '3\n' >"$root/expected/intent/status"
  printf 'file\tboard/seed.txt\t\n' >"$root/expected/intent/filesystem.tsv"

  local saved_root="$repo_root"
  repo_root="$root"
  run_scenarios "$root/scenarios.tsv" "$root/fake" "$root/fake" "$root/expected" "$root/work"
  local status=$?
  repo_root="$saved_root"
  [[ "$status" -eq 0 ]] || return "$status"

  grep -q $'^F\t./board/generated.txt\t' "$root/work/parity/go/tree.after" || return 1
  grep -q $'^F\t./.gitignore\t' "$root/work/parity/go/tree.after" || return 1
  grep -q '^mutation-note$' "$root/work/parity/go/stderr" || return 1
  grep -q '^7$' "$root/work/parity/go/status" || return 1
  grep -q '^intent-note$' "$root/work/intent/rust/stderr" || return 1
  grep -q '^3$' "$root/work/intent/rust/status" || return 1
}

tmp="$(mktemp -d "${TMPDIR:-/tmp}/tickcats-contracts.XXXXXX")"
trap 'rm -rf "$tmp"' EXIT

if [[ "$self_test" == true ]]; then
  self_test_harness "$tmp/self-test"
  exit $?
fi

if [[ -n "${GO_TICKCATS_BIN:-}" ]]; then
  go_bin="$GO_TICKCATS_BIN"
else
  go_bin="$tmp/go-tickcats"
  (cd "$repo_root" && go build -o "$go_bin" ./cmd/tickcats) || exit 1
fi

if [[ -n "${RUST_TICKCATS_BIN:-}" ]]; then
  rust_bin="$RUST_TICKCATS_BIN"
else
  export CARGO_TARGET_DIR="$tmp/cargo-target"
  (cd "$repo_root" && cargo build --quiet) || exit 1
  rust_bin="$CARGO_TARGET_DIR/debug/tickcats"
fi

run_scenarios "$scenario_file" "$go_bin" "$rust_bin" "$expected_root" "$tmp/run"
