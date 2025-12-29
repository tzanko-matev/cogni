# Evaluation workflow

## Demo setup
- Run `cogni init` to scaffold `.cogni.yml`.
- Add a small set of QA tasks with citation checks.
- Set `output_dir` so runs are repeatable.

## Demo run
- `cogni validate`
- `cogni run`
- `cogni report --range main..HEAD`

## Artifacts to share
- `results.json` for raw outcomes.
- `report.html` for stakeholder-friendly output.

## See also
- [spec/roles/working/documentation-education-owner/guides/user-tutorial.md](/working/documentation-education-owner/guides/user-tutorial/)
- [spec/roles/working/core-cli-runner-engineer/engineering/configuration.md](/working/core-cli-runner-engineer/engineering/configuration/)
- [spec/roles/working/core-cli-runner-engineer/engineering/build-and-run.md](/working/core-cli-runner-engineer/engineering/build-and-run/)
