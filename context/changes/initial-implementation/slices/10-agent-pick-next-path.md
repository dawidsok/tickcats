# Slice 10 — Agent Pick-Next Path Output

## Goal

Expose a machine-friendly CLI output so agents can ask TickCats which ticket file to work on next.

## Scope

- Add a CLI flag to `tickcats pick-next` that returns only the selected ticket path.
- Proposed flag: `tickcats pick-next --path`.
- Output should be stable and script-friendly:
  - selected ticket: print path only, e.g. `.tickcats/ready/20260530-1320-manual-tui-test.md`
  - no eligible ticket: exit non-zero with clear stderr message
  - tied candidates: exit non-zero and print candidate paths to stderr, or choose deterministic first if explicitly requested later
- Keep human output unchanged when `--path` is absent.

## Out of Scope

- No JSON output yet.
- No agent protocol integration.
- No automatic ticket claiming/moving.
- No tie-breaking prompt in non-interactive mode.

## Tests

- `tickcats pick-next --path` prints only selected path when exactly one best ticket exists.
- No eligible ticket exits non-zero.
- Tie exits non-zero and surfaces candidate paths.
- Human `tickcats pick-next` output remains unchanged.

## Acceptance Checks

- Agents can run `tickcats pick-next --path` and capture one path from stdout.
- `go test ./...` passes.
