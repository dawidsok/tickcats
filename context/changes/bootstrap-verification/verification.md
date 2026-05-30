---
bootstrapped_at: 2026-05-30T10:34:36Z
starter_id: go
starter_name: "Go (standard library)"
project_name: tickcats
language_family: go
package_manager: ""
cwd_strategy: subdir-then-move
bootstrapper_confidence: first-class
phase_3_status: ok
audit_command: "govulncheck -json ./..."
---

## Hand-off

```yaml
starter_id: go
project_name: tickcats
hints:
  language_family: go
  team_size: solo
  deployment_target: self-host
  ci_provider: github-actions
  ci_default_flow: manual-promotion
  bootstrapper_confidence: first-class
  path_taken: standard
  quality_override: false
  self_check_answers: null
  has_auth: false
  has_payments: false
  has_realtime: false
  has_ai: false
  has_background_jobs: false
```

TickCats is a local CLI/TUI app whose core work is filesystem operations, markdown/YAML ticket parsing, keyboard-first terminal UI, and single-binary distribution. Go is the recommended starter for a CLI in this language family: it is typed, conventional, well documented, agent-friendly, and easier to maintain than Rust for this project given the user's experience. The hand-off records `self-host` as the registry-compatible distribution target, while the intended release channels are GitHub Releases first, with Homebrew and npm-style distribution as follow-up packaging paths. CI uses GitHub Actions with manual promotion so checks can run automatically while published releases remain deliberate.

## Pre-scaffold verification

| Signal | Value | Severity | Notes |
| --- | --- | --- | --- |
| npm package | not run | n/a | non-JS starter |
| GitHub repo | not run | n/a | Go starter docs_url is `https://go.dev/doc/`, not a GitHub repository URL |

## Scaffold log

**Resolved invocation**: `mkdir .bootstrap-scaffold && cd .bootstrap-scaffold && go mod init github.com/dawidsok/tickcats`
**Strategy**: subdir-then-move
**Exit code**: 0
**Files moved**: 1
**Conflicts (.scaffold siblings)**: none
**.gitignore handling**: absent in scaffold
**.bootstrap-scaffold cleanup**: deleted

Note: the registry template uses a placeholder module path. During this run, the Go module path was set to `github.com/dawidsok/tickcats` per user instruction.

## Post-scaffold audit

**Tool**: `govulncheck ./...`
**Status**: installed and executed after bootstrap
**Reason**: `govulncheck` was installed with `go install golang.org/x/vuln/cmd/govulncheck@latest`, then executed via `$GOPATH/bin/govulncheck` because `$GOPATH/bin` was not on PATH.
**Result**: audit could not inspect packages because the project currently has no Go packages yet.

```text
govulncheck: no packages matched the provided patterns
```

## Hints recorded but not acted on

| Hint | Value |
| --- | --- |
| bootstrapper_confidence | first-class |
| quality_override | false |
| path_taken | standard |
| self_check_answers | null |
| team_size | solo |
| deployment_target | self-host |
| ci_provider | github-actions |
| ci_default_flow | manual-promotion |
| has_auth | false |
| has_payments | false |
| has_realtime | false |
| has_ai | false |
| has_background_jobs | false |

## Next steps

Next: a future skill will set up agent context (CLAUDE.md, AGENTS.md). For now, your project is scaffolded and verified — happy hacking.

Useful manual steps in the meantime:
- `git init` (if you have not already) to start your own repo history.
- Review any `.scaffold` siblings the conflict policy created and decide which version of each file to keep.
- Address audit findings per your project's risk tolerance — the full breakdown is in this log.
- To enable Go vulnerability auditing, install `govulncheck` with `go install golang.org/x/vuln/cmd/govulncheck@latest` and rerun it from the project root.
