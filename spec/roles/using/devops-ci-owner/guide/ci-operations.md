# CI operations

## Goal
Run Cogni in CI reliably, capture artifacts, and keep costs predictable.

## Pipeline shape
- Run `go test ./...`.
- Build the `cogni` binary.
- Run `cogni validate` and (optionally) `cogni run`.
- Upload `results.json` and `report.html` artifacts.

## Environment variables
- `LLM_PROVIDER`
- `LLM_MODEL`
- `LLM_API_KEY`

## Operational notes
- Runs should be deterministic and always write `results.json` even on partial failures.
- Backup `output_dir` for historical comparisons.
- Rotate API keys if compromised.

## See also
- [spec/roles/working/release-ci-engineer/ci/ci-cd.md](/working/release-ci-engineer/ci/ci-cd/)
- [spec/roles/working/release-ci-engineer/ci/deployment.md](/working/release-ci-engineer/ci/deployment/)
- [spec/roles/working/customer-support-success/support/runbooks.md](/working/customer-support-success/support/runbooks/)
- [spec/roles/working/customer-support-success/support/backup-restore.md](/working/customer-support-success/support/backup-restore/)
