import { describe, expect, it } from "vitest";
import { formatMetricLabel } from "../format";
import type { MetricDef } from "../types";

describe("formatMetricLabel", () => {
  it("includes unit when provided", () => {
    const metric: MetricDef = {
      name: "tokens",
      description: null,
      unit: "count",
      physicalType: "BIGINT",
    };

    expect(formatMetricLabel(metric)).toBe("tokens (count)");
  });

  it("falls back to name without unit", () => {
    const metric: MetricDef = {
      name: "latency",
      description: null,
      unit: null,
      physicalType: "DOUBLE",
    };

    expect(formatMetricLabel(metric)).toBe("latency");
  });
});
