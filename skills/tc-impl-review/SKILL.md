---
name: tc-impl-review
description: Tickcats-pipeline variant of 10x-impl-review. Review a ticket's implementation against the plan in its body — drift, dangerous decisions, pattern compliance. Use when the user says "tc impl review", "review this ticket's implementation", "review implementation of TC-XXXXXX". Use AFTER /tc-implement. For the classic context/changes workflow use /10x-impl-review instead.
argument-hint: "[ticket-id-or-path]"
allowed-tools:
  - Read
  - Glob
  - Grep
  - Bash
  - Agent
  - AskUserQuestion
  - Edit
  - TaskCreate
  - TaskUpdate
  - TaskList
  - TaskGet
---

# Implementation Review (tickcats)

Compare actual implementation work against the plan living in a tickcats ticket's body to catch drift, dangerous decisions, architecture violations, and pattern misuse before they compound.

Tickcats facts this skill relies on:
- Columns = folders: `.tickcats/{backlog,ready,doing,done,wont-do}/`. A ticket's state IS its folder — never a frontmatter field.
- Frontmatter: `title`, `id` (TC-XXXXXX), `priority` (P0–P3), `created`, `updated`.
- Title format: `Feat|Task: <change-id>: <title>` (labels like `[blocked]` in leading brackets).
- The ticket body is the full spec: `## Implementation Plan` (phases with Success Criteria) and `## Progress` (`- [x] N.M <step> — <sha>` rows written by /tc-implement) are the plan under review.

Two granularities:
- **Full review**: ticket in `done/` — comprehensive sweep of all phases
- **Mid-flight review**: ticket in `doing/` — only phases whose Progress rows are fully `[x]`

Two modes:
- **Fresh review**: analyze → findings → interactive triage
- **Resume triage**: the ticket's latest `## Impl Review` subsection has `Decision: PENDING` findings → jump to per-issue triage

## Input resolution

1. Argument is a path to a ticket file → use it directly.
2. Argument is a `TC-XXXXXX` id → `grep -rl "id: TC-XXXXXX" .tickcats/done/ .tickcats/doing/`.
3. No argument → list `.tickcats/done/*.md` (title + id from frontmatter) and ask via AskUserQuestion which to review. If `done/` is empty, also offer `doing/` tickets.

Refusals:
- Ticket in `backlog/` or `ready/` → "Nothing implemented yet — run /tc-implement first." STOP.
- Ticket in `wont-do/` → "This ticket was split or rejected; review its siblings instead." STOP.

If the ticket already has a `## Impl Review` section whose most recent `### Review` subsection contains `Decision: PENDING` findings, ask: resume triage of those findings (skip to Step 5) or run a fresh review.

## Step 1: Load ticket and detect change scope

TaskCreate: "Implementation Review" / activeForm "Loading context"

1. **Read the ticket file fully** — no limit/offset. The plan under review = `## Requirements`, `## Architecture`, `## Implementation Plan` (per-phase File/Intent/Contract entries + Success Criteria), `## What We're NOT Doing`, `## Progress`.
2. **Read `context/foundation/lessons.md` if present** and use accepted rules as priors when scanning for findings — a deviation that violates a known recurring rule is a stronger signal than a generic style nit.
3. **Read canonical state from `## Progress`**: completion = `count([x]) / count([ ] + [x])`; current phase = phase containing the first `- [ ]` (or last phase if all done).
4. **Scope**: ticket in `done/` → all phases; ticket in `doing/` → only phases whose Progress rows are fully `[x]` (note the mid-flight scope in the report).
5. **Extract** from phases under review: file paths from the plan entries, architectural decisions, success criteria (Automated/Manual bullets + their `[ ]`/`[x]` mirror in Progress), and the "What We're NOT Doing" list (scope guardrails).
6. **Git scope detection** — what actually changed. Collect the commit SHAs from the reviewed Progress rows (`- [x] N.M <step> — <sha>`):
   ```bash
   FIRST_SHA="<earliest sha in reviewed phases>"
   LAST_SHA="<latest sha in reviewed phases>"
   git diff --name-only ${FIRST_SHA}^..${LAST_SHA}
   git log --oneline ${FIRST_SHA}^..${LAST_SHA}
   ```
   If Progress rows carry no SHAs, fall back to the ticket's frontmatter dates:
   ```bash
   git log --oneline --after="<created>" --until="<updated> 23:59" -- .
   ```
   and if that range is still ambiguous, use commits whose messages reference the `<change-id>` from the ticket title (tc-implement commits as `<type>(<change-id>): …`).

Compare changed-file list against plan-file list:
- **In plan AND in diff** → expected change, verify content matches intent
- **In diff but NOT in plan** → unplanned change, investigate and flag
- **In plan but NOT in diff** → potentially missing implementation

Exclude the ticket file itself from the "unplanned change" bucket — tc-implement stages it at every phase commit by design.

Don't pre-read every changed file into the main context — let the sub-agents read what they need. Main context should carry the ticket body and the diff summary, not the full source of 20 files.

## Step 2: Parallel review via sub-agents

TaskUpdate: activeForm "Gathering evidence"

Launch **two** sub-agents simultaneously. Each gets targeted context — don't dump the full ticket into both.

**Agent 1 — Plan Drift Detection** (`subagent_type: "general-purpose"`)

Give it: the `## Implementation Plan` text for the reviewed phases, the list of file paths to read.

Instructions: for each planned change, read the actual file and verify implementation matches intent. Check for:
- Changes implemented differently than planned (intent mismatch, not formatting)
- Planned items skipped without documentation
- Additions not described in the plan (scope creep)

Report each: file path, what the plan said, what exists, verdict (MATCH / DRIFT / MISSING / EXTRA).

**Agent 2 — Safety, Quality & Pattern Compliance** (`subagent_type: "general-purpose"`)

Give it: the full list of changed files to read, the project root path.

Instructions:

1. **Safety & quality scan** on each changed file. Flag:
   - **Security**: injection risks (SQL, command, XSS), hardcoded secrets, missing authn/authz at system boundaries, overly permissive CORS/permissions.
   - **Performance**: N+1 queries, unbounded iteration/recursion, missing pagination, unnecessary sync I/O.
   - **Reliability**: missing error handling at external boundaries (API calls, file I/O, DB), race conditions, resource leaks.
   - **Data safety**: destructive DB ops without rollback, schema changes without migration path, data loss potential.

2. **Pattern compliance** — for each changed file, find 1–2 similar existing files and compare naming, error handling approach, module structure, imports/exports, test structure, config patterns. **Only report substantive mismatches** (e.g., a new module uses camelCase where siblings use snake_case; a new endpoint skips the auth middleware pattern the rest of the API uses). Skip trivial style differences — if the code works and follows the plan, minor formatting is not a finding.

3. **Budget pattern work to scope** — if the diff changed ≤3 files, spend minimal time on patterns. Scale pattern depth with change scope.

Report each finding with: file, line number, category, severity (CRITICAL / WARNING / OBSERVATION), description, recommendation.

## Step 3: Verify success criteria

TaskUpdate: activeForm "Verifying success criteria"

For each reviewed phase:

**Automated**: run each command from the phase's "Success Criteria — Automated" list with Bash. Record command, pass/fail, actual output (truncate if huge).

**Manual**: in `## Progress`, check Manual items as `- [x]` vs `- [ ]`. Flag items marked complete that lack observable evidence in the diff (possible rubber-stamping); acknowledge unchecked items as pending.

Also cross-check `## Acceptance Criteria`: any unchecked AC on a `done/` ticket is a Success Criteria finding.

## Step 4: Compile findings and present report

TaskUpdate: activeForm "Compiling findings"

Each finding has:
- **ID**: F1, F2, F3…
- **Severity**: CRITICAL / WARNING / OBSERVATION (how bad if ignored)
- **Impact**: LOW / MEDIUM / HIGH (how much focus the decision needs)
- **Dimension**: Plan Adherence / Scope Discipline / Safety & Quality / Architecture / Pattern Consistency / Success Criteria
- **Title**: one line
- **Location**: `file:line` (or "N/A" for missing items)
- **Detail**: what's wrong with evidence — plan vs. actual, or code vs. expected
- **Fix options**: 1 or 2 (see below)

### Impact

Orthogonal to severity. A CRITICAL with LOW impact (obvious one-line fix) is cheap; a WARNING with HIGH impact (architectural rework) deserves careful thought.

| Impact | Meaning |
|---|---|
| 🏃 **LOW** | Quick decision. Fix is obvious and narrowly scoped. Safe to batch. |
| 🔎 **MEDIUM** | Worth pausing. Real tradeoff or non-trivial edit — think before deciding. |
| 🔬 **HIGH** | Architectural stakes. Wide blast radius, strategic implications, or unclear best path. |

### Fix options

Default to **one** fix. Only offer two when there's a genuine tradeoff a smart reviewer would want to weigh (e.g. "patch the call site" vs. "fix it at the source"). If you find yourself inventing a weak second option, don't — present one and move on.

**LOW-impact findings**: just `Fix: [one line]`.

**MEDIUM/HIGH-impact findings**: each option gets:
```
[1-sentence approach] · Strength: [advantage, ideally grounded in code/plan evidence] · Tradeoff: [cost or risk] · Confidence: HIGH|MED|LOW — [1-line why] · Blind spot: [what we haven't verified, or "None significant"]
```

When offering two options, mark exactly one `⭐ Recommended`.

### Dimension verdicts

PASS / WARNING / FAIL per dimension:
- **Plan Adherence** — planned changes implemented as described? FAIL on MISSING or major DRIFT.
- **Scope Discipline** — "What We're NOT Doing" boundaries respected? WARNING if EXTRA changes exist but are benign.
- **Safety & Quality** — security, performance, reliability, data safety. FAIL on any CRITICAL finding.
- **Architecture** — module boundaries, dependency direction, abstraction justification. FAIL on violations.
- **Pattern Consistency** — follows existing conventions. WARNING on minor inconsistencies.
- **Success Criteria** — automated checks pass, manual checks and AC addressed. FAIL on automated failures.

### Overall verdict

- **APPROVED** — all PASS, or PASS with ≤2 minor warnings
- **NEEDS ATTENTION** — multiple warnings or 1 non-critical FAIL
- **REJECTED** — any critical FAIL (security, major drift, data safety, failing tests)

Sort findings by severity: CRITICAL → WARNING → OBSERVATION. Cap at 10 — consolidate related findings if more.

### Report format

Plain text, box-drawing. PASS dimensions appear only in the verdicts table, never as findings. Omit severity groups with zero findings.

```
═══════════════════════════════════════════════════════════
  IMPLEMENTATION REVIEW: [Ticket title]
  Ticket: TC-XXXXXX (done/)  |  Scope: Phases 1–3 of 3  |  Date: YYYY-MM-DD
  Findings: [N critical] [N warnings] [N observations]
═══════════════════════════════════════════════════════════

  Plan Adherence        PASS    ✅
  Scope Discipline      WARNING ⚠️   (1 finding)
  Safety & Quality      FAIL    ❌   (1 finding)
  Architecture          PASS    ✅
  Pattern Consistency   PASS    ✅
  Success Criteria      PASS    ✅

  ► Overall: NEEDS ATTENTION

═══════════════════════════════════════════════════════════
  CRITICAL FINDINGS ❌
═══════════════════════════════════════════════════════════

  F1 — SQL injection in auth handler
  ╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌
    Severity:  ❌ CRITICAL
    Impact:    🔎 MEDIUM — real tradeoff; pause to reason through it
    Dimension: Safety & Quality
    Location:  src/auth/handler.ts:42

    Detail:
    SQL query built with string concatenation. Plan specified
    parameterized queries but implementation uses template literals.

    Fix: Replace the template literal with a parameterized query using
         db.query($1, [value]).
      Strength:   Matches the pattern in src/users/query.ts and removes
                  the injection class entirely.
      Tradeoff:   Minor — one call site, a few-line change.
      Confidence: HIGH — identical pattern used elsewhere in this repo.
      Blind spot: None significant.

═══════════════════════════════════════════════════════════
```

Formatting rules:
- The **finding title line** holds only the ID and the short title. Everything else goes below as labeled fields on their own lines (Severity, Impact, Dimension, Location, then `Detail:`).
- **Always pair icons with a word** — `❌ CRITICAL`, never a bare icon.
- **Impact always carries its one-line meaning** (copy from the Impact table) so LOW/MEDIUM/HIGH is self-explanatory at the point of use.

After the report, ask:

```
question: "Review complete. How would you like to proceed?"
header: "Implementation Review — [N] findings"
options:
  - label: "Triage findings"
    description: "Walk through each finding and decide."
  - label: "Record & triage later"
    description: "Append the review to the ticket. Resume with /tc-impl-review <ticket>."
  - label: "Record only"
    description: "Append and finish — I'll handle the findings myself."
multiSelect: false
```

### Recording the review in the ticket

There is no separate review file. Append to the ticket body:

1. If the ticket has no `## Impl Review` section, add one at the end of the body.
2. Under it, always append a dated subsection `### Review <YYYY-MM-DD>` containing: scope, verdict, findings count, the dimension table, and each finding as `#### F1 — <title>` with Severity / Impact / Dimension / Location / Detail / Fix option(s) / `- **Decision**: PENDING` lines (same field structure as the terminal report, in markdown list form).
3. Update the ticket frontmatter `updated:` to today.

The `Decision: PENDING` lines enable resume mode: re-running `/tc-impl-review <ticket>` detects them and offers to resume triage.

### On REJECTED

If the overall verdict is REJECTED and the ticket sits in `done/`, ask via AskUserQuestion:

```
question: "Verdict is REJECTED. Move the ticket back to doing/ so /tc-implement can address the findings?"
header: "Ticket state"
options:
  - label: "Move to doing"
    description: "Runs: tickcats move <bare-filename> done doing"
  - label: "Leave in done"
    description: "Keep the column; findings stay recorded in the ticket."
multiSelect: false
```

On "Move to doing", run `tickcats move <bare-filename> done doing` (remember: the folder is the state — never edit frontmatter to change state).

"Record & triage later" → append, print the ticket path, remind them to run `/tc-impl-review <ticket-id>`.
"Triage" → proceed to Step 5.

## Step 5: Interactive triage

TaskUpdate: activeForm "Triage"

### Resume mode

If entered via a ticket with recorded findings: read the latest `### Review` subsection, parse `#### F` headers, filter to `Decision: PENDING`. If none: "All findings triaged." Done.

### Triage loop

Walk findings in severity order (CRITICAL → WARNING → OBSERVATION). For each, ask via AskUserQuestion — question body carries the full finding (title, Severity, Impact + meaning, Dimension, Location, Detail, Fix block(s)); header `Finding [current] of [total remaining]`:

- **With 2 fix options**: `Apply Fix A ⭐` / `Apply Fix B` / `Skip` / `Record as lesson`
- **With 1 fix option**: `Fix now` / `Fix differently` (different approach — let's discuss) / `Skip` / `Record as lesson` (save as a recurring project rule via /10x-lesson)

**Handling responses:**
- **Apply Fix A/B / Fix now**: show the exact before/after code change. Brief confirmation ("Apply this?"), then edit. Mark FIXED (record which option, e.g. "Fixed via Fix A").
- **Fix differently**: ask the preferred approach, apply, mark FIXED.
- **Record as lesson**: pre-fill four lessons-entry fields directly from the finding — `Context` from the finding's Location, `Problem` from the finding's Detail, `Rule` and `Applies to` left as empty placeholders for the user to fill. Show the proposed entry as a complete markdown block and ask the user to edit / confirm via AskUserQuestion ("Approve this entry?" / "Edit before saving" / "Cancel"). On confirm, append the entry as a new H2 section to `context/foundation/lessons.md` — if the file does not exist, create it first with this canonical 5-line header (no separate template file; the header is embedded inline here):

  ```
  # Lessons Learned

  > Append-only register of recurring rules and patterns. Re-read at start by /10x-frame, /10x-research, /10x-plan, /10x-plan-review, /10x-implement, /10x-impl-review.

  ```

  The pre-fill-then-confirm flow is the load-bearing UX detail; the user must see the full proposed entry with the pre-filled Context/Problem and have a chance to edit Rule and Applies-to before append. After the append succeeds, **always** ask a follow-up via AskUserQuestion: "Lesson saved. Also apply the fix to the current code?" with options "Yes — fix now" / "No — lesson only". **Never skip this question or decide on the user's behalf** — whether the fix is trivial, out of scope, or spans many files, the decision belongs to the user. If yes: show the before/after code change, apply, mark `FIXED + ACCEPTED-AS-RULE: <rule title>`. If no: mark `ACCEPTED-AS-RULE: <rule title>` (finding stays unfixed, rule is recorded for future work).
- **Skip** → SKIPPED. Move on, don't argue.
- **Other (free text)**: interpret the user's intent. Common intents: "fix differently" → ask the preferred approach, apply, mark FIXED; "accept risk" → mark ACCEPTED with the user's justification; "dismiss"/"disagree" → mark DISMISSED.

After each decision, update the finding's `Decision:` line in the ticket's `## Impl Review` section.

### Summary

```
═══════════════════════════════════════════════════════════
  TRIAGE COMPLETE
═══════════════════════════════════════════════════════════

  Fixed:     F1, F2 (Fix A)   (2)
  Rule:      F3 (+ fixed)     (1)
  Skipped:   F4               (1)
  Accepted:  F5               (1)

═══════════════════════════════════════════════════════════
```

Ensure all `Decision:` lines in the ticket reflect final outcomes; bump frontmatter `updated:`. Mark the review task completed.

## Notes

- This is a **review** skill. Default to analyzing and reporting — only make edits during triage when the user explicitly chooses "Apply Fix" or "Fix differently" for a specific finding. The one exception: appending the `## Impl Review` section to the ticket is always allowed.
- Be specific. "src/auth/handler.ts:42 — SQL query built with string concatenation, vulnerable to injection" — not "there might be a security issue somewhere".
- Don't flag style preferences unless they matter. If the code works and follows the plan, minor style differences from existing code are observations, not warnings.
- If the plan itself was flawed (e.g., the ticket planned an insecure approach), flag it — this review catches plan issues too. That's a finding here, not a reason to re-run /tc-plan-review.
- Impact is about *decision effort*, not *severity*. LOW impact on a CRITICAL finding means the fix is obvious; HIGH impact on a WARNING means the tradeoff is real.
- Two fix options only when there's a genuine tradeoff. Don't invent alternatives for trivial fixes.
- On a mid-flight (`doing/`) review, still check whether reviewed phases broke assumptions of earlier phases. Phases interact.
- During triage, keep momentum. User already read the report.
- When fixing, minimal targeted edits. Don't refactor surrounding code or "improve" things that weren't flagged.
- Fixes applied during triage are working-tree edits — leave committing to the user (or the next /tc-implement pass); never commit unless asked.
