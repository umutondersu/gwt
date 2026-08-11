# gwt: Git Worktree Manager

`gwt` is a fast, portable Git worktree orchestrator that wraps Git and Tmux.

This is the Go port of the original `gwt` Fish shell function, which used to
live at `~/.config/fish/functions/gwt.fish` in my
[dotfiles](https://github.com/umutondersu/dotfiles).
Where the Fish version cobbles together `git` plumbing, `tmux`, and `fzf` with
string parsing, `gwt` ships as a single static binary with native fuzzy-finding
(`go-fuzzyfinder`), proper argument parsing (Cobra), and a testable core.

## Why the port?

- **Portable** – a static binary with no shell, `fzf`, or Nix dependencies at
  runtime; works in any UNIX-like environment.
- **No tmux guessing** – connects to the exact `tmux` session per worktree and
  handles edge cases the Fish version papered over (deleted cwd sessions,
  origin-tracking branches, dirty-worktree guards).
- **Testable** – the path/tmux logic lives in a pure `core` package covered by
  Ginkgo specs, plus end-to-end CLI tests that never touch your real tmux.

## Requirements

- Go 1.26+ to build from source (or a Nix flake)
- `git` ≥ 2.13 (worktrees)
- `tmux` ≥ 3.0 (session management)

## Install

```sh
go install github.com/umutondersu/gwt@latest
```

Or build locally with the Makefile:

```sh
make build        # produces ./gwt
make install      # installs to GOPATH/bin
```

Or via the Nix flake (add it as an input to your own flake, or):

```sh
nix run .#gwt
```

## Usage

Run inside a Git repository. Worktrees live in `worktree/<name>` and are
auto-ignored (`gwt init`); each one gets its own tmux session named
`<repo>/<name>`.

| Command | Description |
|---|---|
| `gwt init` | Ensure the `worktree/` dir inside the repo is git-ignored |
| `gwt add <name>` | Create a worktree and its tmux session |
| `gwt add` | Fetch origin and pick a remote branch with the fuzzy-finder |
| `gwt add <name> -b <branch>` | Create the worktree on a specific branch |
| `gwt add <name> -f <ref>` | Create the branch from a start point |
| `gwt ls` | List worktrees and their branches |
| `gwt pick [<name>]` | Connect the current tmux session to a worktree |
| `gwt rm <name>...` | Remove worktrees (deletes merged branches; `-B` keeps, `-f` forces dirty) |
| `gwt rm .` | Refuses to remove the main worktree |
| `gwt mv <old> <new>` | Move the worktree dir; renames the branch (`-B` keeps it) |

## Development

```sh
make check       # gofmt + go mod tidy + go vet + golangci-lint + go test
make test        # run tests only
make run ARGS="ls"
```

Tests use a hermetic tmux shim on an isolated socket, so `make test` never
creates or touches sessions on your real tmux server.
