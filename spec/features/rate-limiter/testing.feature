Feature: Rate limiter core behavior

  Scenario: Rolling limit denies when capacity exceeded
    Given a rolling limit "global:llm:test:model:tpm" with capacity 2 and window 60 seconds
    When I reserve amount 1 for lease "L1"
    And I reserve amount 1 for lease "L2"
    And I reserve amount 1 for lease "L3"
    Then the third reserve is denied
    And the response is returned within 200 milliseconds

  Scenario: Linked reservations are atomic
    Given limits "provider" capacity 1 and "user" capacity 100 in the same request
    When I reserve amount 2 for both limits in a single request
    Then the reserve is denied
    And the "user" limit remains at 100
    And the response is returned within 200 milliseconds

  Scenario: Concurrency is released on Complete
    Given a concurrency limit "global:llm:test:model:concurrency" with capacity 1 and timeout 300 seconds
    When I reserve amount 1 for lease "C1"
    And I complete lease "C1"
    Then I can reserve amount 1 for lease "C2" within 200 milliseconds

  Scenario: Overage creates debt
    Given a rolling limit "global:llm:test:model:tpm" with capacity 100 and overage "debt"
    When I reserve amount 100 for lease "D1"
    And I complete lease "D1" with actual amount 150
    Then the debt for "global:llm:test:model:tpm" is 50
    And the response is returned within 200 milliseconds

  Scenario: Batch reserve preserves order
    Given a rolling limit "global:llm:test:model:rpm" with capacity 1 and window 60 seconds
    When I send a batch reserve with leases "B1" and "B2" for amount 1 each
    Then result 1 is allowed and result 2 is denied
    And the batch response is returned within 300 milliseconds

  Scenario: Capacity decrease blocks new reservations until applied
    Given a rolling limit "global:llm:test:model:tpm" with capacity 100 and window 60 seconds
    And I reserve amount 80 for lease "DC1"
    When the admin decreases capacity for "global:llm:test:model:tpm" to 60
    Then new reservations for "global:llm:test:model:tpm" are denied with error "limit_decreasing:global:llm:test:model:tpm"
    And when the available balance is at least 40 the decrease is applied
    And reservations are accepted again
