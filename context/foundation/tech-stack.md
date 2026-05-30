---
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
---

## Why this stack

TickCats is a local CLI/TUI app whose core work is filesystem operations, markdown/YAML ticket parsing, keyboard-first terminal UI, and single-binary distribution. Go is the recommended starter for a CLI in this language family: it is typed, conventional, well documented, agent-friendly, and easier to maintain than Rust for this project given the user's experience. The hand-off records `self-host` as the registry-compatible distribution target, while the intended release channels are GitHub Releases first, with Homebrew and npm-style distribution as follow-up packaging paths. CI uses GitHub Actions with manual promotion so checks can run automatically while published releases remain deliberate.
