Feature: results.json output format
  As a developer or CI system
  I want machine-readable run data in results.json
  So I can track technical debt over time

  Scenario: results.json contains run metadata and summary
    Given a successful run
    When I inspect "results.json"
    Then it includes:
      | run_id |
      | repo.name |
      | repo.vcs |
      | repo.commit |
      | agents |
      | started_at |
      | finished_at |
      | tasks |
      | summary |

  Scenario: Each task records attempts and metrics
    Given a successful run with at least one task
    When I inspect "results.json"
    Then each task has one or more "attempts"
    And each attempt includes:
      | status |
      | agent_id |
      | model |
      | tokens_in |
      | tokens_out |
      | tokens_total |
      | wall_time_seconds |
      | agent_steps |
      | tool_calls |
      | unique_files_read |

  Scenario: Summary totals align with task outcomes
    Given a run with tasks that pass and fail
    When I inspect "results.json"
    Then summary.tasks_total equals the number of tasks
    And summary.tasks_passed plus summary.tasks_failed equals summary.tasks_total
    And summary.pass_rate reflects the task outcomes

  Scenario: Failures include clear reasons
    Given a run with a failed task
    When I inspect "results.json"
    Then the task "status" is "fail"
    And "failure_reason" explains the failure
