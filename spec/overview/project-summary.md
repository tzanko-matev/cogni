# Project Summary

## One-line description

- Cogni is a slop-testing framework.

## Slop testing definition

- "Slop" means badly-written AI- or human-generated code (context: https://www.merriam-webster.com/wordplay/word-of-the-year).
- A slop test is a question about your code.
- Cogni has AI agents repeatedly answer the question and measures the effort and how it changes as the codebase evolves, exposing maintainability in a way that is comprehensible across the org.

## Problem statement

- Teams lack a repeatable, objective way to quantify how understandable a codebase is over time.
- Technical debt is hard to explain to non-engineering stakeholders.

## Solution overview

- Define a repo-specific question suite in `.cogni.yml`.
- Run an instrumented agent to answer questions and capture resource metrics.
- Generate local results and reports with trends over commit ranges.

## In scope

- Go CLI for `init`, `validate`, `run`, `compare`, and `report`.
- QA tasks with JSON answers and citation checks.
- Cucumber evaluation tasks with Godog or manual expectations.
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
