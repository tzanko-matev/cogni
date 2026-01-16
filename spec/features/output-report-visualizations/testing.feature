Feature: report visualizations (points + candles)
  As a repository owner
  I want to pick a metric and view points or candlesticks
  So I can understand technical debt trends over time

  Scenario: Metric selector lists numeric metrics
    Given a DuckDB report file with numeric and non-numeric metrics
    When I open the report UI
    Then the metric selector lists only numeric metrics
    And the first numeric metric is selected by default

  Scenario: Points view renders dots and edges
    Given a DuckDB report file with v_points and v_edges
    When I select a metric
    And the view is set to "Points"
    Then the chart shows dots for measured commits
    And the chart shows links for minimal ancestor edges

  Scenario: Points view falls back when edges are missing
    Given a DuckDB report file without v_edges
    When I select a metric
    And the view is set to "Points"
    Then the chart shows dots for measured commits
    And the UI displays a warning about missing edge data

  Scenario: Candles view renders OHLC candles
    Given a DuckDB report file with v_candles
    When I select a metric
    And the view is set to "Candles"
    Then the chart shows candlestick wicks and bodies

  Scenario: Candles view reports missing data
    Given a DuckDB report file without v_candles
    When I select a metric
    And the view is set to "Candles"
    Then the chart is empty
    And the UI displays a warning about missing candle data
