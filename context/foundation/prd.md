---
project: TickCats
version: 1
status: draft
created: 2026-05-30
context_type: greenfield
product_type: cli
target_scale:
  users: small
  qps: n/a
  data_volume: small
timeline_budget:
  mvp_weeks: "# TODO: mvp_weeks — see Open Questions"
  hard_deadline: null
  after_hours_only: "# TODO: after_hours_only — see Open Questions"
---

## Vision & Problem Statement

Keyboard-first solo developers managing private or side-project backlogs feel friction during planning and coding: they want to quickly decide the next item to pick up, add/refine work without leaving terminal flow, and avoid being bound to internet access.

The insight is that a local scrum backlog becomes more useful than a TODO list when it can answer a planning decision: the next item is the highest-priority unblocked ticket that is deliberately ready and has enough acceptance detail to start.

## User & Persona

Primary persona: a keyboard-first solo developer managing private or side-project repositories locally.

They reach for TickCats during planning to choose the next item to pick up, and during coding when they need to quickly add or refine future work without switching to a browser or online project-management tool.

## Success Criteria

### Primary

- The user can open TickCats inside one repository and identify the next actionable ticket in under two minutes.
- TickCats recommends the highest-priority unblocked ticket in `.tickcats/ready/` that has a non-empty title and non-empty acceptance criteria.

### Secondary

- The user can create Feature, Ticket, and Bug items from local templates.
- The user can move tickets through the v1 workflow folders: backlog, ready, doing, done.
- The user can open a ticket in their external editor for full markdown editing.

### Guardrails

- TickCats remains usable without internet access.
- TickCats does not require authentication or accounts.
- v1 keeps `.tickcats/` private/local by default and does not require committing ticket data to Git.
- v1 does not expand into GitHub Issues sync, collaboration, metrics, or cross-project overview.

## User Stories

### US-01: Pick the next ready ticket

- **Given** a repository with local TickCats tickets across backlog, ready, doing, and done folders
- **When** the user opens TickCats and invokes pick-next
- **Then** TickCats surfaces the highest-priority unblocked ticket in `.tickcats/ready/` that has a title and non-empty acceptance criteria

#### Acceptance Criteria

- Tickets outside `.tickcats/ready/` are not eligible for pick-next.
- Tickets with non-empty `blocked_by` are not eligible for pick-next.
- `P0` outranks `P1`, `P1` outranks `P2`, and `P2` outranks `P3`.
- When multiple eligible tickets share the same priority, the oldest ticket by `created` metadata is preferred.
- If manual choice is needed, TickCats can show tied candidates for selection.

### US-02: Manage the repo-local board

- **Given** a repository with a `.tickcats/` backlog
- **When** the user opens TickCats
- **Then** they see a kanban-style board with Backlog, Ready, Doing, and Done columns plus the recommended next ticket

#### Acceptance Criteria

- The board maps to `.tickcats/backlog/`, `.tickcats/ready/`, `.tickcats/doing/`, and `.tickcats/done/`.
- Moving a ticket between workflow states moves its markdown file between folders.
- The board supports vim-like navigation with `h`/`l` between columns and `j`/`k` within a column.

### US-03: Create tickets from templates

- **Given** the user is in the TickCats TUI
- **When** they create a new Feature, Ticket, or Bug
- **Then** TickCats creates a markdown ticket from the corresponding local template

#### Acceptance Criteria

- New ticket commands exist for Feature, Ticket, and Bug.
- Generated tickets include required metadata: `type`, `title`, `priority`, `created`, and `updated`.
- Generated tickets include `blocked_by: # optional` as a self-documenting optional field.
- Generated tickets include an Acceptance Criteria section.

### US-04: Inspect and refine a ticket

- **Given** a selected ticket on the board
- **When** the user opens the ticket detail view
- **Then** TickCats shows a full-screen ticket detail view with direct hotkeys for actions

#### Acceptance Criteria

- Long ticket content scrolls with `j`/`k`.
- Detail actions are direct hotkeys, not focusable buttons.
- The user can edit metadata in the TUI.
- The user can open the ticket in their external editor for full markdown edits.

## Functional Requirements

### Local backlog storage

- FR-001: User can initialize or use a repo-local `.tickcats/` folder. Priority: must-have
- FR-002: User can store tickets as markdown files under `.tickcats/backlog/`, `.tickcats/ready/`, `.tickcats/doing/`, and `.tickcats/done/`. Priority: must-have
- FR-003: User can keep `.tickcats/` private/local for v1. Priority: must-have

### Ticket templates

- FR-004: User can create a Feature ticket from a local template. Priority: must-have
- FR-005: User can create a Ticket ticket from a local template. Priority: must-have
- FR-006: User can create a Bug ticket from a local template. Priority: must-have
- FR-007: User can include required ticket metadata: `type`, `title`, `priority`, `created`, and `updated`. Priority: must-have
- FR-008: User can include optional `blocked_by` metadata. Priority: must-have

### Board and workflow

- FR-009: User can view a kanban board with Backlog, Ready, Doing, and Done columns. Priority: must-have
- FR-010: User can move a ticket to Backlog, Ready, Doing, or Done. Priority: must-have
- FR-011: User can navigate the board with vim-like keys. Priority: must-have

### Pick-next

- FR-012: User can ask TickCats to pick the next ticket. Priority: must-have
- FR-013: User can see the recommended next ticket on the default board screen. Priority: must-have
- FR-014: User can exclude blocked tickets from pick-next by setting non-empty `blocked_by`. Priority: must-have

### Search and detail

- FR-015: User can basic-search/filter visible tickets by title/content. Priority: must-have
- FR-016: User can open a full-screen ticket detail view. Priority: must-have
- FR-017: User can scroll long ticket content in detail view with `j`/`k`. Priority: must-have
- FR-018: User can edit selected ticket metadata in the TUI. Priority: must-have
- FR-019: User can open a ticket in their external editor. Priority: must-have

### Command palette

- FR-020: User can open a command palette. Priority: must-have
- FR-021: User can run New Feature, New Ticket, New Bug, Move to Backlog, Move to Ready, Move to Doing, Move to Done, Edit Metadata, Open in Editor, and Pick Next from the command palette. Priority: must-have

## Non-Functional Requirements

- The product remains usable without internet access for all v1 workflows.
- The default planning screen allows a user with an existing `.tickcats/` backlog to identify the recommended next ticket in under two minutes.
- The product supports keyboard-first operation for core v1 workflows without requiring pointer interaction.
- Ticket data remains local/private in v1 unless the user explicitly chooses to manage it outside TickCats.
- The product keeps v1 ticket creation lightweight: generated templates include only the required metadata, optional `blocked_by`, and type-specific markdown sections.

## Business Logic

TickCats recommends the next item by selecting the highest-priority unblocked ticket in `.tickcats/ready/` that has a non-empty title and non-empty Acceptance Criteria.

The rule consumes user-maintained ticket metadata, folder-derived workflow state, blocker state, and acceptance detail. Its output is the single best next ticket to pick up, shown on the default board screen and available through both a hotkey and command palette action.

Priority order is `P0`, `P1`, `P2`, then `P3`. If multiple eligible tickets share the same priority, TickCats prefers the oldest ticket by `created` metadata; if manual choice is needed, it can show tied candidates for selection.

## Access Control

Single user; no auth; data lives on-device only.

## Non-Goals

- No GitHub Issues sync in v1; local templates are not GitHub Issue templates.
- No Repo mode in v1; committing/versioning `.tickcats/` in Git is a future feature.
- No team collaboration, shared server, realtime sync, or multi-user workflow in v1.
- No advanced scrum metrics such as velocity, burndown, or epics in v1.
- No complex dependency graph or automatic unblocking in v1.
- No cross-project dashboard/overview in v1.
- No built-in markdown editor in v1; users can edit markdown tickets manually outside TickCats.
- No authentication/accounts in v1.
- No AI ticket generation or refinement in v1.

## Open Questions

1. **What is the MVP timeline budget?** — Owner: user. Block: no for PRD draft; required before downstream planning.
2. **Is v1 after-hours-only work?** — Owner: user. Block: no for PRD draft; required before downstream planning.
