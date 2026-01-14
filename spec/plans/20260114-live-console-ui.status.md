# Status: Live console UI for concurrent question_eval (2026-01-14)

## Plan
- spec/plans/20260114-live-console-ui.plan.md

## References
- spec/features/output-live-ui/overview.md
- spec/features/output-live-ui/ui.md
- spec/features/output-live-ui/integration.md
- spec/features/output-live-ui/test-suite.md
- spec/features/output-live-ui/implementation-plan.md
- spec/features/output-live-ui/testing.feature
- spec/features/output-console-summary.feature

## Relevant files
- internal/runner/run.go
- internal/runner/question_eval.go
- internal/runner/question_eval_jobs.go
- pkg/ratelimiter/scheduler.go
- pkg/ratelimiter/scheduler_worker.go
- internal/agent/call/runner.go
- internal/agent/call/stream.go
- internal/cli/run.go
- internal/cli/eval.go

## Status
- State: IN_PROGRESS
- Completed steps: none
- Current step: Plan created; ready to start Step 1.
- Notes: Added live UI spec and test-suite pack under spec/features/output-live-ui.
