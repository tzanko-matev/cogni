Feature: serve browser report from DuckDB
  As a repository owner
  I want a local server for viewing Cogni reports
  So I can inspect results in the browser

  Scenario: Report HTML is served
    Given a DuckDB report file
    When I start the report server
    And I request "/"
    Then the response status is 200
    And the response body contains "Cogni Report"

  Scenario: DuckDB file is served
    Given a DuckDB report file
    When I start the report server
    And I request "/data/db.duckdb"
    Then the response status is 200
    And the response body equals the DuckDB file bytes

  Scenario: Assets base URL overrides links
    Given a DuckDB report file
    And an assets base URL "https://assets.example.com"
    When I start the report server
    And I request "/"
    Then the response status is 200
    And the response body contains "https://assets.example.com"
