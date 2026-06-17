# Changelog

## [0.1.0] — 2026-06-01

First public release.

### CLI

- `tickcats init` — create `.tickcats/` folder structure and add to `.gitignore`
- `tickcats new feat|task|bug <title>` — create a ticket in backlog with YAML frontmatter
- `tickcats list` — list tickets grouped by workflow state
- `tickcats move <ticket> <from> <to>` — move ticket between state folders
- `tickcats pick-next` — recommend next ready ticket by priority, then age; detects ties
- `tickcats tui` — open the terminal kanban board
- `--path <dir>` flag on all commands to target a non-default board directory

### TUI

- Four-column board: Backlog → Ready → Doing → Done
- Keyboard navigation: `h/l` columns, `j/k` tickets, `d/u` half-page scroll
- Pick-next banner above the board showing the recommended next ticket
- Detail view with scrollable body and metadata panel
- Ticket creation form: kind (feat/task/bug), title, priority, to-refine checkbox
- Post-create editor prompt with "don't ask again" option
- Move mode (`m`): `h/l` move one column, `H/L` jump to first/last column
- Multi-select (`v`): select multiple tickets, then move them together in move mode
- Manual reorder (`j/k` in move mode with manual sort active)
- Board sorting: priority, title, date, manual — cycles with `s`, persisted to `sort.json`
- In-column reorder for manual sort mode
- Delete with confirmation (`x` + `y`)
- External editor integration (`e`) — respects `$EDITOR`, configurable via config page
- Filesystem auto-refresh via `fsnotify` — board updates when files change on disk
- Config page (`c`): editor preset selector with custom input, six color themes
- Horizontal column scrolling — adapts to terminal width; narrow terminals show fewer columns with scroll indicators
- Quit confirmation (`q` → `y/n`)
- Color themes: mono, gradient, ocean, fire, forest, dim-sum

### Storage

- Tickets stored as plain markdown files with YAML frontmatter
- Workflow state derived from folder location, not frontmatter
- Soft-delete moves tickets to `.tickcats/trash/`
- Config persisted to `.tickcats/config.json`
- Manual sort order persisted to `.tickcats/sort.json`
