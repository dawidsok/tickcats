# Slice 02 — Init Private Board

## Goal

Create the local private `.tickcats/` board safely and idempotently.

## Scope

- Add an init command or init function callable by CLI.
- Create folders:
  - `.tickcats/backlog/`
  - `.tickcats/ready/`
  - `.tickcats/doing/`
  - `.tickcats/done/`
- Create `.gitignore` if missing.
- Add `.tickcats/` to `.gitignore` if not already present.
- Keep existing `.gitignore` lines unchanged.

## Out of Scope

- No Repo mode.
- No committed `.tickcats/` support.
- No config file.

## Tests

- Init creates all folders in a temp directory.
- Init is idempotent.
- Init preserves existing ticket files.
- Init creates `.gitignore` if absent.
- Init does not duplicate `.tickcats/` if already ignored.

## Acceptance Checks

- Running init twice succeeds.
- `.tickcats/` exists with all four state folders.
- `.gitignore` contains `.tickcats/` exactly once.
- `go test ./...` passes.
