# Slice 05 — Pick-Next Rule

## Goal

Implement the core business rule from the PRD.

## Rule

Pick-next selects the highest-priority ticket that:

- lives in `.tickcats/ready/`
- has a non-empty title
- has non-empty `## Acceptance Criteria`
- does not have `[blocked]` in title labels
- does not have `[to refine]` in title labels

Ordering:

1. Priority: `P0` > `P1` > `P2` > `P3`
2. Oldest `created` timestamp wins within the same priority
3. Remaining ties return deterministic candidates for future manual selection

## Scope

- Implement readiness check.
- Implement priority comparison.
- Implement created timestamp tie-break.
- Return either one selected ticket or a deterministic tie set.
- Exclude `[blocked]` and `[to refine]` labels.

## Out of Scope

- No manual tie selection UI.
- No label toggling.
- No dependency graph.

## Tests

- Backlog tickets are never eligible.
- Ready tickets with empty Acceptance Criteria are not eligible.
- Ready tickets with `[blocked]` are not eligible.
- Ready tickets with `[to refine]` are not eligible.
- `P0` beats `P1`, `P2`, and `P3`.
- Older `created` wins within same priority.
- Missing kind prefix does not affect eligibility.

## Acceptance Checks

- Pick-next returns expected ticket across mixed folders and priorities.
- Pick-next behavior matches @context/foundation/prd.md Business Logic.
- `go test ./...` passes.
