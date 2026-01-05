# AGENTS

## Goal
We are building Cogni: A tool and a SaaS for controlling technical
debt. Our headline message is "Cogni: Take control of your techical
debt"

This repo is a complete repository of ALL information about Cogni. It
contains all business documents, all project planning, all of the code
and so on.

The ultimate goal is to be able to generate personal income for me of
at least $10000 per month. I need to reach this goal as soon as
possible.

## Cogni's core idea
We can estimate technical debt using AI agents. The idea is that we
will ask questions about a codebase to an agent and we will measure
the **effort** that the agent takes to answer those questions. We will
track those measurements over time and in this way show the state of
the technical debt in the repository.


## Local Development Environment
We use `flake.nix` and direnv. 

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
- `jj split -m ... file.txt` - non-interactive split command. Without the `-m` flag it will start an editor
