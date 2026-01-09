# Evaluation guide

## What you should verify
- Time to first benchmark is under 15 minutes.
- Results are understandable and backed by citations.
- Outputs are local and easy to share (`report.html`).

## Quick steps
- Run `cogni init`, confirm the suggested `.cogni/` location, and choose a results folder (default `.cogni/results`).
- If a git repo is detected, decide whether to add the results folder to `.gitignore`.
- Add a small task in `.cogni/config.yml`.
- If you have a Question Spec, add a `question_eval` task that references the spec file.
- Run `cogni validate` and `cogni run`.
- Generate a report with `cogni report --range main..HEAD`.
- Run commands from any subdirectory; Cogni finds `.cogni/` by walking up parent directories.

## Constraints to note
- MVP is local-only, read-only, and git-only.
- No hosted dashboards or multi-tenant features in MVP.

## See also
- [spec/roles/working/documentation-education-owner/guides/user-tutorial.md](/working/documentation-education-owner/guides/user-tutorial/)
- [spec/roles/working/core-cli-runner-engineer/engineering/build-and-run.md](/working/core-cli-runner-engineer/engineering/build-and-run/)
- [spec/roles/working/product-owner-manager/overview/project-summary.md](/working/product-owner-manager/overview/project-summary/)
