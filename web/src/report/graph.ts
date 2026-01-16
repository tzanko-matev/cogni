import type { EdgeXY, MetricPoint, ParentEdge } from "./types";

/** Build a parent adjacency map from revision parent rows. */
export function buildParentMap(edges: ParentEdge[]): Map<string, string[]> {
  const map = new Map<string, string[]>();
  edges.forEach((edge) => {
    const existing = map.get(edge.childRevId) ?? [];
    existing.push(edge.parentRevId);
    map.set(edge.childRevId, existing);
  });
  return map;
}

/**
 * Compute minimal ancestor edges between measured revisions.
 */
export function computeMinimalEdges(points: MetricPoint[], parentEdges: ParentEdge[]): Array<{ parentRevId: string; childRevId: string }> {
  const measured = new Set(points.map((point) => point.revId));
  const parentMap = buildParentMap(parentEdges);
  const memo = new Map<string, Set<string>>();
  const visiting = new Set<string>();

  const nearestMeasuredAncestors = (revId: string): Set<string> => {
    const cached = memo.get(revId);
    if (cached) {
      return cached;
    }
    if (visiting.has(revId)) {
      return new Set();
    }
    visiting.add(revId);
    const result = new Set<string>();
    const parents = parentMap.get(revId) ?? [];
    parents.forEach((parent) => {
      if (measured.has(parent)) {
        result.add(parent);
      } else {
        nearestMeasuredAncestors(parent).forEach((ancestor) => result.add(ancestor));
      }
    });
    visiting.delete(revId);
    memo.set(revId, result);
    return result;
  };

  const edges: Array<{ parentRevId: string; childRevId: string }> = [];
  points.forEach((point) => {
    nearestMeasuredAncestors(point.revId).forEach((parent) => {
      edges.push({ parentRevId: parent, childRevId: point.revId });
    });
  });

  return edges;
}

/**
 * Build link coordinates for minimal edges.
 */
export function buildEdgeXY(points: MetricPoint[], edges: Array<{ parentRevId: string; childRevId: string }>): EdgeXY[] {
  const pointByRev = new Map(points.map((point) => [point.revId, point]));
  return edges.flatMap((edge) => {
    const parent = pointByRev.get(edge.parentRevId);
    const child = pointByRev.get(edge.childRevId);
    if (!parent || !child) {
      return [];
    }
    return [
      {
        x1: parent.ts,
        y1: parent.value,
        x2: child.ts,
        y2: child.value,
      },
    ];
  });
}
