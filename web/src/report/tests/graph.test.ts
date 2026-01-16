import { describe, expect, it } from "vitest";
import { buildEdgeXY, computeMinimalEdges } from "../graph";
import type { MetricPoint, ParentEdge } from "../types";

describe("computeMinimalEdges", () => {
  it("stops at the first measured ancestor", () => {
    const points: MetricPoint[] = [
      { repoId: "r1", revId: "A", ts: new Date("2024-01-01T00:00:00Z"), value: 1 },
      { repoId: "r1", revId: "C", ts: new Date("2024-01-03T00:00:00Z"), value: 3 },
    ];
    const parents: ParentEdge[] = [
      { childRevId: "B", parentRevId: "A" },
      { childRevId: "C", parentRevId: "B" },
    ];

    const edges = computeMinimalEdges(points, parents);
    expect(edges).toEqual([{ parentRevId: "A", childRevId: "C" }]);
  });

  it("builds edge XY from points", () => {
    const points: MetricPoint[] = [
      { repoId: "r1", revId: "A", ts: new Date("2024-01-01T00:00:00Z"), value: 1 },
      { repoId: "r1", revId: "B", ts: new Date("2024-01-02T00:00:00Z"), value: 2 },
    ];
    const edges = [{ parentRevId: "A", childRevId: "B" }];

    const edgeXY = buildEdgeXY(points, edges);
    expect(edgeXY).toHaveLength(1);
    expect(edgeXY[0].y1).toBe(1);
    expect(edgeXY[0].y2).toBe(2);
  });
});
