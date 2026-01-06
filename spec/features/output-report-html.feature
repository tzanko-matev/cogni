Feature: report.html output format
  As a manager or developer
  I want an HTML report with summaries and charts
  So results are easy to share and review

  Scenario: report.html is generated for each run
    When I run "cogni run"
    Then "report.html" is written under the configured output directory
    And the report shows the run summary
    And the report includes a per-task table

  Scenario: Reports for ranges include trends
    Given multiple runs exist across commits
    When I run "cogni report --range main..HEAD"
    Then the report includes trend charts
    And the report summarizes the range window

  Scenario: Reports can be opened automatically
    Given a generated report
    When I run "cogni report --range main..HEAD --open"
    Then the report is opened in the default browser
    And the command exits successfully
