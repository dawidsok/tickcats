---
name: tc-refine
description: >
  Grilling-style backlog refinement for the tc-* (tickcats) pipeline — a
  relentless one-question-at-a-time interview where every decision lands in a
  ticket immediately, the way grill-with-docs writes ADRs inline. Two modes:
  refine existing [to refine] tickets (challenge scope, sharpen AC, fix
  priority, wire prerequisites, split or kill), or crystallize a vague pain
  ("auth is a mess") into new tickets. Trigger phrases: "tc refine", "refine
  the backlog", "groom the backlog", "grill me about this ticket", "refine
  tickets", "create improvement tickets", "turn this pain into tickets". This
  is the tickcats-pipeline counterpart of /grill-with-docs; it sits between
  /tc-decompose and /tc-plan but also runs standalone at any time.
argument-hint: "[ticket-id | topic]"
allowed-tools:
  - Read
  - Edit
  - Write
  - Bash
  - Grep
  - Glob
  - AskUserQuestion
  - Task
---

# tc-refine: Grill the Backlog Into Shape

A refinement session is an interview, not a monologue. You grill the user about every unresolved aspect of a ticket (or a pain point) until you share an understanding — and you write each decision into a ticket **the moment it lands**, not in a batch at the end. Tickets are the docs; the grill produces them the way grill-with-docs produces ADRs.

## Grilling rules (binding for the whole session)

These rules are self-contained — do not look for a `/grilling` skill; it is not installed.

1. **Interview thoroughly.** Cover every aspect of the topic until you and the user have a shared understanding. Walk each branch of the decision tree. Where decisions depend on each other, resolve them sequentially — settle the upstream decision before asking about the downstream one.
2. **One question at a time.** Ask a single question, then STOP and wait for the answer. Never batch questions. Never ask "a few quick questions". The answer to one question shapes the next — batching destroys that.
3. **Always recommend.** Every question ships with your recommended answer and a one-line reason. The user should be able to reply "yes" and move on.
4. **Explore before asking.** If a question can be answered by reading the codebase, the board, or the PRD — read it instead of asking. Questions are for judgment calls, priorities, and intent; facts are your job.
5. **Never implement.** No code changes during the session. The only files you touch are ticket files under `.tickcats/`. If the user drifts into "let's just fix it now", note it in the ticket and steer back.

Use AskUserQuestion when the question has 2-4 discrete options (mark your recommendation in the option description); use plain prose for open-ended questions. Either way: one question, then wait.

## Tickcats facts (memorize before touching the board)

- Columns are folders: `.tickcats/{backlog,ready,doing,done,wont-do}/`. **State = folder, never frontmatter.**
- `tickcats new feat|task|bug "<title>"` creates a ticket in backlog and prints the created file path — capture it.
- `tickcats move <bare-filename> <from> <to>` moves between columns.
- Labels are square brackets in the title: `[to refine]`, `[blocked]`.
- Title format: `[labels] Feat|Task|Bug: <change-id>: <human title>` — change-id is kebab-case and stable; splits produce `<change-id>a`, `<change-id>b`.
- Frontmatter keys: `title`, `id` (TC-XXXXXX), `priority` (P0-P3), `created`, `updated`.
- Dependencies are a convention, not a field: `Prerequisites: TC-XXX, TC-YYY` line in `## Context`, plus `[blocked]` in the dependent's title until all prerequisites sit in `done/`.
- A ticket is unblocked when every ID on its `Prerequisites:` line resolves to a file in `done/` — check with Glob/Grep, not memory.

## Phase 0: Preflight and mode detection

Run `ls .tickcats/` — if the board is missing, stop: "No tickcats board found. Run `tickcats init` (or `/tc-decompose`) first."

Read `context/foundation/prd.md` and `context/foundation/lessons.md` if they exist — they are your priors for scope and priority arguments.

Detect the mode from the argument:

| Argument | Mode |
|---|---|
| A ticket ID (TC-XXXXXX), a change-id, or a path into `.tickcats/` | **Refine** — that ticket only, then offer the rest |
| `backlog`, `all`, or empty while `[to refine]` tickets exist in `.tickcats/backlog/` | **Refine** — all `[to refine]` tickets, highest priority first |
| Prose describing a pain or topic ("auth is a mess", "improve error handling") | **Improvement** |
| Empty and no `[to refine]` tickets | Ask: "Nothing labeled `[to refine]`. What's bothering you about the project?" → **Improvement** |

Ambiguous argument? Grep ticket titles and IDs across all columns first; only ask if the grep is inconclusive.

Open the session with a one-screen orientation before the first question:

```
tc-refine session — <Refine|Improvement> mode

  Target:  <ticket queue with priorities | the stated concern>
  Rules:   one question at a time, my recommendation attached,
           decisions written into tickets as they land, no implementation.

First question:
```

Then ask the first question and STOP.

## Refine mode: grill existing tickets

Build the queue: Grep `.tickcats/backlog/` for `[to refine]` in titles, sort by frontmatter priority (P0 first), then process one ticket at a time. Announce the queue before starting: "N tickets to refine, in this order: …".

Per ticket, read the file FULLY, read any tickets it names as prerequisites, and explore the code area it touches if that settles a question. Then grill across four fronts — one question at a time, your recommendation attached, skipping any front the ticket already answers convincingly:

1. **Scope** — the hardest front; open here. Is this ONE coherent PR-able deliverable? Too big (would touch many concerns, or its AC list reads like two features)? Actually still needed, or overtaken by other tickets / a PRD change? Do not soften: "I'd kill this — S-07 already covers the user-facing half" is a valid opening question.
2. **Context** — does `## Context` say WHY this exists, cite PRD refs (FR-NNN, US-NN), and end with an outcome sentence a stranger could act on? Draft the sharpened version and ask the user to confirm or correct it.
3. **Acceptance Criteria** — every `- [ ]` item must be testable: an observable behavior, a command that passes, a state you can point at. "Works well" and "is improved" are not criteria — propose the concrete replacement and ask.
4. **Priority and prerequisites** — argue the P-level against the board (what does it unlock? what blocks it?). Hunt for missed `Prerequisites:` — scan other tickets for work this one silently assumes. If you find one, add it and add `[blocked]` unless that prerequisite is already in `done/`.

The shape of a good grill question — statement of evidence, then ONE question, then a recommendation:

```
s-04's AC says "search should feel fast". That's not testable — nothing can
check "feel". The PRD's NFR-02 sets 300ms p95 for search.

Should the criterion be "p95 search latency < 300ms on the seeded 10k-row
dataset", or do you have a different bar in mind?

My recommendation: yes, use NFR-02's 300ms — it's already the committed
number, and the seed dataset makes it reproducible.
```

**Write immediately after each decision.** When the user settles Context, Edit the ticket's Context right then. When an AC is sharpened, Edit it right then. Bump `updated:` in frontmatter on first edit. Do not accumulate a change list for later.

Each ticket exits through exactly one gate:

**Refined in place** — Context and AC rewritten, priority confirmed, prerequisites wired. Remove `[to refine]` from the frontmatter `title:` — labels live in ONE comma-separated bracket group at the very start of the title (`[blocked, to refine] Feat: …`); drop the label from the group, or the whole group if it empties. Then check its `Prerequisites:` line: if every listed ID is in `done/` (or the line is absent), run `tickcats move <bare-filename> backlog ready`; otherwise ensure `[blocked]` is in the title and leave it in backlog.

**Split** — user agrees it's more than one deliverable. Mechanics, in order:
1. For each part: `tickcats new feat|task|bug "<change-id>a: <part title>"` (siblings `a`, `b`, …; NO labels or kind prefix in the argument — the CLI adds the prefix and ignores labels). Capture each printed path, then edit each file's frontmatter `title:` to prepend `[to refine]` (single leading bracket group) unless the grill already fully refined that part.
2. Write each sibling's `## Context` and draft `## Acceptance Criteria` from the session's decisions; wire `Prerequisites:` between siblings and carry over the original's external prerequisites; set priorities (they may differ — argue each).
3. Append `Split into: TC-AAAA, TC-BBBB` to the original's body, then `tickcats move <bare-filename-of-original> backlog wont-do`.

**Killed** — user decides it's not worth doing. Append a one-line rationale to the body (e.g. `Killed: superseded by s-07 after PRD v2 dropped offline mode.`), then `tickcats move <bare-filename> backlog wont-do`.

After each ticket, print a one-line verdict (`s-03 → refined, moved to ready` / `s-05 → split into s-05a, s-05b` / `s-09 → killed`) and move to the next in the queue. The user may stop the session at any ticket boundary — everything processed so far is already written.

## Improvement mode: grill a pain into tickets

The input is a vague concern. The output is one or more concrete, independently-actionable tickets. Vague in, vague out is failure — do not create a ticket whose title is the complaint restated.

**Ground first (optional but default-on).** Unless the concern is purely process/product (no code to inspect), spawn 1-2 read-only Explore sub-agents in a single message before the first question: "Find how <topic> is currently handled — list the mechanisms, inconsistencies, and hotspots with file:line references. Read-only." The grill argues from their evidence, not from vibes. Skip only if the user says "no need to look" or there is no codebase.

**Then grill** — one question at a time, recommendation attached — until each strand of the concern crystallizes. Useful pressure points:

- "You said 'auth is a mess' — the scan found three concrete smells: <A file:line>, <B>, <C>. Which one actually bit you?" (recommend the one with the worst blast radius)
- "Is this one problem or several?" — force the split until each item is independently shippable as one PR.
- "What observable behavior changes when this is fixed?" — that answer IS the first acceptance criterion; if the user can't answer it, the item isn't ready to be a ticket yet — keep grilling that strand or drop it.
- "Bug or task?" — a defect in existing behavior is `bug`; refactoring/enablement is `task`. Recommend one.
- Priority: argue it against the board and the PRD ("nothing in ready/ depends on this — I'd say P2"). The user gets the final call.

**Create each ticket the moment it crystallizes** — not at session end:

1. `tickcats new task|bug "<change-id>: <title>"` — mint a fresh kebab-case change-id. NO labels or kind prefix in the argument (the CLI adds the prefix and ignores labels).
2. Edit the created file: prepend labels to the frontmatter `title:` as ONE leading comma-separated bracket group — `[to refine]`, or `[blocked, to refine]` if the item depends on not-done tickets (then also add the `Prerequisites:` line). Skip `[to refine]` only when the grill already produced final Context, testable AC, and an agreed priority. Set `priority`; write `## Context` with the evidence (file:line refs from the Explore agents) and the rationale from the session; write draft `## Acceptance Criteria` from the grill's answers.
3. Print one line: `Created TC-XXXX P2 task <change-id>: <title>`.

Continue until the user confirms the original concern is fully covered — ask exactly that as the closing question: "Does <A + B + C> cover 'auth is a mess', or is there a strand left?"

## Session end (both modes)

Run `tickcats list`. Report in this shape:

```
REFINEMENT SESSION COMPLETE

  Created:  TC-XXXX  P2  task  <change-id>: <title>          (backlog, [to refine])
  Edited:   TC-YYYY  P1  feat  <change-id>: <title>          (refined)
  Moved:    TC-YYYY  backlog → ready
  Split:    TC-ZZZZ  → TC-AAAA, TC-BBBB                       (original → wont-do)
  Killed:   TC-QQQQ  <one-line rationale>                     (→ wont-do)

Next: /tc-plan <highest-priority ready or [to refine] ticket>
```

Omit empty categories. If nothing changed, say so plainly — a session that concludes "the backlog is fine" is a valid outcome; do not manufacture edits to justify the session.

## Critical guardrails

1. **One question, then silence.** The strongest failure mode of this skill is asking three questions in one message. If you catch yourself writing a second question mark, delete everything after the first.
2. **Edits land during the grill.** A decision that isn't in a ticket file within the same turn it was made is a decision that will be lost. Never say "I'll update the tickets at the end."
3. **No implementation.** Ticket files only. Not one line of source code, config, or test changes — even "trivial" ones the grill surfaces. Capture them as tickets instead.
4. **Recommendations are mandatory.** A question without your recommended answer pushes work onto the user that you were supposed to do. Recommend, with a reason, every time.
5. **Explore beats asking.** "Which files handle sessions?" is never a question for the user. Grep first. Ask only what the code cannot answer: intent, priority, appetite, scope.
6. **Challenge, don't rubber-stamp.** If a ticket looks fine, say so and move on fast — but if scope smells too big, priority smells inflated, or an AC is untestable, push. The user hired a grill, not a scribe. State the objection once; if the user overrules it, write their decision and proceed without further pushback.
7. **State = folder.** Never record status in frontmatter or body. Moving a ticket means `tickcats move`, nothing else.
8. **Splits kill the original.** Never leave a split parent in backlog. Siblings inherit, original goes to wont-do with the `Split into:` pointer — every time, no exceptions.

## Relationship to other skills

- `/tc-decompose` — produces the `[to refine]` stubs this skill consumes. Refine mode is the natural next step after decompose.
- `/tc-plan` — consumes refined tickets and writes the full implementation plan into the ticket body. tc-refine sharpens WHAT and WHY; tc-plan owns HOW. Do not write `## Implementation Plan`, `## Architecture`, or phase breakdowns here.
- `/grill-with-docs` — the non-tickcats sibling: same grilling loop, but decisions land in ADRs/glossary instead of tickets.

tc-refine is standalone-safe: run it any time the backlog has drifted or a pain needs shaping — no pipeline position required.
