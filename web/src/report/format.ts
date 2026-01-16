import type { MetricDef } from "./types";

/** Format a metric label for chart axes. */
export function formatMetricLabel(metric: MetricDef): string {
  if (metric.unit && metric.unit.trim().length > 0) {
    return `${metric.name} (${metric.unit})`;
  }
  return metric.name;
}
