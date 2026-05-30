# Slice 03 — Ticket Creation and Markdown Parsing

## Goal

Support the simplified v1 markdown ticket schema.

## Scope

- Generate markdown tickets with frontmatter:
  - `title`
  - `priority`
  - `created`
  - `updated`
- Do not generate or parse `type` frontmatter.
- Do not generate or parse `blocked_by` frontmatter.
- Generate title prefixes through creation flow:
  - `Feat:`
  - `Task:`
  - `Bug:`
- Default missing prefix to task when parsing existing files.
- Generate a minimal body:
  - `## Context`
  - `## Acceptance Criteria`
- Parse enough markdown to detect whether Acceptance Criteria is non-empty.
- Parse title labels from a comma-separated bracket list such as `[blocked, to refine]` or `[idea, to refine]`.

## Out of Scope

- No label filtering UI.
- No TUI-assisted label toggling.
- No rich markdown editor.
- No kind-specific body templates unless the open decision is resolved before implementation.

## Tests

- Generated feature/task/bug tickets parse successfully.
- Missing kind prefix parses as task.
- `[blocked]` label is detected.
- `[to refine]` label is detected.
- Arbitrary labels are preserved/detected but have no special behavior.
- Empty Acceptance Criteria is detected as not ready.
- Malformed frontmatter returns a clear parse error.

## Acceptance Checks

- Ticket parser returns title, labels, inferred kind, priority, created, updated, and acceptance presence.
- `go test ./...` passes.
