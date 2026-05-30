# Draft Concept: TickCats

## 1. Seed Idea

TickCats — a simple TUI scrum app for creating and managing tickets in local GitHub repositories. It should be keyboard-first and usable without internet access. GitHub Issues integration is intentionally a future feature, not part of the initial version.

The codename/name comes from `tickets + cats`: TickCats.

The project is intended to be open source, built first for the author's own workflow and later useful to similar solo developers.

## 2. Primary User and Pain

- Primary user: Keyboard-first solo developer managing private or side-project backlogs locally.
- Pain moment: During planning, the developer wants to quickly decide the next item to pick up. During coding, they also want a quick way to add/refine items without leaving terminal flow.
- Current workaround: Existing backlog tools, notes, or GitHub-oriented workflows require more context switching and/or assume internet access.
- Cost today: Friction, slower planning, less confidence about what is actually ready to work on, and weaker overview of planned work in private projects.

## 3. Concept Promise

When opened inside a local repository, the app helps a solo developer quickly pick the next actionable backlog item by surfacing the highest-priority ticket that is ready to start.

## 4. Wedge / First Use Case

The narrow first use case is repo-local planning: open the TUI inside one repository, review the local scrum board, and choose the next ready ticket to work on.

The core rule is:

> Next item = the highest-priority unblocked ticket in `.tickcats/ready/` that has a clear title and acceptance notes.

Status is folder-based: the filesystem is the board. Moving a ticket between workflow states means moving its markdown file between status folders. The v1 board folders are `.tickcats/backlog/`, `.tickcats/ready/`, `.tickcats/doing/`, and `.tickcats/done/`. This makes the app more than a TODO list: it supports a lightweight scrum workflow and nudges tickets toward enough refinement to be actionable.

Priority uses `P0` / `P1` / `P2` / `P3`, where `P0` is highest. A ticket with non-empty `blocked_by` is excluded from pick-next. If multiple ready unblocked tickets share the same priority, TickCats prefers the oldest ticket based on local filesystem metadata. If needed, it can show tied candidates for manual selection.

v1 supports only Private mode: `.tickcats/` is intended to be git-ignored. Repo mode, where `.tickcats/` is committed and portable with the repository, is a future feature. Even in Private mode, tickets include explicit `created` and `updated` frontmatter fields because filesystem metadata is too inconsistent for product behavior.

## 5. Smallest Valuable Version

1. User opens the TUI inside a local repository.
2. App loads that repository's local markdown ticket backlog from `.tickcats/`.
3. User sees a scrum-style board/workflow.
4. User uses a command palette and simple detail form to create or refine tickets.
5. Ticket detail includes type, title, acceptance notes, and priority. Workflow state is derived from the ticket's folder, not duplicated as a field inside the file.
6. Ticket type is selected from local templates: Feature, Ticket, or Bug.
7. App identifies tickets that are ready to start based on hybrid readiness: the ticket is in `.tickcats/ready/` and has clear title + acceptance notes. The same readiness rule applies to Feature, Ticket, and Bug templates.
8. Ticket files use YAML frontmatter for metadata, including `created` and `updated`, and markdown sections for human-readable details.
9. Tickets may include an optional `blocked_by` field/section. If `blocked_by` is non-empty, the ticket is excluded from pick-next, but v1 does not implement complex dependency logic.
10. App surfaces the highest-priority ready ticket as the next item to pick up.
11. User moves that ticket into the active/in-progress state.

## Ticket Template Schema

Ticket files use YAML frontmatter plus markdown sections. Required metadata fields are `type`, `title`, `priority`, `created`, and `updated`. `blocked_by` is optional but included in generated templates as a self-documenting empty field with an inline `# optional` comment.

Readiness is the same for all ticket types:

- file lives in `.tickcats/ready/`
- `title` is non-empty
- `blocked_by` is empty
- `## Acceptance Criteria` is non-empty

### Feature

```md
---
type: feature
title:
priority: P2
created:
updated:
blocked_by: # optional
---

## Problem / Opportunity

## Proposed Behavior

## Acceptance Criteria
-
```

### Ticket

```md
---
type: ticket
title:
priority: P2
created:
updated:
blocked_by: # optional
---

## Context

## Work Required

## Acceptance Criteria
-
```

### Bug

```md
---
type: bug
title:
priority: P2
created:
updated:
blocked_by: # optional
---

## Observed Behavior

## Expected Behavior

## Reproduction

## Acceptance Criteria
-
```

`## Reproduction` is included for bugs but does not affect v1 readiness.

## TUI Design Notes

- Default screen: board + pick-next. Opening TickCats in a repo should immediately show the recommended next ticket while preserving board context.
- Layout: kanban columns for `.tickcats/backlog/`, `.tickcats/ready/`, `.tickcats/doing/`, and `.tickcats/done/`.
- Navigation: vim-like keys plus command palette.
  - `h` / `l`: move between columns
  - `j` / `k`: move within a column
  - `enter`: open selected ticket details
  - `n`: new ticket
  - `e`: edit/refine selected ticket metadata
  - `m`: move ticket/state
  - `p`: pick next
  - `/`: basic search/filter across visible ticket title/content
  - `ctrl-k` or `:`: command palette
  - `?`: help
  - `q`: quit
- Required command palette actions for the first implementation:
  - `New Feature`
  - `New Ticket`
  - `New Bug`
  - `Move to Backlog`
  - `Move to Ready`
  - `Move to Doing`
  - `Move to Done`
  - `Edit Metadata`
  - `Open in Editor`
  - `Pick Next`
- Ticket detail view: full-screen detail for v1, replacing the board with focused ticket content and actions.
- Long ticket content must be scrollable in detail view using `j` / `k`.
- Detail view actions are direct hotkeys, not focusable buttons. `j` / `k` are reserved for scrolling ticket content in detail view.

## 6. What Is Explicitly Out of Scope

- GitHub Issues sync is out of scope for v1; it is a next/future feature. Any issue templates in v1 are local markdown ticket templates, not GitHub Issue templates.
- Repo mode is out of scope for v1: committing/versioning `.tickcats/` in Git is a future feature. v1 supports only Private mode, where `.tickcats/` is intended to be git-ignored.
- Team collaboration, shared server, realtime sync, and multi-user workflows are out of scope.
- Advanced scrum metrics such as velocity, burndown, and epics are out of scope.
- Complex dependency/blocker logic is out of scope. A simple optional `blocked_by` field/section is allowed and excludes a ticket from pick-next when non-empty, but v1 does not need dependency graphs or automatic unblocking.
- Cross-project dashboard/overview is out of scope for v1.
- Built-in markdown editor is out of scope. Tickets should be markdown files so users can edit them manually outside the app.
- Authentication/accounts are not needed.
- AI ticket generation or refinement is a future feature, not part of v1.

## 7. Key Assumptions

| Assumption | Status | How to test |
| --- | --- | --- |
| The user will actually open and use a separate TUI during coding/planning. | High-risk unknown | Build a tiny prototype and dogfood it during real planning sessions for private projects. |
| Tickets will contain enough title and acceptance-note detail for readiness detection to be useful. | High-risk unknown | Track whether the command palette + detail form makes refinement fast enough that tickets stay actionable. |
| Highest-priority ready ticket is a meaningful improvement over a manually ordered TODO list. | Plausible but unproven | Compare planning sessions with the TUI against current notes/TODO workflow. |
| Feature, Ticket, and Bug templates are enough structure for v1 without making ticket creation feel heavy. | Plausible but unproven | Dogfood each template during real work and remove fields that are not consistently useful. |
| Repo-local planning is useful enough before cross-project overview exists. | Plausible but unproven | Limit v1 to one repo and observe whether the missing cross-project view blocks daily use. |
| Other solo developers want a local-first OSS backlog TUI enough to install it. | Plausible but unproven | Publish early demo/readme and invite feedback from keyboard-first solo developers. |

## 8. Socratic Challenges

- If this only stores tickets, it is too close to a TODO list. Resolution: the concept centers on the workflow/readiness rule: recommend the highest-priority ticket that is actually ready to start.
- Manual priority alone may be too weak. Resolution: v1 combines priority with readiness based on clear title and acceptance notes.
- A full board, cross-project overview, GitHub sync, markdown editor, and AI refinement would bloat the MVP. Resolution: v1 stays repo-local and proves the planning loop first.
- Ticket readiness depends on capture/refinement behavior. Resolution: the UX should prioritize command palette + simple detail form, not a large editor experience.

## 9. PRD Readiness

Verdict: Ready for PRD

Before PRD, resolve:

- No blocking product-scope questions remain. The PRD should preserve the decisions above and avoid expanding v1 scope.

Suggested next step:

- Generate a PRD for the first real TickCats app foundation: local `.tickcats/` storage, markdown ticket templates, board + pick-next TUI, command palette, and detail view.
