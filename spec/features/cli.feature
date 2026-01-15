Feature: Cogni CLI
  As a repository owner
  I want task-oriented CLI commands
  So I can run, compare, and report benchmarks predictably

  Background:
    Given a git repository with a valid Cogni configuration
    And LLM provider credentials are available in the environment

  @smoke
  Scenario: Help lists the primary commands
    When I run "cogni --help"
    Then the output lists these commands:
      | init     |
      | validate |
      | run      |
      | compare  |
      | serve    |
      | report   |

  Scenario: Run uses defaults and produces outputs
    When I run "cogni run"
    Then the exit code is 0
    And the console shows a human-readable summary
    And "results.json" is written under the configured output directory
    And "report.html" is written under the configured output directory

  Scenario: Run supports task selectors and agent overrides
    Given the config defines tasks "auth_flow" and "billing_summary"
    And the config defines agents "default" and "fast"
    When I run "cogni run auth_flow billing_summary@fast --agent default"
    Then only the selected tasks are executed
    And "billing_summary" uses agent "fast"
    And "auth_flow" uses agent "default"

  Scenario: Machine-readable summary output for CI
    When I run "cogni run --output json"
    Then stdout is valid JSON
    And the JSON includes the run summary fields
    And no ANSI color codes are present

  Scenario: Verbose logs with optional color
    When I run "cogni run --verbose"
    Then the console includes per-task tool usage and token metrics
    When I run "cogni run --verbose --no-color"
    Then the verbose output contains no ANSI color codes

  Scenario: Compare highlights deltas between runs
    Given two prior runs exist for commits "main" and "HEAD"
    When I run "cogni compare --base main"
    Then the console shows deltas for pass rate, tokens, and wall time
    And regressions and improvements are listed

  Scenario: Report generates a range view
    Given multiple runs exist for commits in "main..HEAD"
    When I run "cogni report --range main..HEAD --open"
    Then "report.html" includes trend charts for the range
    And the report is opened in the default browser

  Scenario Outline: Progress indicators adapt to the output channel
    When I run "cogni run" with stdout "<stdout>"
    Then progress is shown as "<behavior>"
    Examples:
      | stdout  | behavior            |
      | tty     | in-place indicators |
      | not-tty | line-based updates  |

  @smoke
  Scenario: Exit codes reflect validation failures
    Given the config is invalid
    When I run "cogni validate"
    Then the exit code is non-zero
    And the error message points to the invalid field

  Scenario: Exit codes reflect threshold failures
    Given a failure threshold is configured
    When I run "cogni run"
    Then the exit code is non-zero when the threshold is breached
