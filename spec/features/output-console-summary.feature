Feature: Console summary output
  As a developer
  I want a concise human-readable summary in the terminal
  So I can assess benchmark results quickly

  Scenario: Default summary includes key metrics
    Given a run with 2 tasks where 1 passes and 1 fails
    When I run "cogni run"
    Then the console summary includes:
      | tasks_total |
      | tasks_passed |
      | tasks_failed |
      | pass_rate |
    And the summary reports tokens and wall time totals
    And a per-task status table is shown

  Scenario: ANSI color is used only when requested
    When I run "cogni run" in a TTY
    Then pass statuses are colored green and failures are colored red
    When I run "cogni run --no-color"
    Then the output contains no ANSI color codes
