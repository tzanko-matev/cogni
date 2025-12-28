# AGENTS

## Version control

- This repository uses Jujutsu (`jj`) for version control.
- Prefer `jj` commands over `git` unless a task explicitly requires `git`.
- After each self-contained modification, create a new `jj` commit.
  A unit of work is self-contained when all relevant checks pass (for example
  the code builds successfully and relevant tests exist and pass).

## Basic `jj` commands

- `jj status` - show working copy status
- `jj log` - show change history
- `jj diff` - show local diff
- `jj new` - start a new change
- `jj describe -m "message"` - set the change description
- `jj git push` - push changes to a git remote
