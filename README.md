# TickCats

A keyboard-first, local kanban board for solo developers. Tickets are plain markdown files stored in `.tickcats/` inside your repo — no accounts, no sync, no servers.

```
┌─ BACKLOG ─────────────────┐  ┌─ READY ──────────────────┐
│ > ★ Task: Auth refactor   │  │   Feat: Dark mode         │
│   TC-A7K9Q2  P1           │  │   TC-B8L0R3  P2           │
│──────────────────────────  │  │──────────────────────────  │
│   Feat: Dashboard         │  │                           │
│   TC-C1M2N3  P2           │  │                           │
└───────────────────────────┘  └───────────────────────────┘
h/l cols  j/k rows  enter detail  n new  p/b move  i !  x del  q quit
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
| `tickcats init [--no-intro]` | Create board folders and update `.gitignore` |
| `tickcats new feat\|task\|bug <title> [--ac <text>]` | Create a ticket in backlog |
| `tickcats list` | List tickets grouped by column |
| `tickcats move <ticket> <from> <to>` | Move a ticket one adjacent column; accepts filename or `TC-XXXXXX` id |
| `tickcats pick-next [--print-path]` | Print the next recommended ready ticket |
| `tickcats ids migrate` | Add IDs to existing tickets and rename files |
| `tickcats` | Open the terminal board (default when no command given) |
| `tickcats tui` | Open the terminal board (explicit) |

All commands accept `--path <dir>` before the subcommand to target a board other than `.tickcats`.

## Shell completion

Homebrew installs shell completions automatically. For direct-download installs, copy the scripts from `completions/`:

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
| `h` / `l` | Move between columns |
| `j` / `k` | Move between tickets |
| `d` / `u` | Half-page down / up |
| `enter` / `o` | Open detail panel |
| `n` | New ticket form |
| `p` | Progress ticket → next column |
| `b` | Move ticket back ← previous column |
| `i` | Toggle important on focused ticket |
| `x` | Soft-delete (with confirmation) |
| `e` | Open ticket in `$EDITOR` |
| `r` | Reload board from disk |
| `M` | Toggle urgency/importance matrix prioritisation |
| `/` | Fuzzy search across all columns |
| `?` | Help overlay |
| `q` / `Ctrl-C` | Quit |

### Detail panel (`enter`)

| Key | Action |
|---|---|
| `j` / `k` / `d` / `u` | Scroll |
| `i` | Toggle important |
| `e` | Open in `$EDITOR` |
| `esc` | Return to board |

## Ticket format

Tickets are markdown files with YAML frontmatter:

```markdown
---
title: "Feat: Add dark mode support [to refine]"
id: TC-A7K9Q2
priority: P1
important: true
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

State is derived from which folder the file lives in — not from frontmatter. `id` is a stable ticket reference. `important` is optional (toggle with `i`). `deadline` is optional (`YYYY-MM-DD`).

## Board layout

```
.tickcats/
  backlog/   ← new tickets land here
  ready/     ← refined, unblocked, ready to start
  doing/     ← active work (displayed as WIP)
  done/      ← completed
  .trash/    ← soft-deleted tickets
  config.json
```

The `.tickcats/` directory is gitignored by default.

## Configuration

`.tickcats/config.json` stores preferences:

| Setting | Description |
|---|---|
| `disable_matrix_prioritisation` | Set `true` to disable the urgency/importance matrix. Default: matrix on. |

Press `M` in the TUI to toggle the matrix and persist the change.

## Agent skills

[`skills/`](skills/) ships Agent Skills for Claude Code, OpenAI Codex CLI, Pi, and other compatible harnesses. They drive a full idea-to-implementation workflow on a tickcats board — each ticket becomes a self-contained, human-readable implementation spec (requirements, architecture, phased plan, progress tracking):

| Skill | Job |
|---|---|
| `tc-workflow` | Inspects the board, shows pipeline position, recommends next skill |
| `tc-shape` → `tc-prd` | Idea → discovery notes → PRD |
| `tc-decompose` | PRD → ticket stubs |
| `tc-refine` | Backlog refinement: sharpen, split, or kill tickets |
| `tc-plan` | Refine one ticket into a full implementation plan |
| `tc-plan-review` / `tc-impl-review` | Review plan / implementation |
| `tc-implement` | Execute the ticket: `ready → doing → done` |
| `tickcats-from-roadmap` | Convert a `roadmap.md` into tickets |

Install interactively:

```bash
./scripts/install-skills.sh
```

## Philosophy

- **Local first** — board data never leaves your machine
- **Plain files** — tickets are markdown; read and edit them with any tool
- **Git-friendly** — `.tickcats/` is gitignored; no merge conflicts
- **No dependencies** — single static binary, no runtime required
