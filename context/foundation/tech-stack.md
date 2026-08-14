---
starter_id: rust
project_name: tickcats
hints:
  language_family: rust
  team_size: solo
  deployment_target: self-host
  ci_provider: github-actions
  ci_default_flow: manual-promotion
  bootstrapper_confidence: first-class
  path_taken: migration-from-go
  quality_override: false
  self_check_answers: null
  has_auth: false
  has_payments: false
  has_realtime: false
  has_ai: false
  has_background_jobs: false
---

## Why this stack

TickCats is a local CLI/TUI app whose core work is filesystem operations, markdown/YAML ticket parsing, keyboard-first terminal UI, and single-binary distribution. The implementation was migrated from Go to Rust to give a stable, single-binary distribution with no runtime requirement. Rust's safety guarantees and the Ratatui ecosystem make it well-suited for the terminal-first, offline product model. The hand-off records `self-host` as the registry-compatible distribution target, while the intended release channels are GitHub Releases first, with Homebrew as the primary package manager integration.
