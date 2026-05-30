# Repository Guidelines

TickCats is a Go CLI/TUI for local, keyboard-first scrum ticket management. Treat @context/foundation/prd.md as the product contract and @context/foundation/tech-stack.md as the stack hand-off; do not expand v1 beyond those files.

## Hard Rules

- Keep v1 private/local: `.tickcats/` is user data and should be git-ignored by the app’s init flow.
- Do not add GitHub Issues sync, collaboration, metrics, cross-project dashboards, auth, or AI features in v1; these are explicit non-goals in @context/foundation/prd.md.
- Status comes from folders, not ticket frontmatter: `.tickcats/backlog/`, `.tickcats/ready/`, `.tickcats/doing/`, `.tickcats/done/`.
- Preserve the pick-next rule exactly: highest-priority ticket in `.tickcats/ready/` with non-empty title and Acceptance Criteria, excluding titles labeled `[blocked]` or `[to refine]`.

## Project Structure

- `go.mod` defines module `github.com/dawidsok/tickcats` and Go 1.26.2.
- `context/foundation/` holds planning docs: PRD, concept, and tech-stack decisions.
- `context/changes/bootstrap-verification/verification.md` records bootstrap/audit history.
- Source layout is not established yet. When adding code, prefer conventional Go packages and keep command entrypoints under `cmd/` if multiple binaries appear.

## Build, Test, and Verification Commands

- `go test ./...` — run all Go tests once packages exist.
- `gofmt -w <files>` — format edited Go files before committing.
- `go vet ./...` — run static checks once code exists.
- `govulncheck ./...` — run vulnerability analysis; install with `go install golang.org/x/vuln/cmd/govulncheck@latest` if missing.

## Coding Conventions

Use idiomatic Go: small packages, explicit errors, table-driven tests for domain rules, and no framework until the need is visible. Keep filesystem operations behind narrow functions so ticket storage rules are testable without a real repo.

## Testing Guidelines

Prioritize tests around markdown/frontmatter parsing, conventional title parsing (`Feat:`, `Task:`, `Bug:`), title labels (`[blocked]`, `[to refine]`), folder-state transitions, readiness checks, and priority ordering. Use fixtures for `.tickcats/` trees; avoid tests that depend on the developer’s real home directory or network.

## Commit & PR Guidelines

Current history only has `Initial commit`, so no commit convention is established. Use concise imperative subjects until a convention is added. PRs should mention affected PRD requirements by FR/US number where practical.
