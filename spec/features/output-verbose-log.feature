Feature: Verbose log file output
  As a repository owner
  I want to write verbose logs to a file
  So I can inspect full run logs without changing stdout

  Background:
    Given a git repository with a valid Cogni configuration
    And LLM provider credentials are available in the environment

  Scenario: Log file is written when --log is set
    When I run "cogni run --log run.log"
    Then the log file "run.log" exists
    And the log file "run.log" contains "[verbose]"

  Scenario: Log file captures verbose output without changing stdout
    When I run "cogni run --log run.log"
    Then stdout does not include verbose logs
    And the log file "run.log" contains "LLM prompt"

  Scenario: Log file and stdout both receive verbose output when --verbose and --log are set
    When I run "cogni run --verbose --log run.log"
    Then the console includes verbose logs
    And the log file "run.log" contains "LLM output"
