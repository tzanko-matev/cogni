Feature: YAML configuration format
  As a repo owner
  I want to define tasks, agents, and repo settings in YAML
  So Cogni can run consistently across environments

  Scenario: Minimal valid .cogni.yml passes validation
    Given a file named "spec/questions/sample.yml" with:
      """
      version: 1
      questions:
        - question: "What is 1+1?"
          answers: ["2"]
          correct_answers: ["2"]
      """
    Given a file named ".cogni.yml" with:
      """
      version: 1
      repo:
        output_dir: "./cogni-results"
      agents:
        - id: default
          type: builtin
          provider: "openrouter"
          model: "gpt-4.1-mini"
      default_agent: "default"
      tasks:
        - id: auth_flow_summary
          type: question_eval
          questions_file: "spec/questions/sample.yml"
      """
    When I run "cogni validate"
    Then the exit code is 0

  Scenario: Tasks inherit the default agent
    Given a file named "spec/questions/sample.yml" with:
      """
      version: 1
      questions:
        - question: "What is 1+1?"
          answers: ["2"]
          correct_answers: ["2"]
      """
    Given a file named ".cogni.yml" with a "default_agent" of "default"
    And a task without an explicit "agent" field
    When I run "cogni run --output json"
    Then the task uses the "default" agent

  Scenario: Optional repo setup commands run before tasks
    Given a file named "spec/questions/sample.yml" with:
      """
      version: 1
      questions:
        - question: "What is 1+1?"
          answers: ["2"]
          correct_answers: ["2"]
      """
    Given a file named ".cogni.yml" with:
      """
      version: 1
      repo:
        output_dir: "./cogni-results"
        setup_commands:
          - "go mod download"
      agents:
        - id: default
          type: builtin
          provider: "openrouter"
          model: "gpt-4.1-mini"
      default_agent: "default"
      tasks:
        - id: build_layout
          type: question_eval
          questions_file: "spec/questions/sample.yml"
      """
    When I run "cogni run"
    Then each setup command runs before any task execution

  Scenario: Validation errors are actionable
    Given a file named ".cogni.yml" missing the "tasks" field
    When I run "cogni validate"
    Then the exit code is non-zero
    And the error message names ".cogni.yml" and "tasks"
