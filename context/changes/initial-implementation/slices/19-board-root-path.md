# Slice 19 — Board Root Path Parameter

## Goal

Allow TickCats to operate on a non-default board directory for testing and alternate local boards.

## Scope

- Add a global `--path <dir>` option.
- Default remains `.tickcats` under current working directory.
- `--path .tickcats-test` makes all commands use `.tickcats-test/` instead of `.tickcats/`.
- Apply to:
  - `init`
  - `new`
  - `list`
  - `move`
  - `pick-next`
  - `tui`
- TUI must load, move, edit, and safe-delete tickets under the configured board path.
- `init --path .tickcats-test` creates the alternate board directory and updates `.gitignore` for that path.
- Existing command behavior without `--path` remains unchanged.

## Out of Scope

- No multiple boards visible at once.
- No project discovery above cwd.
- No global config for default path.
- No migration between board roots.

## Tests

- Commands default to `.tickcats`.
- `--path .tickcats-test init` creates `.tickcats-test/{backlog,ready,doing,done}`.
- `new/list/move/pick-next --path .tickcats-test` operate only on `.tickcats-test`.
- TUI model receives the configured board path.
- Safe delete uses `.tickcats-test/.trash` when configured.

## Acceptance Checks

- User can keep real tickets in `.tickcats/` and mocked TUI tickets in `.tickcats-test/`.
- Running `tickcats --path .tickcats-test tui` opens the test board.
- `go test ./...` passes.
