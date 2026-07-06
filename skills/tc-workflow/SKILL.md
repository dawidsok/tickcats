---
name: tc-workflow
description: >
  Pipeline navigator for the tc-* (tickcats) skill family. Inspects project
  state (context/foundation/, .tickcats/ board columns and labels) to show
  where you are in the idea-to-implementation pipeline and recommend the next
  tc-* skill — then offers to invoke it. Use when the user says "tc workflow",
  "which tc skill should I use", "what's next in the pipeline", "where am I in
  the tc workflow", "tc help", "how do the tc skills fit together", or is
  unsure which tc-* skill fits their situation.
argument-hint: ""
allowed-tools:
  - Read
  - Bash
  - Glob
  - Grep
  - AskUserQuestion
  - Skill
---

# tc-workflow: Where Am I, What's Next

Read-only inspection, one recommendation, optional dispatch. This skill never edits files, never moves tickets, never auto-chains — the only write-adjacent action is invoking another tc-* skill via the Skill tool AFTER explicit user consent.

## The pipeline (reference card — print it in Step 2)

```
  /tc-shape ──► /tc-prd ──► /tc-decompose ──► /tc-refine ──► /tc-plan ──► /tc-plan-review
   idea to       notes to     PRD to ticket      grill &        plan in       review plan
   shape-notes   prd.md       stubs on board     sharpen        ticket body   (optional)
                                                 (optional)          │
                                                                     ▼
                              /tc-impl-review ◄────────────── /tc-implement
                               review result                   ready ► doing ► done
                               (optional)                      per-phase commits
```

One-liners:

| Skill | Use when |
|---|---|
| `/tc-shape` | You have an idea (greenfield) or a meaningful change to an existing system (brownfield) and no `shape-notes.md` |
| `/tc-prd` | `shape-notes.md` exists; you need `context/foundation/prd.md` |
| `/tc-decompose` | `prd.md` exists; the board has no stubs for it yet |
| `/tc-refine` | Stubs carry `[to refine]`, or you have a vague pain ("auth is a mess") to crystallize into tickets |
| `/tc-plan` | A stub is worth building next; it needs a full implementation plan in its body |
| `/tc-plan-review` | A planned ticket sits in `ready/` and you want the plan challenged before code |
| `/tc-implement` | A ticket in `ready/` is planned and unblocked (or `doing/` holds unfinished work to resume) |
| `/tc-impl-review` | A ticket in `done/` has no `## Impl Review` section |

State lives in two places: `context/foundation/` (shape-notes.md, prd.md, lessons.md) and `.tickcats/` (columns are folders — `backlog/ ready/ doing/ done/ wont-do/`; labels are bracket groups at the start of the frontmatter `title:`).

## Step 1: Inspect (read-only)

Gather all of it before concluding anything:

```bash
ls context/foundation/ 2>/dev/null
ls .tickcats/ 2>/dev/null
for c in backlog ready doing done wont-do; do echo "$c: $(ls .tickcats/$c/*.md 2>/dev/null | wc -l | tr -d ' ')"; done
grep -l "to refine" .tickcats/backlog/*.md 2>/dev/null | wc -l
grep -rl "\[blocked" .tickcats/backlog/ .tickcats/ready/ 2>/dev/null
grep -rL "## Impl Review" .tickcats/done/*.md 2>/dev/null
```

Then refine what the counts alone can't tell you:

- **`doing/` non-empty** → for each, Read the ticket and find the first `- [ ]` row in `## Progress` (that's the resume point) or note all boxes checked (implementation finished but never moved — flag it).
- **`ready/` non-empty** → note the highest-priority unlabeled ticket (what `tickcats pick-next` would return; run it if the CLI is on PATH). Check whether it has a `## Plan Review` section.
- **`[blocked]` tickets** → parse each `Prerequisites:` line and resolve which prerequisite tickets are NOT in `done/`. The most-depended-on open prerequisite is the chokepoint.
- **`backlog/` stubs without `[to refine]` and without `## Implementation Plan`** → anomalies; note them (likely a label edit went wrong — labels must be ONE leading bracket group in the title).
- **No `.tickcats/` but `context/foundation/prd.md` exists** → decompose is pending.
- **No `prd.md`** → check for `shape-notes.md` (→ tc-prd) and whether the cwd looks like an existing codebase (source files present → brownfield: tc-shape for a big change, tc-refine improvement mode for targeted fixes; empty/new dir → greenfield tc-shape).

## Step 2: Report

Print the pipeline card from above with a `◄── you are here` marker on the current stage, then a status block:

```
tc-workflow status

  Foundation:  prd.md ✓ (v1, 2026-07-02) · shape-notes ✓ · lessons ✓
  Board:       backlog 4 (3 to refine, 1 blocked) · ready 1 · doing 1 · done 2 · wont-do 1

  Findings, ranked:
  1. doing/ holds TC-ABCDEF (s-02-search) at Progress 2.3 — unfinished implementation.
  2. done/ TC-XXYYZZ has no impl review.
  3. ready/ TC-GGHHII (s-03-export) is planned and unblocked — pick-next's choice.
  4. 3 backlog stubs still [to refine]; s-05 is blocked on s-03.

  Recommended: /tc-implement TC-ABCDEF  (resume — in-flight work beats new work)
```

Ranking rule — first match wins the recommendation, but LIST every finding:

1. `doing/` with unchecked Progress boxes → resume `/tc-implement <id>` (in-flight work first).
2. `doing/` with all boxes checked → `/tc-implement <id>` to run its completion steps (move to done, unblock dependents).
3. `done/` without `## Impl Review` → `/tc-impl-review <id>` — but only rank this above new work if the user's recent asks lean quality; otherwise list it and rank 4 first.
4. `ready/` non-empty with an unblocked ticket → `/tc-implement` (name pick-next's choice; mention `/tc-plan-review <id>` as the cautious alternative if it has no Plan Review section).
5. `[to refine]` stubs in backlog → `/tc-plan <highest-priority-unblocked>` (or `/tc-refine` first when stubs are thin/contradictory — say which and why).
6. `prd.md` exists, board missing or has no tickets for it → `/tc-decompose`.
7. `shape-notes.md` exists, no `prd.md` → `/tc-prd`.
8. Nothing exists → `/tc-shape` (greenfield) or `/tc-refine <pain>` (brownfield quick items — offer both).
9. Everything done, backlog empty → say so; suggest `/tc-refine` improvement mode for the next round.

If everything is `[blocked]`, the recommendation is the chokepoint: the ticket whose completion unblocks the most others — plan or implement THAT.

## Step 3: Offer dispatch (consent-gated)

One AskUserQuestion, then act:

- question: "Recommended next step: <command>. Run it now?"
  header: "Next step"
  options:
  - label: "Yes — run <command> (Recommended)"
    description: "<one line: what the skill will do to which artifact>"
  - label: "Pick a different finding"
    description: "Choose one of the other ranked findings instead."
  - label: "Just the status, thanks"
    description: "Stop here — commands are printed above to run later."
  multiSelect: false

On "Yes": invoke the recommended skill via the **Skill** tool (NOT Bash), passing the ticket id/path as args. On "Pick a different finding": re-ask with the remaining findings as options, then invoke the chosen one the same way. On "Just the status": STOP — print nothing further.

## Rules

1. **Read-only until consent.** No Edit, no Write, no `tickcats move`, no git. Inspection only.
2. **List all findings, recommend one.** Multiple pipeline states are usually true at once; hiding the others forces the user to re-run you.
3. **Ground every claim.** Name real ticket ids, real files, real Progress rows — never "some tickets need refinement".
4. **Don't re-explain what triggered fine.** If the user already knows they want to plan ticket X, they didn't need this skill — point them at `/tc-plan X` in one line and stop; skip the full card.
5. **Anomalies beat recommendations.** A malformed label group, a ticket stuck in doing/ with all boxes checked, a dangling Prerequisites id — surface these at the top; broken state makes every downstream recommendation wrong.
