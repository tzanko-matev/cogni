# User tutorial
This is how cogni looks in day-to-day use for a repo owner.

1) Add questions
   - Start with `cogni init`, confirm the suggested `.cogni/` location, and scaffold `.cogni/config.yml` plus `.cogni/schemas/`.
   - In a git repo, `cogni init` defaults to the repo root; otherwise it uses the current folder.
   - Choose a results folder when prompted (default `.cogni/results`); this is written to `repo.output_dir`.
   - If a git repo is detected, decide whether to add the results folder to `.gitignore`.
   - Define `question_eval` tasks that reference a Question Spec file.
   - Define one or more agents and assign each question to an agent (or rely on the default agent).
   - Set the output folder once in `.cogni/config.yml` so CLI commands stay short.
   - Save the following example in `.cogni/config.yml` (sample questions for the future cogni codebase):

     ```yaml
     repo:
       output_dir: ".cogni/results"
     agents:
       - id: default
         type: builtin
         provider: "openrouter"
         model: "gpt-4.1-mini"
         max_steps: 25
         temperature: 0.0
     default_agent: "default"
     tasks:
       - id: core_questions
         type: question_eval
         agent: "default"
         questions_file: "spec/questions/core.yml"
         budget:
           max_tokens: 9000
           max_seconds: 120
     ```

   - Example Question Spec (`spec/questions/core.yml`):

     ```yaml
     version: 1
     questions:
       - id: cli_commands
         question: "Which CLI commands are supported by Cogni?"
         answers: ["run, eval, compare, report", "only run"]
         correct_answers: ["run, eval, compare, report"]
     ```

2) Validate the spec
   - `cogni validate` ensures YAML and JSON schemas are correct before running.

3) Run the benchmark
   - `cogni run` (runs the whole benchmark at the current commit)
   - `cogni run question-id1 question-id2@my-agent` (run a subset, choose an agent for a specific question)
   - `cogni run --agent default` (override agent for all tasks in this run)
   - Produces `results.json` and `report.html` under `<output_dir>/<commit>/<run-id>/`
   - Prints a terminal summary with pass rate and resource usage.
   - Commands can run from subdirectories; Cogni finds `.cogni/` by walking up parent directories.

4) Compare runs in the CLI
   - `cogni compare --base main` (compare the current commit against main)
   - `cogni compare --range main..HEAD`
   - For `--range`, cogni queries the repo to expand the commit list, then compares the runs in that window.
   - Shows deltas in pass rate, tokens, and time, plus any questions that regressed.

5) Produce and view a report
   - `cogni report --range main..HEAD --open`
   - For `--range`, cogni queries the repo to expand the commit list and renders trends for that window.
   - Use a directory with multiple runs to render trend charts.

---
