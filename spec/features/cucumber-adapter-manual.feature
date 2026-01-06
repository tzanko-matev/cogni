Feature: Cucumber evaluation via manual expectations
  As a repo owner
  I want to define expected outcomes for Cucumber scenarios manually
  So Cogni can evaluate implementation even without a test suite

  Background:
    Given a task of type "cucumber_eval" referencing a manual adapter
    And the adapter points to an expectations directory

  Scenario: Manual expectations are used as ground truth
    Given an expectations file defines example "cli_run_defaults" as implemented
    When I run "cogni run cucumber_cli_manual"
    Then Cogni does not execute any test runner
    And the ground truth for "cli_run_defaults" is "implemented"

  Scenario Outline: Example IDs avoid line numbers
    Given a scenario tagged "@id:<scenario_id>"
    And the Examples table includes a column "id" with value "<row_id>"
    When the adapter builds example IDs
    Then the example ID is "<scenario_id>:<row_id>"
    Examples:
      | scenario_id     | row_id |
      | cli_run_outputs | e1     |
      | cli_run_outputs | e2     |

  Scenario: Agent answers are scored against expectations
    Given the expectations file marks example "cli_run_defaults" as implemented
    And the agent returns implemented "true" for that example
    When Cogni scores the example
    Then the example is marked correct
