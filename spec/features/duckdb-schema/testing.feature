Feature: DuckDB measurement schema invariants
  The measurement schema must be deterministic, idempotent, and safe for plotting.

  Scenario: Schema creation succeeds
    Given a fresh DuckDB database
    When the schema DDL is applied
    Then all core tables and the v_points view exist

  Scenario: Agent upsert is idempotent
    Given a fresh DuckDB database with the schema loaded
    And an agent spec with a stable fingerprint
    When I upsert the same agent spec twice
    Then there is exactly 1 row in the agents table

  Scenario: Measurements obey value-column invariants
    Given a fresh DuckDB database with the schema loaded
    And a metric definition with physical_type "BIGINT"
    And a valid context
    When I insert a measurement with value_bigint set
    Then the invariant query reports 0 invalid rows

  Scenario: v_points exposes plot-ready rows
    Given a fresh DuckDB database with the schema loaded
    And a repo with one revision at timestamp "2026-01-14T12:00:00Z"
    And a tokens metric definition
    And a valid context
    When I insert a tokens measurement with value_bigint set
    Then selecting v_points for metric "tokens" returns 1 row with a non-null ts
