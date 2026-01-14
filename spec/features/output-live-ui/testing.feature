Feature: Live console UI for question_eval

  Scenario: Live UI appears for TTY when not verbose
    Given a TTY stdout
    And a question_eval task with 2 questions
    When I run "cogni eval spec/questions/sample.yml"
    Then a live UI is shown
    And the UI lists each question with a status

  Scenario: Tool activity is shown
    Given a question that invokes a tool
    And a TTY stdout
    When I run "cogni eval spec/questions/sample.yml"
    Then the UI shows a tool call status for that question

  Scenario: Non-TTY output falls back to plain summary
    Given stdout is not a TTY
    When I run "cogni eval spec/questions/sample.yml"
    Then the output uses plain summary text
