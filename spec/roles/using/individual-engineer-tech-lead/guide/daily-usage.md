# Daily usage

## What you do
Run targeted checks to validate changes and spot regressions in understanding.

## Common workflows
- Run a subset of tasks: `cogni run task-id1 task-id2@agent-id`.
- Compare to main: `cogni compare --base main`.
- Generate trends: `cogni report --range main..HEAD`.

## Evidence and outcomes
- Tasks should return JSON with citations that point to repo files.
- Failures should include a reason (invalid JSON, missing citations, budget exceeded).

## When something fails
- Validate the config first: `cogni validate`.
- Check `results.json` and per-task logs in the run output folder.

## See also
- [spec/roles/working/core-cli-runner-engineer/engineering/build-and-run.md](/working/core-cli-runner-engineer/engineering/build-and-run/)
- [spec/roles/working/customer-support-success/support/troubleshooting.md](/working/customer-support-success/support/troubleshooting/)
- [spec/roles/working/qa-test-engineer/testing/integration-e2e-tests.md](/working/qa-test-engineer/testing/integration-e2e-tests/)
