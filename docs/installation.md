# Installation

## Homebrew (macOS and Linux)

```sh
brew tap dawidsok/tap
brew install tickcats
```

Homebrew automatically installs shell completions for bash, zsh, and fish.

## Direct download

1. Go to [GitHub Releases](https://github.com/dawidsok/tickcats/releases).
2. Download the archive matching your platform:
   - `tickcats_<version>_darwin_arm64.tar.gz` — macOS (Apple Silicon)
   - `tickcats_<version>_darwin_amd64.tar.gz` — macOS (Intel)
   - `tickcats_<version>_linux_amd64.tar.gz` — Linux (x86-64)
   - `tickcats_<version>_linux_arm64.tar.gz` — Linux (ARM64)
   - `tickcats_<version>_windows_amd64.zip` — Windows (x86-64)
3. Extract and move the `tickcats` binary to a directory on your `$PATH`.
4. Optionally install shell completions from the `completions/` folder inside
   the archive (see below).

## Shell completions (non-Homebrew installs)

```sh
# bash
source completions/tickcats.bash

# zsh — copy into a directory in $fpath and restart your shell
mkdir -p ~/.zsh/completions
cp completions/_tickcats.zsh ~/.zsh/completions/_tickcats

# fish
mkdir -p ~/.config/fish/completions
cp completions/tickcats.fish ~/.config/fish/completions/tickcats.fish
```

The completion scripts call `tickcats __complete tickets` and
`tickcats __complete columns` to return live candidates from the local board.

## Verifying the download

Each release includes a `checksums.txt` file with SHA-256 hashes of every
archive. Verify with:

```sh
sha256sum --check checksums.txt
```

(on macOS: `shasum -a 256 --check checksums.txt`)
