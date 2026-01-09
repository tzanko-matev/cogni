# UI and UX

## User journeys

- Initialize `.cogni.yml`, add questions, and run `cogni run`.
- Compare a commit against main using `cogni compare --base main`.
- Generate a trend report for a range using `cogni report --range main..HEAD`.
- Evaluate a Question Spec with `cogni eval` or `question_eval` tasks via `cogni run`.

## Wireframes or mockups

- Repo overview: summary + trend charts (pass rate, tokens, time).
- Run detail: per-task table with expanders for schema and citation errors.

## Accessibility considerations

- Reports should be readable without excessive scrolling.
- Use clear labels for commits, timestamps, and task IDs.
