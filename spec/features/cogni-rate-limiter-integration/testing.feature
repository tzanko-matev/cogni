Feature: Cogni rate limiter integration and concurrency

  Scenario: Disabled mode runs questions concurrently
    Given a question spec with 2 questions
    And a fake provider that sleeps 150 milliseconds per call
    And a config with rate_limiter mode "disabled" and workers 2
    When I run "cogni eval spec/questions/sample.yml"
    Then the run completes within 300 milliseconds

  Scenario: Embedded limiter enforces concurrency
    Given a limits file with concurrency capacity 1 for provider "openrouter" model "model"
    And a question spec with 2 questions
    And a fake provider that sleeps 150 milliseconds per call
    And a config with rate_limiter mode "embedded" and workers 2
    When I run "cogni eval spec/questions/sample.yml"
    Then no more than 1 call is in flight at any time
    And the run completes within 500 milliseconds

  Scenario: Embedded limiter accepts inline limits
    Given inline limits with concurrency capacity 1 for provider "openrouter" model "model"
    And a question spec with 2 questions
    And a fake provider that sleeps 150 milliseconds per call
    And a config with rate_limiter mode "embedded" and workers 2
    When I run "cogni eval spec/questions/sample.yml"
    Then no more than 1 call is in flight at any time
    And the run completes within 500 milliseconds

  Scenario: Remote mode uses ratelimiterd
    Given a stub ratelimiterd server that always allows
    And a question spec with 1 question
    And a config with rate_limiter mode "remote" and the stub base URL
    When I run "cogni eval spec/questions/sample.yml"
    Then the server receives at least 1 reserve request
    And the server receives at least 1 complete request
    And the run completes within 1 second
