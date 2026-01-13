# Integration and End-to-End Test Suite

## Purpose

- Prove that Cogni can run against a real LLM provider and produce usable results.
- Validate the full CLI workflow a real user follows: init, validate, run, report, compare.
- Provide stakeholder-friendly evidence that benchmark outputs are trustworthy and stable.

## Audience

- Engineering managers, product owners, and users who run benchmarks in their repos.

## Definitions

- Integration test: exercises Cogni with a live LLM provider while controlling inputs.
- End-to-end (E2E) test: runs the CLI as a user would, producing outputs and reports.

## Test environments

- Local: developer machine with valid `LLM_PROVIDER`, `LLM_API_KEY`, and `LLM_MODEL`.
- CI (smoke): a minimal set that confirms provider connectivity and artifacts.
- CI (full): scheduled or manual run covering all cases, due to cost and rate limits.

## Fixture recipes and inputs

Fixture repos are created on demand so we do not store repos inside this repo.
Each fixture is described as a short sequence of `git` and `cat` commands.
Run each recipe in an empty directory created by the test harness.

### Simple repo recipe (single commit)

```bash
git init
git config user.email "cogni-tests@example.com"
git config user.name "Cogni Tests"
cat > README.md <<'EOF'
# Sample Service
This repo exists only for Cogni integration tests.
EOF
cat > app.md <<'EOF'
# App Notes
Service owner: Platform Team
EOF
cat > config.yml <<'EOF'
service_name: Sample Service
owner: Platform Team
EOF
mkdir -p config
cat > config/app-config.yml <<'EOF'
mode: sample
EOF
git add README.md app.md config.yml config/app-config.yml
git commit -m "init"
```

### History repo recipe (two commits)

```bash
git init
git config user.email "cogni-tests@example.com"
git config user.name "Cogni Tests"
cat > README.md <<'EOF'
# Sample Service
Release stage: alpha
EOF
cat > change-log.md <<'EOF'
- 0.1.0: initial
EOF
git add README.md change-log.md
git commit -m "init"
cat > README.md <<'EOF'
# Sample Service
Release stage: beta
EOF
git add README.md
git commit -m "update release stage"
```

### Config recipe (per test)

Use a minimal `.cogni/config.yml` created with `cat`. Create `.cogni/` and `.cogni/schemas/` first. Adjust tasks per test case.

```bash
mkdir -p .cogni/schemas
cat > .cogni/config.yml <<'EOF'
version: 1
repo:
  output_dir: ".cogni/results"
agents:
  - id: default
    type: builtin
    provider: "openrouter"
    model: "gpt-4.1-mini"
    temperature: 0.0
default_agent: "default"
tasks:
  - id: sample_task
    type: question_eval
    questions_file: "spec/questions/sample.yml"
EOF
```

Questions are written in plain language with objective, verifiable answers
(titles, file names, short enumerations).

### Question spec fixture recipe (per test)

Create a small Question Spec file for question evaluation tasks.

```bash
mkdir -p spec/questions
cat > spec/questions/sample.yml <<'EOF'
version: 1
questions:
  - id: q1
    question: "What is the project title?"
    answers: ["Sample Service", "Other"]
    correct_answers: ["Sample Service"]
EOF
```

## Acceptance criteria (global)

- CLI exits successfully for passing runs and clearly signals failures otherwise.
- `results.json` and `report.html` are produced for successful runs.
- Task outcomes are understandable to non-engineers (pass/fail with reasons).
- Answers may vary in wording, but must satisfy each task's success criteria.

## Test cases

### T1: Provider connectivity smoke test

- Goal: confirm a valid provider key enables a full run.
- Setup: Simple repo recipe with one QA task in `.cogni/config.yml`.
- Steps: run `cogni run` from the fixture repo.
- Expected: run succeeds, 1 task passes, artifacts exist.

### T2: Basic QA with citations

- Goal: verify the agent cites a specific file to support an answer.
- Setup: Simple repo recipe where README contains a clear sentence to quote.
- Steps: ask "What is the project name? Cite the README."
- Expected: pass status; answer references the README content.

### T3: Multi-file evidence

- Goal: ensure the agent can combine evidence from more than one file.
- Setup: Simple repo recipe where README and app notes each contain part of the answer.
- Steps: ask a question that requires both sources.
- Expected: pass status; citations reference both files.

### T4: Repository navigation

- Goal: confirm the agent can locate information without hints.
- Setup: Simple repo recipe includes `config/app-config.yml`.
- Steps: ask "Where is app-config.yml located? Provide the path."
- Expected: pass status; answer includes the correct relative path.

### T5: Multiple tasks in one run

- Goal: validate summary counts and per-task statuses.
- Setup: Simple repo recipe and a config with 3 QA tasks of varying difficulty.
- Steps: run `cogni run`.
- Expected: all tasks report status; summary counts match task outcomes.

### T6: Multiple agents / model override

- Goal: show that per-task agent selection works in real runs.
- Setup: Simple repo recipe and a config with two agents and one task per agent.
- Steps: run `cogni run`.
- Expected: results list the correct agent and model per task.

### T7: Budget limits and graceful failure

- Goal: ensure limits prevent runaway runs with a clear outcome.
- Setup: Simple repo recipe and one task with intentionally strict limits.
- Steps: run `cogni run`.
- Expected: task fails with a clear "budget exceeded" style reason; run continues.

### T8: Output artifacts integrity

- Goal: confirm artifacts are well-formed and complete.
- Setup: any passing run.
- Steps: inspect `results.json` and `report.html`.
- Expected: results include run metadata, task list, model/provider info; report loads.

### T9: Compare across commits

- Goal: verify `cogni compare` highlights changes across runs.
- Setup: History repo recipe with two commits and two runs.
- Steps: run twice, then execute `cogni compare`.
- Expected: compare report exists and shows per-task deltas.

### T10: Config validation errors

- Goal: ensure invalid configs are caught early and clearly.
- Setup: create an invalid `.cogni/config.yml` with missing required fields.
- Steps: run `cogni validate`.
- Expected: command fails with actionable error messages.

### T11: Init-to-run flow

- Goal: demonstrate a "first run" experience.
- Setup: empty fixture repo.
- Steps: run `cogni init`, confirm the proposed `.cogni/` location, choose a results folder (accept default), accept adding it to `.gitignore`, edit prompts minimally, then run `cogni run`.
- Expected: `.cogni/` created at the repo root, `repo.output_dir` matches the chosen folder, `.gitignore` includes the entry, run succeeds, artifacts produced.

### T12: Init results folder and gitignore opt-out

- Goal: ensure users can decline `.gitignore` changes and pick a custom output folder.
- Setup: empty fixture repo with an existing `.gitignore`.
- Steps: run `cogni init`, choose a non-default results folder, and decline `.gitignore` changes.
- Expected: `.cogni/config.yml` uses the chosen folder; `.gitignore` remains unchanged.

### T13: Config discovery from subdir

- Goal: confirm commands work when invoked from a nested folder.
- Setup: Simple repo recipe with `.cogni/` at the repo root.
- Steps: `mkdir -p nested/dir`, `cd nested/dir`, run `cogni validate` and `cogni run`.
- Expected: CLI locates `.cogni/` by walking up parent directories and completes.

### T14: Provider failure handling

- Goal: verify clear errors when the provider is unavailable.
- Setup: Simple repo recipe and an invalid API key or disabled provider.
- Steps: run `cogni run`.
- Expected: CLI fails with a clear authentication/availability message.

### T15: Question evaluation via task

- Goal: evaluate a Question Spec using `question_eval`.
- Setup: Simple repo recipe, Question Spec fixture, and a `question_eval` task.
- Steps: run `cogni run question_eval_core`.
- Expected: run succeeds, per-question verdicts exist, and accuracy is computed.

### T16: Question evaluation via eval command

- Goal: evaluate a Question Spec with the direct CLI.
- Setup: Simple repo recipe and Question Spec fixture.
- Steps: run `cogni eval spec/questions/sample.yml --agent default`.
- Expected: run succeeds, per-question verdicts exist, and accuracy is computed.
