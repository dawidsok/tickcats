# Release Automation Verification

Date: 2026-06-01
Ticket: `.tickcats/doing/tc-kc8c8n-task-verify-goreleaser-config-and-github-actions-release-workflow.md`

## Scope

Verified the existing GoReleaser + GitHub Actions release path for TickCats public distribution through GitHub Releases, Homebrew tap, and `go install`.

## Commands run

```sh
go install github.com/goreleaser/goreleaser/v2@v2.15.2
goreleaser check
goreleaser release --snapshot --clean --skip=publish
go test ./...
go vet ./...
go build -o tickcats ./cmd/tickcats
```

## Results

### GoReleaser config

`goreleaser check` passed with GoReleaser `v2.15.2`.

GoReleaser reports that `brews` is being phased out in favor of `homebrew_casks`, but the configuration is valid for the pinned release version and still generates the expected Homebrew formula.

### GitHub Actions release workflow

Updated `.github/workflows/release.yml`:

- Tag trigger narrowed from all `v*` tags to semantic-version-shaped tags: `v[0-9]*.[0-9]*.[0-9]*`.
- GoReleaser action remains `goreleaser/goreleaser-action@v6`.
- GoReleaser version pinned to `v2.15.2` instead of `latest` to avoid surprise release breaks from future GoReleaser deprecations.
- Go version still comes from `go.mod` via `actions/setup-go@v5`.

### Snapshot artifacts

Local snapshot release succeeded and generated:

- `tickcats_<version>_darwin_amd64.tar.gz`
- `tickcats_<version>_darwin_arm64.tar.gz`
- `tickcats_<version>_linux_amd64.tar.gz`
- `tickcats_<version>_linux_arm64.tar.gz`
- `tickcats_<version>_windows_amd64.zip`
- `checksums.txt`
- `homebrew/tickcats.rb`

Windows ARM64 remains intentionally ignored.

### Homebrew formula

The generated formula:

- Targets `dawidsok/homebrew-tap` via `.goreleaser.yml`.
- Installs `tickcats` with `bin.install "tickcats"`.
- Includes a formula test that runs `tickcats --path <tmp>/.tickcats init` and checks the backlog directory.

### Docs

Updated installation docs to match verified release behavior:

- GitHub Actions release tags are semantic-version-shaped tags such as `v0.1.4`.
- GoReleaser publishes archive names of the form `tickcats_<version>_<os>_<arch>`.
- Homebrew tap is `dawidsok/homebrew-tap` / `dawidsok/tap/tickcats`.

## Notes / future cleanup

- GoReleaser `brews` is valid in `v2.15.2` but deprecated in newer GoReleaser versions. A future packaging ticket can evaluate switching from formula generation to `homebrew_casks` if desired.
