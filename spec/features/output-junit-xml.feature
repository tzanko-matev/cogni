Feature: JUnit XML output format
  As a CI system
  I want JUnit XML output for Cogni results
  So CI dashboards can display pass and fail results

  Scenario: JUnit XML can be emitted for a run
    Given a run with multiple tasks
    When I run "cogni run --output junit"
    Then stdout is valid JUnit XML
    And there is one testcase per task
    And each testcase includes a duration

  Scenario: Failed tasks are reported as failures
    Given a run with a failed task
    When I run "cogni run --output junit"
    Then the failing task includes a failure element
    And the failure message includes the failure_reason
