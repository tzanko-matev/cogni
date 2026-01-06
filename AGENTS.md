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
We use `flake.nix` and direnv.  To properly run commands in the
development environment you can use `nix develop`. Otherwise
modifications to flake.nix in the agent's session won't be reflected
in the agent's environment.

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

## Testing

* All tests should have configured timeouts. Aim to write tests with short timeouts. 
* We want to use Cucumber .feature files to describe our test suites
  and test cases whenever it makes sense. We'll use godog for the Go code of
  those test cases. Any behaviour which is visible by the user should
  be tested in this way.

## Refactoring and code quality

We want to follow these principles in our codebase:


| Area           | Refactoring Goal                   | Implementation Strategy                                                     |
|----------------|------------------------------------|-----------------------------------------------------------------------------|
| Architecture   | Atomic Modularity                  | Limit files to <200 lines. Use Single Responsibility Principle.             |
| Design Pattern | Functional Core / Imperative Shell | Isolate business logic from I/O to enable safe agentic reasoning.           |
| Typing         | Strict Explicitness                | 100% Type Hint coverage. No any. Use Pydantic/Zod for runtime validation.   |
| Context        | Mapability                         | Flat directory structures. Barrel files (index.ts). AGENTS.md present.      |
| Testing        | Deterministic Feedback             | Zero-flakiness tests. Mock all external dependencies. Rich error reporting. |
| Naming         | Semantic Density                   | "Verbose, intent-revealing names to optimize vector retrieval."             |
| Workflow       | Reflexion Support                  | Fast test suites (<10s) to enable agent self-correction loops.              |

After each successful step and jj commit you are allowed to do a
**partial refactor**. This means: review the code that was just
written and related code. If the code doesn't follow our code quality
guidelines you can refactor it and create a jj commit with message
"refactor:...".
