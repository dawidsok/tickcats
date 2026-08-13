# Migration contract fixtures

`manifest.tsv` maps every approved feature, persisted-data contract, and known-defect decision to a fixture and the phase that will make its check executable. `fixture-cases.tsv` inventories the concrete edge cases frozen in the corpus so broad manifest rows cannot claim coverage from a generic board alone.

`scenarios.tsv` drives `scripts/compare-go-rust.sh`:

- `parity` compares Go and Rust stdout, stderr, exit status, and copied fixture trees.
- `intent` compares Rust with `expected/<scenario>/` because an approved contract intentionally differs from Go.
- `readonly` scenarios must leave their copied fixture tree byte-identical.

Phase 1 validates the manifest and the harness itself. Later phases turn the mapped contracts into product assertions. `--allow-gaps` accepts exactly the names in `expected-gaps.txt`: unexpected mismatches and stale entries both fail CI, so the allowlist shrinks as Rust contracts land.
