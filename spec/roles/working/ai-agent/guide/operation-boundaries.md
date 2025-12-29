# Operation boundaries

## Allowed actions
- Read-only access to the repo (list, search, read).
- Write outputs only under the configured `output_dir`.

## Hard limits
- Enforce per-task budgets for tokens, steps, and wall time.
- Limit file read sizes and tool output volume.

## Safety rules
- Do not modify code in the repo.
- Do not persist secrets in results or reports.

## See also
- [spec/roles/working/core-cli-runner-engineer/requirements/non-functional.md](/working/core-cli-runner-engineer/requirements/non-functional/)
- [spec/roles/working/security-compliance-reviewer/security/security-architecture.md](/working/security-compliance-reviewer/security/security-architecture/)
- [spec/roles/working/agent-llm-integration-engineer/agent/builtin-agent.md](/working/agent-llm-integration-engineer/agent/builtin-agent/)
