# Installation Investigation

## Binary naming

The Go package lives at `cmd/tickcats/`. `go build` and `go install` both derive the
binary name from the directory, so the executable is always named `tickcats` (or
`tickcats.exe` on Windows) with no alias or rename step required.

## Provider matrix

| Provider | Binary name | Config needed |
|---|---|---|
| `go install` | `tickcats` ✓ | none |
| GitHub Releases (goreleaser) | `tickcats` ✓ | `.goreleaser.yml` + GH Actions workflow |
| Homebrew tap (via goreleaser) | `tickcats` ✓ | tap repo + goreleaser `brews:` block |
| Scoop (Windows) | `tickcats.exe` ✓ | scoop manifest JSON |

## v1 recommendation

**Phase 1 — ship now, zero infra:**

```
go install github.com/dawidsok/tickcats/cmd/tickcats@latest
```

Works immediately for Go users. Binary lands in `$GOPATH/bin/tickcats`.
Requires no CI, no release workflow, no external accounts.

**Phase 2 — widen reach:**

Add GoReleaser (`.goreleaser.yml`) + a GitHub Actions release workflow triggered on
`v*` tags. GoReleaser produces:
- pre-built tarballs for macOS/Linux/Windows uploaded to GitHub Releases
- a Homebrew tap formula auto-generated in a separate `homebrew-tickcats` repo

This covers non-Go macOS/Linux users via `brew install <tap>/tickcats` and
anyone who prefers a direct download.

No hosted services, auth, or sync are introduced by either phase.
