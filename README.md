# TickCats

A keyboard-first, local kanban board for solo developers. Tickets are plain markdown files stored in `.tickcats/` inside your repo — no accounts, no sync, no servers.

```
┌─ Next: [P1] Add dark mode support ───────────────────────────────────────┐
│                                                                           │
├─ BACKLOG ──────┬─ READY ────────┬─ DOING ────────┬─ DONE ─────────────── │
│ > [P1] Auth    │   [P0] Login   │   [P1] Dark    │   [P2] Init flow      │
│   [P2] Tests   │   [P2] Signup  │                │   [P3] Readme         │
└────────────────┴────────────────┴────────────────┴───────────────────────┘
BOARD MODE: h/l col  j/k/d/u ticket  v select  m move  s sort  n new  q quit
```

## Installation

### Homebrew (macOS and Linux)

```sh
brew tap dawidsok/tap
brew install tickcats
```

or

```sh
brew install dawidsok/tap/tickcats

```

### Direct download

Download the `tickcats_<version>_<os>_<arch>` archive for your platform from the [GitHub Releases](https://github.com/dawidsok/tickcats/releases) page, extract, and move the `tickcats` binary to a directory on your `$PATH`.

### go install

```sh
go install github.com/dawidsok/tickcats/cmd/tickcats@latest
```

Requires Go installed. The binary lands in `$GOPATH/bin/tickcats`.

## Quick start

```sh
cd your-project

tickcats init          # create .tickcats/ and add it to .gitignore
tickcats new feat "Add dark mode support"
tickcats new task "Write tests" --ac "All handlers covered"
tickcats                # open the board (no command defaults to tui)
```

## Commands

| Command | Description |
|---|---|
| `tickcats init` | Create board folders and update `.gitignore` |
| `tickcats new feat\|task\|bug <title>` | Create a ticket in backlog |
| `tickcats list` | List tickets grouped by configured column |
| `tickcats move <ticket> <from> <to>` | Move a ticket between columns; accepts folder IDs (`code-review`) or display names (`Code Review`) |
| `tickcats pick-next` | Print the next recommended ready ticket |
| `tickcats ids migrate` | Add IDs to existing tickets and rename migrated files |
| `tickcats` | Open the terminal board (default when no command given) |
| `tickcats tui` | Open the terminal board (explicit) |

All commands accept `--path <dir>` to target a board other than `.tickcats`.

## Shell completion

Homebrew installs shell completions automatically. If you installed with `go install`, copy or source the scripts from `completions/` yourself:

```sh
# bash: source directly or copy into your bash-completion directory
source completions/tickcats.bash

# zsh: copy into a directory listed in $fpath, then restart your shell
mkdir -p ~/.zsh/completions
cp completions/_tickcats.zsh ~/.zsh/completions/_tickcats

# fish
mkdir -p ~/.config/fish/completions
cp completions/tickcats.fish ~/.config/fish/completions/tickcats.fish
```

The completion scripts call hidden helpers (`tickcats __complete tickets` and `tickcats __complete columns`) so ticket and column candidates reflect your local `.tickcats/` board.

## TUI keyboard reference

### Board

| Key | Action |
|---|---|
| `h` / `l` | Move between columns (`3l` moves three columns) |
| `j` / `k` | Move between tickets (`10j` moves ten tickets) |
| `d` / `u` | Half-page down / up |
| `v` | Toggle selection on focused ticket |
| `m` | Enter move mode |
| `p` | Progress focused ticket to the next column |
| `enter` / `o` | Open detail view |
| `e` | Open ticket in external editor |
| `n` | New ticket form |
| `x` | Delete (with confirmation) |
| `s` | Cycle sort: priority → title → date → manual |
| `r` | Reload board from disk |
| `c` | Open config (editor, theme, columns) |
| `q` | Quit |

### Move mode (`m`)

| Key | Action |
|---|---|
| `h` / `l` | Move focused / selected ticket one column |
| `H` / `L` | Move to first / last column |
| `j` / `k` | Reorder within column (manual sort only) |
| `esc` | Return to board |

Use `v` in board mode to build a multi-ticket selection before entering move mode.

### Detail view

| Key | Action |
|---|---|
| `j` / `k` | Scroll content |
| `d` / `u` | Half-page scroll |
| `e` | Open in external editor |
| `esc` | Return to board |

## Ticket format

Tickets are markdown files with YAML frontmatter:

```markdown
---
title: "Feat: Add dark mode support [to refine]"
id: TC-A7K9Q2
priority: P1
created: 2026-05-30T10:00:00Z
updated: 2026-05-31T14:22:00Z
deadline: 2026-06-15
---

## Context

Users have requested a dark mode option for the dashboard.

## Acceptance Criteria

- Dark mode can be toggled in settings
- Preference is persisted across sessions
```

State is derived from which folder the file lives in — not from frontmatter. `id` is a stable ticket reference used in new filenames and commit references. `deadline` is optional and, when present, uses `YYYY-MM-DD`; new tickets omit deadlines by default.

## Board layout

```
.tickcats/
  backlog/   ← new tickets land here
  ready/     ← refined, unblocked, ready to start
  doing/     ← active work
  done/      ← completed
  wont-do/   ← intentionally rejected / not pursuing
  .trash/    ← soft-deleted tickets
  config.json
  sort.json
```

The `.tickcats/` directory is gitignored by default so the board stays local.

## Custom columns

Columns are folders under `.tickcats/`. The folder name is the column ID used on disk and by the CLI. Display names and order are stored in `.tickcats/config.json`:

```json
{
  "columns": [
    { "id": "backlog", "name": "Backlog" },
    { "id": "ready", "name": "Ready" },
    { "id": "code-review", "name": "Code Review" },
    { "id": "done", "name": "Done" }
  ]
}
```

On load, TickCats reconciles config with folders on disk:

- a folder on disk but missing from config is appended as a column,
- a config column whose folder is missing is removed,
- hidden/system folders such as `.trash` are ignored,
- missing folders are not recreated just because config mentions them.

`tickcats move` accepts both folder IDs and display names. The pick-next rule remains tied to `.tickcats/ready/`.

## Configuration

Press `c` in the TUI to open the config page. Settings are saved to `.tickcats/config.json`.

| Setting | Description |
|---|---|
| Editor | External editor command (`nvim`, `vim`, `nano`, `code`, …) or `$EDITOR` |
| Theme | Color theme: mono, gradient, ocean, fire, forest |
| Columns | Add, rename, reorder, or delete board columns. Deleted column tickets move to the first column. |

## Philosophy

- **Local first** — board data never leaves your machine
- **Plain files** — tickets are markdown; read and edit them with any tool
- **Git-friendly** — `.tickcats/` is gitignored; no merge conflicts
- **No dependencies** — single static binary, no runtime required
