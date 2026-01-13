Feature: JSON configuration format
  As a repo owner
  I want to define tasks, agents, and repo settings in JSON
  So Cogni can use the same schema as YAML

  Scenario: Minimal valid .cogni.json passes validation
    Given a file named "spec/questions/sample.json" with:
      """
      {
        "version": 1,
        "questions": [
          {
            "question": "What is 1+1?",
            "answers": ["2"],
            "correct_answers": ["2"]
          }
        ]
      }
      """
    Given a file named ".cogni.json" with:
      """
      {
        "version": 1,
        "repo": {
          "output_dir": "./cogni-results"
        },
        "agents": [
          {
            "id": "default",
            "type": "builtin",
            "provider": "openrouter",
            "model": "gpt-4.1-mini"
          }
        ],
        "default_agent": "default",
        "tasks": [
          {
            "id": "auth_flow_summary",
            "type": "question_eval",
            "questions_file": "spec/questions/sample.json"
          }
        ]
      }
      """
    When I run "cogni validate --spec .cogni.json"
    Then the exit code is 0

  Scenario: JSON config uses the same validation rules as YAML
    Given a file named ".cogni.json" missing the "tasks" field
    When I run "cogni validate --spec .cogni.json"
    Then the exit code is non-zero
    And the error message names ".cogni.json" and "tasks"
