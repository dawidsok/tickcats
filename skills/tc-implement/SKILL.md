---
name: tc-implement
description: Implement a refined tickcats ticket from .tickcats/ready/ with verification. Use when the user says "tc implement", "implement this ticket", "implement TC-XXXXXX", or "pick next ticket and implement". Use AFTER /tc-plan (optionally /tc-plan-review). For the classic context/changes workflow use /10x-implement instead.
argument-hint: "[ticket-id-or-path]"
allowed-tools:
  - Read
  - Glob
  - Grep
  - Write
  - Edit
  - Bash
  - Task
  - AskUserQuestion
  - TaskCreate
  - TaskUpdate
  - TaskList
  - TaskGet
---

# Implement Ticket

You are tasked with implementing a refined tickcats ticket. The ticket body is the full executable plan: `## Implementation Plan` contains phases with specific changes, and a canonical `## Progress` section drives execution state. There is NO `context/changes/` folder, NO `change.md`, NO `plan.md` — the ticket file is the single artifact and the single source of truth.

**Tickcats facts** (internalize before touching the board):

- Columns are folders: `.tickcats/{backlog,ready,doing,done,wont-do}/`. A ticket's state IS its folder — never a frontmatter field.
- Move tickets only via `tickcats move <bare-filename> <from> <to>`. **Moving relocates the file** to `.tickcats/<to>/<same-filename>` — recompute the ticket path after EVERY move and use the new path from then on.
- Labels live in title brackets: `[blocked]`, `[to refine]`.
- Frontmatter: `title`, `id` (TC-XXXXXX), `priority` (P0–P3), `created`, `updated`.
- Ticket title format: `[labels] Feat|Task|Bug: <change-id>: <human title>`. The `<change-id>` (kebab-case, e.g. `s-03`) is the token between the kind prefix and the human title — you need it for commit messages.

## Initial Setup

When this command is invoked:

1. **Resolve the ticket**:
   - If invoked as `/tc-implement TC-XXXXXX [phase N]`, locate the ticket file: `grep -rl "id: TC-XXXXXX" .tickcats/` (search all columns — you need to know WHICH column it is in).
   - If invoked with a file path (e.g., `@.tickcats/ready/s-03-user-login.md`), accept it.
   - If nothing was provided, run `tickcats pick-next`, read the picked ticket's frontmatter and `## Context`, then confirm with `AskUserQuestion`:

     - question: "pick-next chose [id] — [title]. Implement it?"
       header: "Picked ticket"
       options:
       - label: "Yes, implement it (Recommended)"
         description: "Proceed with this ticket."
       - label: "Pick another"
         description: "Show `tickcats list` and let me choose a different ready ticket."
       - label: "Abort"
         description: "Stop without touching the board."
       multiSelect: false

2. **Refusal gates** — check in order, print the reason and STOP on the first failure:
   - Title contains `[blocked]` → "This ticket is blocked. Its prerequisites are not done yet." STOP.
   - Title contains `[to refine]` → "This ticket is an unrefined stub. Run `/tc-plan [id]` first." STOP.
   - Ticket is NOT in `.tickcats/ready/`:
     - In `.tickcats/doing/` → this is a **resume** (see "Resuming Work" below). Proceed without moving.
     - In `backlog/` → "Ticket is unplanned. Run `/tc-plan [id]` first." STOP.
     - In `done/` or `wont-do/` → "Ticket is closed. Nothing to implement." STOP.
   - **Prerequisites gate**: parse the `Prerequisites:` line in `## Context` (comma-separated TC-XXXXXX ids; absent line = no prerequisites). For each id, verify it lives in `.tickcats/done/`: glob `.tickcats/done/*<id>*`; if no filename match, fall back to `grep -l "id: <id>" .tickcats/done/`. If ANY prerequisite is not in done/, list the missing ids, say the ticket should carry `[blocked]`, and STOP.

## Getting Started

Once the ticket passes the gates:

- Read the ticket file completely. The `## Progress` section is authoritative for execution state — checkmarks (`- [x]`) live ONLY there (plus `## Acceptance Criteria`, which you do not drive from). Phase blocks under `## Implementation Plan` contain plain `- ` bullets, no checkboxes.
- Read `context/foundation/lessons.md` if present and internalize each entry before starting any phase — these are the team's accepted recurring rules and must shape every implementation choice you make in this run.
- Read all files referenced in the ticket's `## Current State` and `## Implementation Plan` sections.
- **Read files fully** — never use limit/offset parameters, you need complete context.
- Think deeply about how the pieces fit together.
- **Move the ticket to doing** (skip if resuming from doing/): run `tickcats move <bare-filename> ready doing`. Then **recompute the ticket path** — it now lives at `.tickcats/doing/<same-filename>`. All subsequent Reads, Edits, and staging use the doing/ path.
- Extract the `<change-id>` from the title (`Feat: s-03: User login via OAuth` → `s-03`). You need it for every commit subject.
- Count total phases (from `### Phase N:` headers) and create one TaskCreate entry per phase (these appear in the user's status bar):
  - For each phase, create a task with `subject: "Phase N: [Phase Name]"` and `activeForm: "Implementing Phase N"`
  - Set the current phase to `in_progress` via TaskUpdate before starting work
  - Mark each phase `completed` via TaskUpdate when its success criteria pass
- **Find the next pending step** by scanning the `## Progress` section: the first `- [ ]` line in document order is where you start. If a `phase N` argument was passed, jump to the first `- [ ]` inside `### Phase N:` instead.
- Start implementing if you understand what needs to be done.

## Implementation Philosophy

Plans are carefully designed, but reality can be messy. Follow the plan's intent while adapting to what you find; implement each phase fully before moving to the next; verify your work makes sense in the broader codebase context; update checkboxes in the ticket as you complete sections. When things don't match the plan exactly, think about why and communicate clearly — the plan is your guide, but your judgment matters too.

If you encounter a mismatch:

- STOP and think deeply about why the plan can't be followed
- Present the issue clearly as text:

  ```
  Issue in Phase [N]:
  Expected: [what the ticket says]
  Found: [actual situation]
  Why this matters: [explanation]
  ```

- Then use `AskUserQuestion` to get a structured decision:

  AskUserQuestion:
  - question: "How should I handle this mismatch?"
    header: "Mismatch"
    options:
    - label: "Adapt and continue"
      description: "Adjust the implementation to match reality. I'll explain the adaptation."
    - label: "Skip this part"
      description: "Move on to the next section/phase. This change isn't needed."
    - label: "Stop and re-plan"
      description: "This mismatch is too significant. Run /tc-plan again to update the ticket first."
      multiSelect: false

## Tracking files touched during a phase

The phase-end commit ritual (see "Verification Approach" below) stages files from a **touched-file set** that you maintain in working memory throughout each phase. This set is the canonical input to `git add` — never fall back to `git status` heuristics for staging decisions.

**Discipline**:

- Every time you call `Edit` or `Write` on a file during the current phase, add its repo-relative path to the touched-file set.
- The set always contains the **ticket file itself** (its current `.tickcats/doing/` path) — the board is committed, and each phase produces at least one Edit to the ticket's `## Progress` section. Add it on entry to a phase even before any checkboxes flip.
- **Phase 1 bootstrap**: the `ready → doing` move is a file rename that sits dirty after the move. Seed the touched-file set with BOTH paths — the old `.tickcats/ready/<file>` (stages the deletion) and the new `.tickcats/doing/<file>` — so the move lands in the first phase's commit rather than dangling.
- The set **resets at each phase boundary**. After the phase-end commit completes, clear it before starting the next phase (the ticket's current path re-enters immediately on the next phase).
- This list overrides any heuristic from `git status`. If the touched set is `{src/a.ts, .tickcats/doing/s-03.md}` but `git status --porcelain` also reports `src/c.ts` dirty, `src/c.ts` is unrelated — handle it via the dirty-path prompt in the ritual, never silently bundle it into the commit.

## Tracking issue/task references for commits

Before proposing any phase-end or epilogue commit message, scan the conversation for external tracker references tied to this work (Jira keys, Linear ids, GitHub `#123`/URLs). If present, include them comma-separated on a `Refs:` line in the commit body, preserving exact identifiers, on every phase-end and epilogue commit unless the user narrows one to a specific phase. Do NOT invent references — the TC id and change-id are already carried by the subject line and the committed ticket file.

## Verification Approach

After implementing a phase:

- Run the success criteria checks from the phase's `#### Automated` subsection (usually `make check test` or equivalent covers everything)
- Fix any issues before proceeding
- Update your progress in your tasks and in the ticket's `## Progress` section
- **Mutate ONLY the `## Progress` section.** All other ticket sections (Context, Requirements, Architecture, Current State, Implementation Plan, What We're NOT Doing, review sections) are read-only during implementation. Use Edit to flip `- [ ] N.M <title>` → `- [x] N.M <title>` in Progress as each step completes. Do NOT edit Phase block bullets, do NOT add HTML comment markers, and do NOT write any state-file sidecar.
- **Run the phase-end commit ritual**: After all automated checks pass for the phase, walk through this sequenced ritual to author one Conventional-Commits commit and write the closing short SHA back into every Progress row flipped during the phase.

  1. **Manual confirmation gate.** Inform the human that automated verification passed and list the manual verification items from the phase's `#### Manual` subsection. Pause here. Do not proceed until the human confirms manual testing succeeded. Use this format:

     ```
     Phase [N] Complete - Ready for Manual Verification

     Automated verification passed:
     - [List automated checks that passed]

     Please perform the manual verification steps listed in the ticket:
     - [List manual verification items]

     Let me know when manual testing is complete so I can proceed to the commit step.
     ```

     **Cross-phase manual rollup (final phase only).** Before printing the gate message, determine whether the current phase is the final phase: scan the `## Progress` section for `### Phase M:` headings and treat the current phase as final iff no heading with `M > N` exists in document order. If the current phase is **not** final, the gate message is exactly the format above — no rollup. If the current phase **is** final, after the manual-steps block, scan the entire Progress section for `- [ ]` rows that sit under a `#### Manual` subsection in any phase **other than the current one**. If any such rows exist, append (in document order, one row per line, formatted as `<phase>.<index> <title>` — strip any `- [ ]` prefix and any trailing ` — <sha>` suffix):

     ```
     Pending manual checks from earlier phases:
     - [phase.index title]
     ```

     If no earlier-phase manual rows are pending, omit the rollup block entirely. The gate still pauses for human confirmation; this is informational, not a hard block.

  2. **Compute the staging set.** Take the touched-file set maintained during the phase (see "Tracking files touched during a phase" above) and union it with the ticket file's current path (`.tickcats/doing/<file>`). The ticket is always staged because each phase produces at least one Edit to its `## Progress` section.

  3. **Detect unrelated dirty paths.** Run `git status --porcelain` and intersect with paths *outside* the staging set. If the dirty-but-untouched set is non-empty, present the offending paths and use `AskUserQuestion`:

     - question: "<N> unrelated path(s) are dirty. How should I handle them?"
       header: "Dirty paths"
       options:
       - label: "Continue — stage only the planned set (Recommended)"
         description: "Commit only files this phase touched. Leave the unrelated paths dirty for you to handle separately."
       - label: "Stage all"
         description: "Add the unrelated paths to this commit. You take responsibility for the broader scope."
       - label: "Abort"
         description: "Stop the phase commit. Resolve the dirty paths first, then re-run the ritual."
       multiSelect: false

     If the dirty-but-untouched set is empty, skip this step.

  4. **Stage explicitly by path.** `git add` each file in the chosen set by name. Do NOT use `git add -A` or `git add .` — explicit paths only.

  5. **Check empty diff.** Run `git diff --cached --quiet`. Exit code 0 means no staged diff. If empty, print:

     ```
     Phase [N] had no diff to commit; rows remain SHA-less. This is a valid state for manual-only or no-op phases.
     ```

     Set `SHA=""` and skip to step 8.

  6. **Propose a Conventional-Commits message.** Build the subject as `<type>(<change-id>): <phase title> (p<N>)`, where `<change-id>` is the one extracted from the ticket title and `<type>` is one of `feat / fix / chore / refactor / docs` chosen from the phase's nature (the ticket's kind prefix is a hint — `Feat:` tickets usually produce `feat`, `Bug:` tickets `fix` — but the phase's actual content decides). The phase title is the meaningful part and leads; the `(p<N>)` suffix carries the phase index. Build a short body listing the touched files, plus the `Refs:` line when applicable. Use `AskUserQuestion`:

     - question: "Approve commit message?"
       header: "Commit msg"
       options:
       - label: "Approve as proposed (Recommended)"
         description: "Use the message as drafted."
       - label: "Edit subject line"
         description: "Override the subject; keep the body."
       - label: "Override entirely"
         description: "Replace both subject and body."
       multiSelect: false

  7. **Commit via heredoc.** Run `git commit` per the global commit-message protocol:

     ```bash
     git commit -m "$(cat <<'EOF'
     <type>(<change-id>): <phase title> (p<N>)

     <short body listing touched files>
     <Refs: issue/task references, if applicable>
     EOF
     )"
     ```

     Never pass `--no-verify`, `--amend`, or signing-bypass flags. If a pre-commit hook fails, fix the underlying issue and create a NEW commit — the original commit did NOT happen, so amending would touch the previous phase's commit instead.

  8. **Capture the short SHA.** Run `git rev-parse --short HEAD` and store as `SHA`. Skip this step if `SHA=""` was set by step 5.

  9. **Write the SHA back into Progress.** For every Progress row flipped during this phase, run a targeted Edit on the ticket file:

     - Find: `- [x] N.M <title>` (no existing ` — <sha>` suffix at end of line)
     - Replace with: `- [x] N.M <title> — <SHA>`

     Skip rows that already carry a SHA suffix (resume safety: if the ritual is re-entered after a partial run, do not double-append). If `SHA=""`, skip the append entirely — the rows stay SHA-less.

  10. **Bump the ticket's `updated:` frontmatter** to today's date (idempotent per phase).

  11. **Reset the touched-file set.** Clear it before starting the next phase. The ritual is self-contained per phase.

- **Next phase decision**: If there is a next phase, help the user decide whether to continue or start fresh.

  AskUserQuestion:
  - question: "Phase [N] complete. How to proceed?"
    header: "Next phase"
    options:
    - label: "Continue to Phase [N+1]"
      description: "Stay in this context and proceed to the next phase."
    - label: "Clear context first"
      description: "Copy resume command to clipboard. Start fresh for Phase [N+1]."
    - label: "Review this phase first"
      description: "Run /tc-impl-review to verify implementation against the ticket before proceeding."
      multiSelect: false

  **If user chooses to review**: Run `/tc-impl-review [id] phase [N]`. After the review completes, re-present the continue/clear decision (without the review option this time).

  **If user chooses to continue**: Proceed directly to the next phase — read its ticket section, set the task to `in_progress`, and implement. No need to re-read the entire ticket or already-loaded files.

  **If user chooses to clear**: Copy the resume command to clipboard and display it:

  ```bash
  echo -n "/tc-implement <TC-id> phase [next-phase-number]" | pbcopy 2>/dev/null || echo -n "/tc-implement <TC-id> phase [next-phase-number]" | xclip -selection clipboard 2>/dev/null || true
  ```

  ```
  → /tc-implement <TC-id> phase [next-phase-number] (✓ copied)
  ```

If instructed to execute multiple phases consecutively, skip the AskUserQuestion between phases.

Do not check off items in the manual testing steps until confirmed by the user.

## State Tracking

**The `## Progress` section in the ticket file is the single source of truth.** State-as-folder tells you WHICH ticket is in flight (doing/); Progress tells you WHERE inside it you are. No state file. No comment markers.

### After each step

Use Edit to flip exactly one Progress line at a time:

- Find: `- [ ] N.M <title>`
- Replace with: `- [x] N.M <title>`

Do not append the SHA suffix on a per-step Edit — the SHA is written back at phase end by the commit ritual, and only the closing commit's SHA goes onto every row that flipped during the phase. Mid-phase, completed rows sit `[x]` without a SHA suffix; this is a valid intermediate state.

### After each phase

When all `- [ ]` items inside `### Phase N:` are now `- [x]`: run the phase-end commit ritual (manual confirmation → staging → dirty-path prompt → commit → SHA write-back).

Empty-diff phases (manual-verification-only or no-op adapted phases) commit nothing and leave their rows SHA-less. This is intentional — not every phase produces code.

### After all phases

When every `- [ ]` in the entire `## Progress` section is now `- [x]`:

1. **Defensive pending-items surface.** Re-scan the entire `## Progress` section one last time for any `- [ ]` rows. Under normal flow this is a no-op — it exists to make unexpected stragglers explicit rather than silently lost (partial run, manual edit, resume path). If the count is non-zero, list each row as `<phase>.<index> <title>` grouped by Automated vs Manual subsection in document order, then ask via `AskUserQuestion`:

   - question: "<N> Progress item(s) still pending. How to proceed?"
     header: "Stragglers"
     options:
     - label: "Pause (Recommended)"
       description: "STOP without closing the ticket. Address the stragglers manually, then re-run /tc-implement."
     - label: "Proceed to close-out"
       description: "Move the ticket to done anyway. The unchecked rows stay visible in the done ticket."
     multiSelect: false

   On "Pause": STOP immediately — the ticket stays in doing/. On "Proceed": continue below. If the count is zero, skip this step.

2. **Move the ticket to done**: `tickcats move <bare-filename> doing done`. **Recompute the ticket path** — it now lives at `.tickcats/done/<same-filename>`.

3. **Unblock dependents.** This ticket's completion may unblock others:
   - `grep -lE "Prerequisites:.*<this-TC-id>" .tickcats/backlog/ .tickcats/ready/ -r` to find dependents.
   - For each dependent: parse its full `Prerequisites:` list and check EVERY id against `.tickcats/done/` (glob `.tickcats/done/*<id>*`, fall back to `grep -l "id: <id>"`).
   - If ALL of a dependent's prerequisites now live in done/, Edit its frontmatter `title:` to remove `blocked` from the leading label group (labels are ONE comma-separated bracket group at the start of the title — `[blocked, to refine]` becomes `[to refine]`; drop the group entirely if it empties) and bump its `updated:`. Do NOT rename the file, do NOT move it between columns.
   - If some prerequisites are still open, leave it `[blocked]` and note which ids remain.
   - Track every dependent file you edit — they join the epilogue staging set.

4. **Run the epilogue commit.** The final phase's commit cannot contain its own SHA (chicken-and-egg), so the final SHA write-back, the `doing → done` rename, and any dependent unblock edits sit dirty after step 3. Author one closing commit to land them:
   1. Stage explicitly by path: `.tickcats/doing/<file>` (stages the deletion side of the rename), `.tickcats/done/<file>`, and every dependent ticket file edited in step 3. No `git add -A`, no `git add .`.
   2. Run `git diff --cached --quiet`; if exit code 0, skip the epilogue (nothing trailing to commit).
   3. Propose subject `chore(<change-id>): close out ticket (epilogue)` with a short body noting the final SHA write-back, the move to done, and any tickets unblocked, plus the `Refs:` line when applicable. Use AskUserQuestion to approve / edit subject / override (same options as the phase ritual).
   4. Commit via heredoc per the global protocol (never `--no-verify` / `--amend`).
   5. Do NOT write the epilogue's own SHA back into the ticket — its only job is to land the trailing edits cleanly.

### "Where am I?" — derived, not stored

The board answers at the column level: `tickcats list`, or the ticket's folder. Inside the ticket, parse `## Progress`: the first `- [ ]` line is the next step; the current phase is the `### Phase N:` heading immediately above it; completion is `count([x]) / count([ ] + [x])`. No JSON, no markers, no sidecar.

## Ticket Completion

When ALL phases are implemented, the ticket sits in done/, and the epilogue has landed:

1. Present the closing summary:

```
Ticket [TC-id] implemented and moved to done. 🎉

Summary:
- Phases completed: [N]
- Files changed: [list key files]
- Newly unblocked: [TC-ids + titles whose [blocked] label was removed, or "none"]
- Still blocked: [dependents with remaining open prerequisites, if any]

Next: run `/tc-plan [unblocked-id]` to refine an unblocked ticket, or `tickcats pick-next` to grab the next ready one.
```

2. Then offer a final review via AskUserQuestion — header "Ticket done", question "Ticket complete. Would you like a final implementation review?", options: "Run full review (/tc-impl-review)" (comprehensive review of all phases against the ticket, catches cross-phase issues) / "Skip review — I'm satisfied". If user chooses review → run `/tc-impl-review [TC-id]` (no phase number = full review).

## If You Get Stuck

When something isn't working as expected: make sure you've read and understood all the relevant code; consider whether the codebase has evolved since the ticket was planned; present the mismatch clearly and ask for guidance.

Use sub-tasks sparingly — mainly for targeted debugging or exploring unfamiliar territory:

- **Explore** (`subagent_type: "Explore"`) — Fast search for files, patterns, similar code
- **general-purpose** (`subagent_type: "general-purpose"`) — Deep analysis requiring multi-step reasoning

## Resuming Work

If the ticket is already in `.tickcats/doing/` when you resolve it, a previous run was interrupted — this is a resume, not an error:

- Do NOT run `tickcats move` again; the ticket is already where it belongs.
- Trust that completed work is done: rows marked `- [x]` (with or without SHA suffix) stand.
- Pick up from the first `- [ ]` line in `## Progress`.
- If the first pending row's phase has earlier `[x]` rows without SHAs, the phase was interrupted mid-flight — finish its remaining steps, then run the ritual normally; the closing SHA covers all rows flipped in that phase (including the pre-existing SHA-less ones).
- Verify previous work only if something seems off.

Remember: You're implementing a solution, not just checking boxes. Keep the end goal in mind and maintain forward momentum. The ticket is self-contained by design — if you find yourself needing context it doesn't provide, that's a planning gap worth reporting, not something to silently improvise around.
