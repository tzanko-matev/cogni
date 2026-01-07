Feature: Cucumber evaluation via Godog
  As a repo owner
  I want Cogni to evaluate Cucumber scenarios using Godog
  So I can score the agent's implementation assessment against tests

  Background:
    Given a task of type "cucumber_eval" referencing a Godog adapter
    And the adapter uses runner "godog" with JSON formatter

  Scenario: Run evaluates selected feature files
    Given the task includes "spec/features/cli.feature"
    When I run "cogni run cucumber_cli_features"
    Then Cogni executes Godog for the selected feature files
    And Godog results are captured as JSON
    And each scenario result is mapped to a stable example_id
    And Cogni evaluates each feature file with a single batch agent run
    And the prompt includes the feature path, full feature text, and expected example IDs

  Scenario Outline: Example IDs use explicit tags and row IDs
    Given a scenario tagged "@id:<scenario_id>"
    And the Examples table includes a column "id" with value "<row_id>"
    When the adapter builds example IDs
    Then the example ID is "<scenario_id>:<row_id>"
    Examples:
      | scenario_id     | row_id |
      | cli_run_outputs | e1     |
      | cli_run_outputs | e2     |

  Scenario: Test outcomes map to implemented status
    Given Godog reports a scenario as "passed"
    When Cogni evaluates the example
    Then the ground truth status is "implemented"
    Given Godog reports a scenario as "failed"
    When Cogni evaluates the example
    Then the ground truth status is "not implemented"

  Scenario: Agent answers are scored against tests
    Given an example with ID "cli_run_defaults"
    And the agent returns implemented "true" for that example
    And the Godog result for the example is "passed"
    When Cogni scores the example
    Then the example is marked correct

  Scenario: Batch agent responses must cover all examples
    Given a feature file with examples "cli_run_defaults" and "cli_run_outputs:e1"
    When the agent returns a batch result missing "cli_run_outputs:e1"
    Then the cucumber evaluation task fails with an invalid agent response error
