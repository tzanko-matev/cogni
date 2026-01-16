import type { BucketSize, MetricDef, ViewMode } from "./types";

/** Filter to metrics with numeric physical types. */
export function filterNumericMetrics(metrics: MetricDef[]): MetricDef[] {
  return metrics.filter((metric) => metric.physicalType === "DOUBLE" || metric.physicalType === "BIGINT");
}

/** Sort metrics by name ascending for stable UI ordering. */
export function sortMetricsByName(metrics: MetricDef[]): MetricDef[] {
  return [...metrics].sort((a, b) => a.name.localeCompare(b.name));
}

/**
 * Select the active metric name based on previous selection.
 */
export function selectMetricName(metrics: MetricDef[], previous?: string | null): string | null {
  if (metrics.length === 0) {
    return null;
  }
  if (previous && metrics.some((metric) => metric.name === previous)) {
    return previous;
  }
  return sortMetricsByName(metrics)[0]?.name ?? null;
}

/** Validate whether a value is a supported bucket size. */
export function isBucketSize(value: string): value is BucketSize {
  return value === "day" || value === "week" || value === "month";
}

/** Normalize a bucket size value to a supported default. */
export function normalizeBucketSize(value: string | null | undefined): BucketSize {
  if (value && isBucketSize(value)) {
    return value;
  }
  return "day";
}

/** Normalize a view mode value to a supported default. */
export function normalizeViewMode(value: string | null | undefined): ViewMode {
  if (value === "candles") {
    return "candles";
  }
  return "points";
}
