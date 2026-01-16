import { describe, expect, it } from "vitest";
import { computeCandles, computeComponentLinks, computeComponents } from "../components";
import type { MetricPoint } from "../types";

describe("component aggregation", () => {
  it("groups nodes into bucket components", () => {
    const points: MetricPoint[] = [
      { repoId: "r1", revId: "A", ts: new Date("2024-01-01T10:00:00Z"), value: 1 },
      { repoId: "r1", revId: "B", ts: new Date("2024-01-01T12:00:00Z"), value: 2 },
      { repoId: "r1", revId: "C", ts: new Date("2024-01-02T09:00:00Z"), value: 3 },
    ];
    const edges = [
      { parentRevId: "A", childRevId: "B" },
      { parentRevId: "B", childRevId: "C" },
    ];

    const { components } = computeComponents(points, edges, "day");
    expect(components).toHaveLength(2);
  });

  it("computes candle OHLC values", () => {
    const points: MetricPoint[] = [
      { repoId: "r1", revId: "A", ts: new Date("2024-01-01T10:00:00Z"), value: 5 },
      { repoId: "r1", revId: "B", ts: new Date("2024-01-01T12:00:00Z"), value: 2 },
      { repoId: "r1", revId: "C", ts: new Date("2024-01-01T18:00:00Z"), value: 7 },
    ];
    const edges = [
      { parentRevId: "A", childRevId: "B" },
      { parentRevId: "B", childRevId: "C" },
    ];

    const { components } = computeComponents(points, edges, "day");
    const candles = computeCandles(points, components);
    expect(candles[0].open).toBe(5);
    expect(candles[0].close).toBe(7);
    expect(candles[0].low).toBe(2);
    expect(candles[0].high).toBe(7);
  });

  it("deduplicates component links", () => {
    const points: MetricPoint[] = [
      { repoId: "r1", revId: "A", ts: new Date("2024-01-01T10:00:00Z"), value: 1 },
      { repoId: "r1", revId: "B", ts: new Date("2024-01-02T10:00:00Z"), value: 2 },
    ];
    const edges = [
      { parentRevId: "A", childRevId: "B" },
      { parentRevId: "A", childRevId: "B" },
    ];

    const { componentByRev, components } = computeComponents(points, edges, "day");
    const candles = computeCandles(points, components);
    const links = computeComponentLinks(edges, componentByRev, candles);
    expect(links).toHaveLength(1);
  });
});
