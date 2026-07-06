---
name: tc-plan
description: Tickcats-pipeline variant of 10x-plan (absorbs 10x-research). Refine a `[to refine]` backlog ticket into a fully self-contained implementation plan written INTO the ticket body — research digest, mermaid architecture, phased plan, progress checkboxes. Use when the user says "tc plan", "plan this ticket", "refine ticket into implementation plan", "plan TC-XXXXXX", or wants the tickcats ticket-based workflow. Use AFTER /tc-decompose. For the classic context/changes workflow use /10x-plan instead.
argument-hint: "[ticket-id-or-path]"
allowed-tools:
  - Read
  - Glob
  - Grep
  - Write
  - Edit
  - Bash
  - Task
  - Agent
  - AskUserQuestion
  - TaskCreate
  - TaskUpdate
  - TaskList
  - TaskGet
---

# Ticket Plan (tickcats)

You are tasked with refining a tickcats ticket into a detailed, self-contained implementation plan through an interactive, iterative process. Be skeptical, thorough, and collaborative. There is NO plan.md, NO plan-brief.md, NO research.md, NO context/changes/ folder in this workflow — **the ticket body IS the plan**. You edit the ticket's .md file in `.tickcats/` directly.

The finished ticket must be readable by a human developer who has **never seen the repo**: they should be able to implement it from the ticket alone.

## Tickcats Board Facts

- Columns are folders: `.tickcats/{backlog,ready,doing,done,wont-do}/`. **State = folder location, never frontmatter.**
- Ticket frontmatter keys: `title`, `id` (TC-XXXXXX), `priority` (P0–P3), `created`, `updated`.
- Labels live as square brackets in the title: `[to refine]`, `[blocked]`. Title format: `[labels] Feat|Task: <change-id>: <human title>`.
- Dependencies are a convention, not a CLI feature: `Prerequisites: TC-XXX, TC-YYY` line inside `## Context` + `[blocked]` label on the dependent ticket.
- CLI: `tickcats new feat|task|bug "<title>"` (prints the created file path), `tickcats list`, `tickcats move <bare-filename> <from> <to>`, `tickcats pick-next`.

## Step 0: Resolve the Ticket

1. **If an argument was provided**:
   - A file path → that's the ticket. Read it FULLY.
   - A `TC-XXXXXX` id → find it: `grep -rl "id: TC-XXXXXX" .tickcats/` — expect it in `backlog/`. Read it FULLY.
2. **If no argument**: pick the highest-priority `[to refine]` ticket from `.tickcats/backlog/` (P0 first, then P1, P2, P3; skip `[blocked]` tickets whose prerequisites are not all in `.tickcats/done/`). Announce which ticket you picked and why, and confirm with the user before proceeding.
3. **If no `[to refine]` tickets exist**, print: "No `[to refine]` tickets in the backlog. Run /tc-decompose to create stubs, or pass a ticket path explicitly." and STOP.
4. **Refuse** tickets in `done/` or `wont-do/` — print: "This ticket is closed. Pick a backlog ticket instead." and STOP.

## Step 1: Read Everything Fully

1. **Read these files immediately and FULLY** (Read tool WITHOUT limit/offset — NEVER read partially):
   - The ticket file itself
   - `context/foundation/prd.md` if present — the ticket's Context section references FR-NNN / US-NN ids defined there
   - `context/foundation/lessons.md` if present — treat its rules as priors when probing scope, edge cases, and architecture choices
   - `context/foundation/shape-notes.md`, `tech-stack.md` if present and relevant
   - Any files the user or the ticket body explicitly mentions
   - **CRITICAL**: DO NOT spawn sub-agents before reading these files yourself in the main context

2. **Extract from the ticket stub**: the change-id, PRD refs, `Prerequisites:` ids, draft Acceptance Criteria, and the outcome sentence. These are decisions already made by /tc-decompose — don't re-ask them.

## Step 2: Research the Codebase (absorbed from 10x-research)

No research.md is written. Findings are digested straight into the ticket's `## Current State` section later.

1. **Decompose the ticket's touched area into research dimensions** and create tasks via TaskCreate to track them (visible in the user's status bar; TaskUpdate as each completes).

2. **Spawn 2-4 parallel sub-agents in a single message**:
   - **Explore** (`subagent_type: "Explore"`) — fast file/pattern search: "find all files related to X", "find similar implementations of Y", "find where Z is wired up"
   - **general-purpose** (`subagent_type: "general-purpose"`) — deep analysis requiring reading many files and multi-step reasoning: "explain how the A system works end to end"
   - Each prompt must be specific, read-only, and request **file:line references** and usage patterns (not just definitions)

3. **Wait for ALL sub-agents to complete**, then **read every file they identified as relevant FULLY** into the main context. Sub-agents locate; you verify by reading.

4. **Synthesize**: cross-reference the ticket requirements with actual code, note patterns and conventions to follow, constraints to work within, and discrepancies with the stub's assumptions. Don't quote large blocks — digest.

## Step 3: Split Decision Tree

Run this AFTER research, BEFORE the deep interview. With codebase reality in hand, estimate the plan's true footprint, then walk the tree:

1. **Is this one coherent PR-able deliverable?** (single concern, lands with green CI, deployable on its own)
   - NO → **split by concern**.
2. **Does it predict >8 files touched, OR >3 phases, OR >~500-line plan body?**
   - YES → **split at a phase boundary** (each sibling = one or more consecutive phases that stand alone).
3. **Otherwise** → plan it as one ticket. Skip to Step 4.

**If splitting, confirm first.** Present the proposed siblings (titles, one-line scope each, dependency order), then use AskUserQuestion:

- question: "This ticket is too big for one PR. Split it as proposed?"
  header: "Split"
  options:
  - label: "Yes, split as proposed"
    description: "Create the sibling tickets, retire the original, continue planning the first unblocked sibling."
  - label: "Adjust the split"
    description: "I'll describe different boundaries before you create anything."
  - label: "Don't split"
    description: "Plan it as one oversized ticket anyway — I accept the context-window risk."
    multiSelect: false

**Split mechanics** (only after user approval):

1. Create sibling tickets via `tickcats new feat|task "<change-id>a: <title>"`, then `<change-id>b`, … (kind matches the original; NO labels or kind prefix in the argument — the CLI adds the prefix and ignores labels; it prints each created file path — capture it). Edit each sibling's frontmatter `title:` to prepend `[to refine]` as a single leading bracket group, and stub each body with `## Context` (PRD refs, outcome sentence, scope carved from the original) + draft `## Acceptance Criteria`.
2. Wire ordering: add `Prerequisites: TC-<sibling-a-id>` to sibling b's Context (and so on down the chain), and add `blocked` to the label group of every sibling after the first (one group, comma-separated: `[blocked, to refine]` — never two bracket groups).
3. Retire the original: append `Split into: TC-<a>, TC-<b>, …` to its body, then `tickcats move <bare-filename-of-original> backlog wont-do`.
4. **Continue planning the FIRST unblocked sibling** — it becomes "the ticket" for the rest of this session.

## Step 4: Complexity-Scaled Interview

1. **Present informed understanding first**:

   ```
   Based on the ticket and my research of the codebase, I understand we need to [accurate summary].

   I've found that:
   - [Key discovery — code reference, existing asset, or constraint]
   - [Relevant pattern or convention discovered]
   - [Potential complexity or edge case identified]
   ```

2. **Assess complexity and confirm it**:

   ```
   **Complexity Assessment: [HIGH / MEDIUM / LOW]**

   [2-3 sentences on WHY — systems touched, integration points, data model changes,
   unknown unknowns, testing surface.]

   I'd like to ask **[N] questions** across multiple rounds to nail down [key decision areas].
   ```

   | Level      | Questions | When to use                                                                                              |
   | ---------- | --------- | -------------------------------------------------------------------------------------------------------- |
   | **LOW**    | 4-6       | Clear requirements, few moving parts, follows established patterns. E.g. single-file change, config tweak. |
   | **MEDIUM** | 7-10      | Multiple interacting components, real design decisions, edge cases worth discussing. E.g. multi-file feature, new API endpoint. |
   | **HIGH**   | 11-15     | Cross-cutting concerns, significant unknowns, expensive rework if wrong. E.g. system redesign, data migration. |

   Confirm via AskUserQuestion:
   - question: "Does this complexity assessment match your expectations?"
     header: "Complexity"
     options:
     - label: "Agree — proceed with [N] questions"
       description: "The assessment is accurate, let's dig into the details."
     - label: "Higher — ask more questions"
       description: "There's more complexity than identified. I'll explain what's missing."
     - label: "Lower — fewer questions needed"
       description: "This is simpler than it looks. Let's keep it focused."
       multiSelect: false

3. **Ask the confirmed number of questions using AskUserQuestion**, in rounds of 1-4, as many rounds as needed.

   **Rules:**
   - Each question has 2-4 concrete options; `multiSelect: true` only when choices aren't mutually exclusive; `header` max 12 chars
   - Exactly one option per question marked `⭐ Recommended` in its label
   - Every option's `description` follows: `[1-sentence what this does] · Strength: [key advantage] · Tradeoff: [key cost or risk]`
   - Recommendations must be grounded in the Step 2 research — cite the codebase pattern that backs them, not guesses

   **Categories** (pick what fits, scaled by complexity):
   - Universal: **Scope boundaries** (in vs out), **Edge cases / failure modes**, **Success criteria**, **Priority** (must-have vs nice-to-have)
   - MEDIUM+: **Data model decisions**, **Error handling strategy**, **Testing approach**, **Performance boundaries**
   - HIGH: **Architecture choices**, **State management**, **Security model**, **Migration & rollback**, **Observability**

   **What NOT to ask:**
   - Anything already settled in the ticket stub, the PRD, or lessons.md — re-asking erodes trust in upstream artifacts
   - Low-level implementation details you can determine from the codebase research yourself
   - Preferences that don't affect the plan's structure or success

   **CRITICAL**: Ask the full count for the confirmed level. Do not shortcut — thorough questioning prevents rework. Each question must force a real decision.

4. **If the user corrects a misunderstanding**: do NOT just accept it — spawn a verification sub-agent or read the files they mention, and only proceed once you've confirmed the facts yourself.

## Step 5: Phase Breakdown Approval

Print the proposed structure as text:

```
Here's my proposed plan structure for <ticket title>:

## Overview
[1-2 sentence summary]

## Implementation Phases:
1. [Phase name] - [what it accomplishes]
2. [Phase name] - [what it accomplishes]
```

Then AskUserQuestion:
- question: "Does this phase breakdown look right?"
  header: "Phases"
  options:
  - label: "Looks good, proceed"
    description: "Write the full plan into the ticket with these phases."
  - label: "Needs adjustment"
    description: "I'll explain what to change before you write the plan."
  - label: "Too granular"
    description: "Combine some phases — this is simpler than it looks."
  - label: "Too coarse"
    description: "Split some phases — there are hidden complexities."
    multiSelect: false

If the approved breakdown now exceeds the Step 3 caps (>3 phases / >8 files), go back and split — don't write an oversized ticket.

## Step 6: Write the Refined Ticket Body

Edit the ticket's .md file directly. Keep the frontmatter (update `updated:` to today); replace/extend the body to match this contract exactly. This is the anatomy /tc-implement and both reviews depend on:

````markdown
---
title: "Feat: s-03: User login via OAuth"
id: TC-ABCDEF
priority: P1
---
## Context
Why this exists; PRD refs (FR-NNN, US-NN); Prerequisites: TC-XXX, TC-YYY; outcome sentence.

## Requirements
What must be true when done — user-visible behavior, edge cases, error handling. Written for a dev who has never seen the repo.

## Acceptance Criteria
- [ ] checklist (tickcats-native section)

## Architecture
Mermaid diagram(s): component interaction / data flow / sequence for the touched area. Prose explaining WHERE this fits in the repo and WHY this approach.

## Current State
Research digest: key discoveries with file:line refs, patterns to follow, constraints. (Replaces research.md.)

## Implementation Plan
### Phase N: <name>
Per change: **File**: path · **Intent**: 1-2 sentences · **Contract**: signature/schema/route/invariant · illustrative code snippet when non-obvious (example, not final implementation).
### Success Criteria — Automated (commands) / Manual (steps)

## What We're NOT Doing

## Progress
- [ ] N.M <step> — (SHA appended at phase end; only checkbox section besides AC)

## Plan Review   (appended by tc-plan-review)
## Impl Review   (appended by tc-impl-review)
````

Do NOT write the `## Plan Review` / `## Impl Review` sections — those belong to the review skills.

**Section-by-section guidance:**

1. **Context** — preserve the stub's PRD refs and `Prerequisites:` line; sharpen the outcome sentence with what you now know.

2. **Requirements** — full prose, not bullets-only. State user-visible behavior, edge cases, and error handling explicitly. The test: a developer with zero repo knowledge reads this and knows exactly what "done" means.

3. **Acceptance Criteria** — refine the stub's draft into verifiable `- [ ]` items. Every requirement must be covered by at least one criterion.

4. **Architecture** — at least one mermaid diagram (```mermaid fenced block): component interaction, data flow, or sequence diagram for the touched area — whichever best explains this change; use more than one when they answer different questions. Surround with prose: where this sits in the repo, why this approach won over the alternatives discussed in the interview.

5. **Current State** — the Step 2 research digest. Key discoveries with `path/to/file.ext:line` references, existing patterns the implementer must follow, constraints to work within, reusable components. This section replaces research.md — it must stand alone.

6. **Implementation Plan** — one `### Phase N: <name>` per approved phase. Per change within a phase:
   - **File**: exact path
   - **Intent**: 1-2 sentences — what this change does and why
   - **Contract**: the interface, signature, schema field, route, file-structure delta, or invariant the change touches
   - **Snippet**: an illustrative code snippet when it clarifies the shape of the change — this is an *example* showing structure and approach, NOT a final implementation to copy-paste. Because the reader may not know the repo, lean toward including one for non-obvious changes (tricky regex, unusual API call, counterintuitive ordering, a signature other phases depend on). For routine edits (add a field, wire a handler, follow an existing pattern), Intent + Contract suffice.
   - Each phase ends with `### Success Criteria` split into **Automated** (exact commands: test, lint, typecheck, build) and **Manual** (concrete human steps to verify).

7. **What We're NOT Doing** — explicit out-of-scope list to prevent scope creep. Include anything cut during the interview and anything split into sibling tickets.

8. **Progress** — mechanical: one `- [ ] N.M <step>` line per Success Criteria item, numbered by phase (`1.1`, `1.2`, `2.1`, …). Do not rename step titles later. /tc-implement checks these off and appends the commit SHA at each phase end — leave the SHAs out now. This and Acceptance Criteria are the ONLY checkbox sections; Implementation Plan phases carry plain prose/bullets.

**No open questions in the final ticket**: if anything is unresolved, STOP and research or ask — every decision must be made before the body is written.

## Step 7: Finish — Label, Move, Announce

1. **Remove `[to refine]`** from the ticket title (frontmatter `title:` — remember state is the folder, labels are only in the title).
2. **Check prerequisites**: for every id in the `Prerequisites:` line, verify the ticket is in `.tickcats/done/` (`grep -rl "id: TC-XXX" .tickcats/done/`).
   - **All done (or no prerequisites)** → remove `[blocked]` if present, then `tickcats move <bare-filename> backlog ready`.
   - **Not all done** → keep (or add) `[blocked]` in the title and leave the ticket in `backlog/`. Name which prerequisites are outstanding.
3. **Update** the frontmatter `updated:` date.
4. **Report**:

   ```
   Ticket refined: <title>
   File: .tickcats/<ready|backlog>/<file>.md
   [If split: siblings created TC-A, TC-B; original moved to wont-do.]
   [If blocked: waiting on TC-XXX, TC-YYY.]

   Next step:
   → /tc-plan-review <ticket-id>   (validate the plan before implementing)
   → /tc-implement <ticket-id>     (start implementation — ready tickets only)
   ```

5. **Iterate on feedback** — be ready to add phases, adjust the approach, sharpen success criteria, or move scope items until the user is satisfied. All edits go into the ticket file.

## Important Guidelines

1. **The ticket is the single artifact** — never create plan.md, plan-brief.md, research.md, or anything under context/changes/. Edit the ticket in place with Edit; findings, decisions, and revisions all land in its body.

2. **Write for a stranger** — the bar for every section is a competent dev who has never opened this repo. Name paths precisely, explain conventions instead of assuming them, and let the mermaid diagrams carry the spatial understanding a repo veteran gets for free.

3. **Be skeptical** — question vague requirements, verify claims against code, don't assume. If research contradicts the stub, surface it before planning around it.

4. **Be interactive** — don't write the full body in one shot. Gates in order: split confirmation (if triggered) → complexity confirmation → interview rounds → phase approval → write.

5. **Be thorough** — read all mentioned files COMPLETELY before planning; delegate discovery to parallel sub-agents but read the load-bearing files yourself; include specific file:line references throughout Current State and the Implementation Plan.

6. **Describe intent, shape, and contract — not final code** — snippets illustrate; the implementer writes the real thing from File + Intent + Contract + the surrounding pattern.

7. **Respect the size caps** — a refined ticket predicted to exceed 8 files, 3 phases, or ~500 body lines should have been split in Step 3. If you only realize mid-write, stop and run the split mechanics rather than finishing an oversized plan.

8. **Track progress** — TaskCreate/TaskUpdate for research and writing stages so the user sees status; synthesize sub-agent output instead of accumulating it, and if context degrades mid-session, the ticket file already holds the draft — offer to resume with `/tc-plan <ticket-id>` in a fresh window.
