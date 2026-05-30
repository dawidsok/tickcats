# Initial Implementation

- **Status:** planned
- **Created:** 2026-05-30
- **Scope:** Build the first real TickCats app foundation from the PRD.
- **PRD:** @context/foundation/prd.md
- **Tech stack:** @context/foundation/tech-stack.md

## Goal

Implement the core local ticket engine before TUI complexity: initialize `.tickcats/`, parse markdown tickets, derive state from folders, move tickets, and compute pick-next.

## Non-Goals

- No GitHub Issues sync.
- No Repo mode / committed `.tickcats/` support.
- No team collaboration, metrics, cross-project dashboard, auth, or AI.
- No rich markdown editor.
