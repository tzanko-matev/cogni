/** Escape a string literal for DuckDB SQL. */
export function escapeSqlString(value: string): string {
  return value.replace(/'/g, "''");
}

/** Wrap a string in single-quoted SQL literal form. */
export function sqlStringLiteral(value: string): string {
  return `'${escapeSqlString(value)}'`;
}

/** Build SQL for the metric_points temp view. */
export function buildMetricPointsViewSQL(metric: string): string {
  const metricLiteral = sqlStringLiteral(metric);
  return `
    CREATE OR REPLACE TEMP VIEW metric_points AS
    SELECT
      repo_id,
      rev_id,
      ts,
      arg_max(value, run_ts) AS value
    FROM (
      SELECT v.repo_id, v.rev_id, v.ts, v.value, r.collected_at AS run_ts
      FROM cogni.main.v_points v
      JOIN cogni.main.runs r ON r.run_id = v.run_id
      WHERE v.metric = ${metricLiteral}
        AND v.status = 'ok'
        AND v.value IS NOT NULL
    ) grouped
    GROUP BY repo_id, rev_id, ts
  `;
}

/** Build SQL to fetch metric points for graph computation. */
export function buildMetricPointsSelectSQL(): string {
  return `
    SELECT repo_id, rev_id, ts, value
    FROM metric_points
    ORDER BY ts
  `;
}
