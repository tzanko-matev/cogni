Feature: Environment variable configuration
  As a repo owner
  I want to keep provider credentials out of config files
  So sensitive data is not committed to source control

  Scenario: Provider credentials are supplied via environment variables
    Given a valid ".cogni.yml" without provider credentials
    And the environment variables "LLM_PROVIDER", "LLM_MODEL", and "LLM_API_KEY" are set
    When I run "cogni run"
    Then the run succeeds using the provider settings from the environment

  Scenario: Missing credentials fail fast
    Given a valid ".cogni.yml" without provider credentials
    And the environment variable "LLM_API_KEY" is not set
    When I run "cogni run"
    Then the exit code is non-zero
    And the error message explains which credential is missing
