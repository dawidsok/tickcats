# Ticket Unique IDs Exploration

Date: 2026-06-01
Ticket: `.tickcats/done/20260531-0802-ticket-unique-ids-exploration.md`

## Goal

Add stable ticket identifiers so tickets can be referenced reliably from Git commits, CLI output, TUI detail views, and future ticket operations even when titles, filenames, or workflow folders change.

Status must continue to come from `.tickcats/` folders, not ticket frontmatter.

## Strategies compared

### 1. Filename-derived ID

Example: `20260531-0802-ticket-unique-ids-exploration` from `20260531-0802-ticket-unique-ids-exploration.md`.

Pros:
- No frontmatter migration required.
- Human-readable and already present.
- Works across workflow moves because only the parent folder changes.

Cons:
- Breaks when a file is renamed externally.
- Copy/paste behavior is ambiguous unless filename is regenerated.
- Long and noisy in commit messages.
- Tightly couples identity to storage naming, which makes future rename/title cleanup risky.

Verdict: not stable enough for the core goal.

### 2. Frontmatter UUID/ULID

Example:

```yaml
id: 01JZ0F8CQ3F7H9N0AGD6X7V1MD
```

Pros:
- Stable across moves, renames, and title edits.
- Very low collision risk.
- No central counter or registry required.
- Good fit for local markdown storage.

Cons:
- Too long for everyday CLI/TUI display and commit messages.
- Users are less likely to type or recognize it.
- Visual noise in metadata-heavy views.

Verdict: technically strong, but too unfriendly for user-facing references.

### 3. Frontmatter short generated ID

Example:

```yaml
id: TC-A7K9Q2
```

Pros:
- Stable across moves, renames, and title edits.
- Short enough for commits and CLI/TUI display.
- Does not require a central counter; can be generated randomly with collision checks against the local board.
- Keeps tickets as local markdown files under workflow folders.
- Can remain optional for existing tickets during migration.

Cons:
- Small collision risk, so generation and board loading must detect duplicates.
- External manual edits can remove or duplicate IDs.
- Slightly less sortable than sequence IDs.

Verdict: best balance for TickCats v1/v0.x.

### 4. Frontmatter sequence ID

Example:

```yaml
id: TC-42
```

Pros:
- Very readable.
- Nice in commits and conversation.

Cons:
- Requires a durable counter file, probably `.tickcats/config.json` or a new `.tickcats/ids.json`.
- Counter conflicts become awkward if users copy boards, restore backups, or manually create files.
- Needs stronger locking/atomicity than random IDs to avoid duplicated numbers.

Verdict: readable, but introduces more state than necessary for a local file-first board.

## Recommendation

Use an optional frontmatter field named `id` with a short generated value:

```yaml
id: TC-A7K9Q2
```

Recommended ID format:
- Prefix: `TC-`
- Body: 6 Crockford Base32 characters, uppercase, excluding confusing characters where practical.
- Generated with crypto/rand or equivalent strong randomness.
- Checked against all loaded board IDs before writing.

Recommended new-ticket filename format:

```text
tc-a7k9q2-implement-stable-ticket-ids.md
```

Rules:
- Use the generated ID in the filename instead of the current date/time prefix.
- Keep frontmatter `id` as the source of truth.
- Lowercase the filename ID prefix for filesystem/readability consistency (`TC-A7K9Q2` → `tc-a7k9q2`).
- Keep the title slug after the ID for human scanning.
- Do not rename existing ticket files during normal load or migration planning.

This provides roughly one billion combinations with a compact, recognizable reference. For TickCats' small local board scale, collision risk is low and can be handled by retrying generation and warning on duplicates found during load.

## Behavior by workflow

### Move between columns

Preserve `id` and filename unchanged. Workflow state still comes only from the folder.

### Copy/paste ticket

The pasted ticket must receive a new `id` and a new ID-based filename. Copying must not preserve the source ID, otherwise two distinct tickets would collide.

### Rename file externally

Preserve identity if frontmatter `id` remains present. Filename should normally include the ID for new TickCats-created tickets, but frontmatter remains the source of truth after external renames.

### Edit title externally

Preserve identity if frontmatter `id` remains present. Title and ID are independent.

### Delete/trash

Preserve `id` in the moved markdown file. Trash is history; if restored manually, the same ticket identity returns.

### External manual file creation

If an `id` is missing, the ticket still loads during migration period. If an `id` is duplicated or malformed, board loading should warn clearly without changing folder-derived status.

## Migration for existing tickets

Avoid silently rewriting user files during normal board load, but include an explicit migration function in this feature so existing local boards can opt in.

Recommended migration path:
1. New tickets created by TickCats include `id` by default once the feature is implemented.
2. New ticket filenames use the generated ID instead of a date prefix.
3. Existing tickets without `id` continue to parse and load with their current filenames until migration.
4. Add an explicit CLI command in this feature, e.g. `tickcats ids migrate`, that assigns IDs to tickets missing them.
5. The migration command should update frontmatter and rename migrated ticket files to the new ID-based filename format.
6. Migration must preserve workflow folders: a backlog ticket stays in backlog, a ready ticket stays in ready, etc.
7. Migration must be idempotent: running it again does not change already-migrated tickets.
8. Migration must not overwrite an existing file; if the target filename already exists, fail clearly or choose a safe unique suffix.
9. Detail view can show `ID: —` for missing IDs before migration.
10. Duplicate IDs produce a warning that points to both file names and should block migration until resolved.

This keeps normal board loading safe while giving users a deliberate way to upgrade existing local boards.

## CLI and TUI display

Recommended display rules:

- Detail metadata: always show `ID: TC-A7K9Q2` or `ID: —`.
- CLI `list`: show ID between filename and priority when present, e.g.
  - `ticket.md  TC-A7K9Q2  [P1] Task: example`
- CLI `pick-next`: show ID when present, e.g.
  - `ticket.md  TC-A7K9Q2  [P1] Task: example`
- Board cards: do not show ID by default unless future user feedback asks for it. Board space is scarce.
- Search can later match by ID as plain text.

## Commit reference guidance

Use the ID in the commit body or subject when useful.

Examples:

```text
feat(tui): add deadline SLA bars

Refs: TC-A7K9Q2
```

or, for small local work:

```text
fix(store): preserve ticket id on moves refs TC-A7K9Q2
```

Keep Conventional Commit type/scope first; the ticket reference should not replace the commit subject.

## Edge cases

### Missing ID

Allowed during migration. Display `ID: —`. Future migration command can fill it.

### Duplicate ID

Warn during board load. Do not silently rewrite either file. Operations that rely on unique IDs should refuse ambiguous ID-based references until resolved.

### Malformed ID

Warn during board load. Treat the ticket as loaded if other required metadata is valid, but mark identity as unusable. A strict parse failure would be too disruptive during rollout.

### Copy preserving ID accidentally

Must be prevented in TickCats copy/paste implementation. Pasted tickets are new work items and need new IDs.

### Manual external edit removes ID

Ticket remains loadable but loses stable identity until migration or manual repair.

### Manual external edit changes ID

TickCats should accept a valid unique ID but this effectively changes identity. Document that users should avoid editing IDs unless repairing duplicates.

## Follow-up implementation ticket

Create a follow-up ticket to implement the recommended frontmatter short-ID strategy.
