import { describe, expect, it } from "vitest";
import { filterNumericMetrics, selectMetricName } from "../state";
import type { MetricDef } from "../types";

describe("metric selection", () => {
  it("filters numeric metrics", () => {
    const metrics: MetricDef[] = [
      { name: "tokens", description: null, unit: null, physicalType: "BIGINT" },
      { name: "label", description: null, unit: null, physicalType: "VARCHAR" },
    ];

    expect(filterNumericMetrics(metrics).map((metric) => metric.name)).toEqual(["tokens"]);
  });

  it("selects first metric alphabetically when no previous selection", () => {
    const metrics: MetricDef[] = [
      { name: "zeta", description: null, unit: null, physicalType: "DOUBLE" },
      { name: "alpha", description: null, unit: null, physicalType: "DOUBLE" },
    ];

    expect(selectMetricName(metrics, null)).toBe("alpha");
  });

  it("preserves the previous selection when available", () => {
    const metrics: MetricDef[] = [
      { name: "tokens", description: null, unit: null, physicalType: "BIGINT" },
      { name: "latency", description: null, unit: null, physicalType: "DOUBLE" },
    ];

    expect(selectMetricName(metrics, "latency")).toBe("latency");
  });
});
