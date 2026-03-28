# autocommit

A minimal terminal UI for staging files and writing conventional commits.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/license-MIT-green)

## What it does

`autocommit` gives you an interactive TUI for reviewing diffs, staging files, and writing properly formatted [conventional commits](https://www.conventionalcommits.org/). No more `git add -p` gymnastics or forgetting what type of commit you're making. `autocommit` also support multiple LLM providers to aid in generating commit messages (as of now: Claude, Codex, Ollama)

```
AUTOCOMMIT
v1.0.0
@migwell-dev
──────────────────────────────────────────

▌ cmd/model.go  (will be staged)
  cmd/git.go    (staged)
  cmd/root.go
```

## Features

- Browse all changed files (staged and unstaged) in one view
- Preview diffs with syntax highlighting before staging
- Queue files for staging and flush them all at once with `ctrl+s`
- Guided conventional commit flow: type → message → confirm
- Keyboard-driven, no mouse required

## Install

```bash
go install github.com/migwell-dev/autocommit@latest
```

Or clone and build locally:

```bash
git clone https://github.com/migwell-dev/autocommit
cd autocommit
go install .
```

Make sure `$GOPATH/bin` is in your `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

## Usage

Run inside any git repository:

```bash
autocommit
```

## Keybindings

### File list

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `enter` | View file diff |
| `ctrl+a` | Queue all unstaged files |
| `ctrl+x` | Clear staging queue |
| `ctrl+s` | Stage queued files |
| `c` | Start commit (requires staged files) |
| `q` | Quit |

### Diff view

| Key | Action |
|-----|--------|
| `↑` / `k` | Scroll up |
| `↓` / `j` | Scroll down |
| `enter` | Toggle file in staging queue |
| `esc` / `b` | Back to file list |

### Provider view

| Key | Action |
|-----|--------|
| `↑` / `k` | Scroll up |
| `↓` / `j` | Scroll down |
| `enter` | Select Provider |
| `esc` / `b` | Back to commit type list |

### Commit flow

| Key | Action |
|-----|--------|
| `↑` / `↓` | Select commit type |
| `enter` | Confirm / continue |
| `esc` | Go back |

## Commit types

| Type | Description |
|------|-------------|
| `feat` | A new feature |
| `fix` | A bug fix |
| `chore` | Maintenance, tooling, config |
| `docs` | Documentation changes |
| `refactor` | Code restructure, no behavior change |
| `style` | Formatting, missing semicolons, etc |
| `test` | Adding or updating tests |
| `perf` | Performance improvements |

## Requirements

- Go 1.21+
- Git
