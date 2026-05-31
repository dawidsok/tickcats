# Slice 17 — Config Page and Editor Selection

## Goal

Let users configure which editor TickCats opens instead of relying only on `$EDITOR` or fallback `vi`.

## Scope

- Add local TickCats config under `.tickcats/config.toml` or `.tickcats/config.json`.
- Add TUI config page opened by `c`.
- Config page shows current editor command.
- Support selecting common editor presets:
  - `nvim`
  - `vim`
  - `vi`
  - `code --wait`
  - custom command
- Editor resolution order:
  1. TickCats config editor command
  2. `$EDITOR`
  3. fallback `vi`
- Support editor commands with args, e.g. `nvim`, `nvim -f`, `code --wait`.
- Persist config locally inside `.tickcats/`.
- Keep `.tickcats/` git-ignored/private.

## Out of Scope

- No global user config yet.
- No theme configuration.
- No keybinding remap UI.
- No validation beyond command execution failure message.

## Tests

- Editor command uses config before `$EDITOR`.
- `$EDITOR` is used when config editor is empty.
- fallback `vi` is used when both are empty.
- editor command with args is parsed correctly.
- config load/save round trip works.

## Acceptance Checks

- User can set editor to LazyVim via `nvim` or custom command.
- Pressing `e` opens configured editor, not baseline `vi`.
- `go test ./...` passes.
