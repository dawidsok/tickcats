# Slice 01 — Project Skeleton and Core Types

## Goal

Establish the Go package layout and core domain types used by the storage, parser, and pick-next engine.

## Scope

- Create `cmd/tickcats/main.go` with a minimal runnable entrypoint.
- Create `internal/ticket` for domain types and parsing helpers.
- Create `internal/store` for `.tickcats/` filesystem constants/helpers.
- Define states: `backlog`, `ready`, `doing`, `done`.
- Define priorities: `P0`, `P1`, `P2`, `P3`.
- Define inferred kinds: `feature`, `task`, `bug`.
- Define special title labels: `blocked`, `to refine`.
- Parse title labels and conventional kind prefixes:
  - `Feat:` → feature
  - `Bug:` → bug
  - `Task:` → task
  - missing prefix → task

## Out of Scope

- No TUI.
- No markdown file parsing yet beyond title parsing.
- No ticket creation command yet.

## Tests

- Priority ordering.
- Valid/invalid state parsing.
- Title parsing with no prefix defaults to task.
- Title parsing with labels before prefix.
- `[blocked, to refine] Feat: example` parses labels + feature kind.

## Acceptance Checks

- `go test ./...` passes.
- `go run ./cmd/tickcats` runs without error and prints minimal help/version text.
