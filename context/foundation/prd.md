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
- TickCats recommends the highest-priority unblocked, refined ticket in `.tickcats/ready/` that has a non-empty title and non-empty acceptance criteria.

### Secondary

- The user can create tickets whose kind is inferred from a conventional title prefix: `Feat:`, `Task:`, or `Bug:`.
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
- **Then** TickCats surfaces the highest-priority ticket in `.tickcats/ready/` that has a title, non-empty acceptance criteria, and no `[blocked]` or `[to refine]` title label

#### Acceptance Criteria

- Tickets outside `.tickcats/ready/` are not eligible for pick-next.
- Tickets whose title contains `[blocked]` are not eligible for pick-next.
- Tickets whose title contains `[to refine]` are not eligible for pick-next.
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

### US-03: Create tickets with conventional title prefixes

- **Given** the user is in the TickCats TUI
- **When** they create a new feature, task, or bug ticket
- **Then** TickCats creates a markdown ticket whose kind is inferred from the title prefix

#### Acceptance Criteria

- New ticket commands exist for feature, task, and bug creation.
- Generated feature titles start with `Feat:`.
- Generated task titles start with `Task:`.
- Generated bug titles start with `Bug:`.
- A ticket without `Feat:`, `Task:`, or `Bug:` is treated as a task.
- Generated tickets include required metadata: `title`, `priority`, `created`, and `updated`.
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

### Ticket creation and title labels

- FR-004: User can create a feature ticket whose title starts with `Feat:`. Priority: must-have
- FR-005: User can create a task ticket whose title starts with `Task:`. Priority: must-have
- FR-006: User can create a bug ticket whose title starts with `Bug:`. Priority: must-have
- FR-007: User can omit a kind prefix and have TickCats treat the ticket as a task. Priority: must-have
- FR-008: User can include required ticket metadata: `title`, `priority`, `created`, and `updated`. Priority: must-have
- FR-009: User can add title labels before the kind prefix, such as `[blocked]`, `[to refine]`, or `[idea]`. Priority: must-have

### Board and workflow

- FR-010: User can view a kanban board with Backlog, Ready, Doing, and Done columns. Priority: must-have
- FR-011: User can move a ticket to Backlog, Ready, Doing, or Done. Priority: must-have
- FR-012: User can navigate the board with vim-like keys. Priority: must-have

### Pick-next

- FR-013: User can ask TickCats to pick the next ticket. Priority: must-have
- FR-014: User can see the recommended next ticket on the default board screen. Priority: must-have
- FR-015: User can exclude blocked tickets from pick-next by adding a `[blocked]` title label. Priority: must-have
- FR-016: User can exclude unrefined tickets from pick-next by adding a `[to refine]` title label. Priority: must-have

### Search and detail

- FR-017: User can basic-search/filter visible tickets by title/content. Priority: must-have
- FR-018: User can open a full-screen ticket detail view. Priority: must-have
- FR-019: User can scroll long ticket content in detail view with `j`/`k`. Priority: must-have
- FR-020: User can edit selected ticket metadata in the TUI. Priority: must-have
- FR-021: User can open a ticket in their external editor. Priority: must-have

### Command palette

- FR-022: User can open a command palette. Priority: must-have
- FR-023: User can run New Feature, New Task, New Bug, Move to Backlog, Move to Ready, Move to Doing, Move to Done, Edit Metadata, Open in Editor, and Pick Next from the command palette. Priority: must-have

## Non-Functional Requirements

- The product remains usable without internet access for all v1 workflows.
- The default planning screen allows a user with an existing `.tickcats/` backlog to identify the recommended next ticket in under two minutes.
- The product supports keyboard-first operation for core v1 workflows without requiring pointer interaction.
- Ticket data remains local/private in v1 unless the user explicitly chooses to manage it outside TickCats.
- The product keeps v1 ticket creation lightweight: generated tickets use title prefixes and labels instead of separate `type` or `blocked_by` metadata.

## Business Logic

TickCats recommends the next item by selecting the highest-priority ticket in `.tickcats/ready/` that has a non-empty title, non-empty Acceptance Criteria, and no `[blocked]` or `[to refine]` title label.

The rule consumes user-maintained ticket metadata, folder-derived workflow state, title labels, and acceptance detail. Its output is the single best next ticket to pick up, shown on the default board screen and available through both a hotkey and command palette action.

Priority order is `P0`, `P1`, `P2`, then `P3`. If multiple eligible tickets share the same priority, TickCats prefers the oldest ticket by `created` metadata; if manual choice is needed, it can show tied candidates for selection.

## Access Control

Single user; no auth; data lives on-device only.

## Non-Goals

- No GitHub Issues sync in v1; local templates are not GitHub Issue templates.
- No Repo mode in v1; committing/versioning `.tickcats/` in Git is a future feature.
- No team collaboration, shared server, realtime sync, or multi-user workflow in v1.
- No advanced scrum metrics such as velocity, burndown, or epics in v1.
- No complex dependency graph or automatic unblocking in v1; `[blocked]` is only a title label that excludes a ticket from pick-next.
- No cross-project dashboard/overview in v1.
- No built-in markdown editor in v1; users can edit markdown tickets manually outside TickCats.
- No authentication/accounts in v1.
- No AI ticket generation or refinement in v1.

## Open Questions

1. **What is the MVP timeline budget?** — Owner: user. Block: no for PRD draft; required before downstream planning.
2. **Is v1 after-hours-only work?** — Owner: user. Block: no for PRD draft; required before downstream planning.
