# Git Syncer

A daemon that keeps multiple local git repositories in sync with their remotes across machines. Designed for syncing notes, dotfiles, and personal documents.

## How it works

- Watches your repos for file changes using filesystem notifications (inotify on Linux, FSEvents on Mac)
- Debounces changes — waits for a quiet period before committing (default: 60s)
- Auto-commits, rebases from remote, and pushes
- Periodically polls remote for new changes (default: 300s)
- Manages multiple repos from a single config file

## Installation

```sh
go install github.com/minhajuddin/git-syncer@latest
```

Or build from source:

```sh
git clone https://github.com/minhajuddin/git-syncer.git
cd git-syncer
go build -o git-syncer .
```

## Configuration

Create `~/.config/git-syncer/config.toml`:

```toml
[defaults]
debounce_seconds = 60
poll_interval_seconds = 300

[[repos]]
path = "~/notes"
remote = "origin"
branch = "main"

[[repos]]
path = "~/dotfiles"
remote = "origin"
branch = "master"
debounce_seconds = 30  # override default
```

## Usage

```sh
# Start the daemon (backgrounds itself)
git-syncer start

# Start with verbose logging
git-syncer start --verbose

# Start with a custom config file
git-syncer start --config /path/to/config.toml

# Check if the daemon is running
git-syncer status

# Stop the daemon
git-syncer stop

# Run a one-shot sync for all repos (no daemon)
git-syncer sync --verbose
```

## Conflict handling

When pulling from remote, git-syncer uses `git pull --rebase`. If a rebase conflict occurs, it aborts the rebase and logs the error. You'll need to resolve conflicts manually.
