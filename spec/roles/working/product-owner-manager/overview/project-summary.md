# Project Summary

## One-line description

- cogni is a cognitive benchmarking CLI that measures codebase maintainability by probing it with questions.

## Problem statement

- Teams lack a repeatable, objective way to quantify how understandable a codebase is over time.
- Technical debt is hard to explain to non-engineering stakeholders.

## Solution overview

- Define a repo-specific question suite in `.cogni/config.yml` (stored in `.cogni/`).
- Run an instrumented agent to answer questions and capture resource metrics.
- Generate local results and reports with trends over commit ranges.

## In scope

- Go CLI for `init`, `validate`, `run`, `compare`, and `report`.
- QA-only tasks with JSON answers and citation checks.
- Local outputs (`results.json`, `report.html`) under configurable `output_dir`.
- Git-only repo integration and OpenRouter-only provider in MVP.
- Built-in agent with per-task agent selection.

## Out of scope

- Code-changing tasks (patches/tests/linting).
- Sandboxed runners, SaaS, hosted dashboards, RBAC/SSO.
- Multi-VCS, external agents, multi-provider support (future).

## Success metrics

- New repo can set up and run a benchmark in under 15 minutes.
- Reports show clear trends across a commit range.
- Stakeholders can interpret question results and resource trends.
