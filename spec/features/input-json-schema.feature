Feature: JSON Schema inputs for task validation
  As a repo owner
  I want tasks to validate structured answers against JSON Schema
  So Cogni can enforce objective success criteria

  Scenario: Valid schema files pass validation
    Given a file named ".cogni.yml" with a task that references "schemas/auth_flow.schema.json"
    And "schemas/auth_flow.schema.json" is a valid JSON Schema file
    When I run "cogni validate"
    Then the exit code is 0

  Scenario: Missing or invalid schemas fail validation
    Given a file named ".cogni.yml" with a task that references "schemas/missing.schema.json"
    When I run "cogni validate"
    Then the exit code is non-zero
    And the error message references "schemas/missing.schema.json"

  Scenario: Schema mismatch fails a task
    Given a task with "json_schema" set to "schemas/auth_flow.schema.json"
    And the agent response does not match the schema
    When I run "cogni run"
    Then the task status is "fail"
    And the failure_reason is "schema_validation_failed"
