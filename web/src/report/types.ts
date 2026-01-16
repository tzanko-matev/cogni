import { z } from "zod";

/** UI status levels used by the report header. */
export type StatusLevel = "idle" | "loading" | "ready" | "error";

/** Available chart view modes. */
export type ViewMode = "points" | "candles";

/** Bucket sizes for candle aggregation. */
export type BucketSize = "day" | "week" | "month";

/** Metric definition row extracted from DuckDB. */
export interface MetricDef {
  name: string;
  description: string | null;
  unit: string | null;
  physicalType: string;
}

/** Metric point derived from v_points. */
export interface MetricPoint {
  repoId: string;
  revId: string;
  ts: Date;
  value: number;
}

/** Parent edge row from revision_parents. */
export interface ParentEdge {
  childRevId: string;
  parentRevId: string;
}

/** XY edge for plotting links. */
export interface EdgeXY {
  x1: Date;
  y1: number;
  x2: Date;
  y2: number;
}

/** Candle aggregate for a bucket component. */
export interface Candle {
  bucket: string;
  componentId: string;
  x: Date;
  open: number;
  close: number;
  low: number;
  high: number;
}

/** Component link between candle vertices. */
export interface ComponentEdgeXY {
  x1: Date;
  y1: number;
  x2: Date;
  y2: number;
}

const MetricDefRowSchema = z.object({
  name: z.string(),
  description: z.string().nullable().optional(),
  unit: z.string().nullable().optional(),
  physical_type: z.string(),
});

const MetricPointRowSchema = z.object({
  repo_id: z.string(),
  rev_id: z.string(),
  ts: z.unknown(),
  value: z.number(),
});

const ParentEdgeRowSchema = z.object({
  child_rev_id: z.string(),
  parent_rev_id: z.string(),
});

/** Parse metric definition rows from DuckDB. */
export function parseMetricDefRows(rows: unknown[]): MetricDef[] {
  return rows.map((row) => {
    const parsed = MetricDefRowSchema.parse(row);
    return {
      name: parsed.name,
      description: parsed.description ?? null,
      unit: parsed.unit ?? null,
      physicalType: parsed.physical_type,
    };
  });
}

/** Convert a DuckDB timestamp cell into a Date. */
export function parseTimestamp(value: unknown): Date {
  if (value instanceof Date) {
    return value;
  }
  if (typeof value === "string" || typeof value === "number") {
    const date = new Date(value);
    if (!Number.isNaN(date.getTime())) {
      return date;
    }
  }
  throw new Error("Invalid timestamp value from DuckDB.");
}

/** Parse metric point rows from DuckDB. */
export function parseMetricPointRows(rows: unknown[]): MetricPoint[] {
  return rows.map((row) => {
    const parsed = MetricPointRowSchema.parse(row);
    return {
      repoId: parsed.repo_id,
      revId: parsed.rev_id,
      ts: parseTimestamp(parsed.ts),
      value: parsed.value,
    };
  });
}

/** Parse parent edge rows from DuckDB. */
export function parseParentEdgeRows(rows: unknown[]): ParentEdge[] {
  return rows.map((row) => {
    const parsed = ParentEdgeRowSchema.parse(row);
    return {
      childRevId: parsed.child_rev_id,
      parentRevId: parsed.parent_rev_id,
    };
  });
}
