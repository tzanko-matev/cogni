# User tutorial
This is how cogni looks in day-to-day use for a repo owner.

1) Add questions
   - Start with `cogni init`, confirm the suggested `.cogni/` location, and scaffold `.cogni/config.yml` plus `.cogni/schemas/`.
   - In a git repo, `cogni init` defaults to the repo root; otherwise it uses the current folder.
   - Choose a results folder when prompted (default `.cogni/results`); this is written to `repo.output_dir`.
   - If a git repo is detected, decide whether to add the results folder to `.gitignore`.
   - Define `qa` tasks with prompts tied to key product features and stakeholder concerns.
   - If you maintain a Question Spec, add `question_eval` tasks that reference the spec file.
   - Require citations so answers are traceable to code.
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
       - id: cli_command_map
         type: qa
         agent: "default"
         prompt: >
           List the CLI commands supported by cogni and where each is implemented.
           Return JSON with keys:
           {"commands":[{"name":...,"file":...,"description":...}],"citations":[{"path":...,"lines":[start,end]}]}
         eval:
           must_contain_strings: ["commands", "citations"]
           validate_citations: true
         budget:
           max_tokens: 8000
           max_seconds: 120
       - id: report_generation_flow
         type: qa
         prompt: >
           Explain how cogni generates report.html, including which inputs it reads and how it
           summarizes results. Return JSON with keys:
           {"inputs":[...],"outputs":[...],"steps":[...],"citations":[{"path":...,"lines":[start,end]}]}
         eval:
           must_contain_strings: ["inputs", "outputs", "citations"]
           validate_citations: true
         budget:
           max_tokens: 9000
           max_seconds: 120
       - id: results_json_summary
         type: qa
         prompt: >
           Describe how results.json is structured and where summary metrics are computed.
           Return JSON with keys:
           {"summary_fields":[...],"computation":[...],"citations":[{"path":...,"lines":[start,end]}]}
         eval:
           must_contain_strings: ["summary_fields", "citations"]
           validate_citations: true
         budget:
           max_tokens: 9000
           max_seconds: 120
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
