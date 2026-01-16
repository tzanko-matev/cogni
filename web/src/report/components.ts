import { bucketKey } from "./graph";
import type { BucketSize, Candle, ComponentEdgeXY, MetricPoint } from "./types";

/**
 * Compute connected components for a bucket size.
 */
export function computeComponents(
  points: MetricPoint[],
  edges: Array<{ parentRevId: string; childRevId: string }>,
  bucketSize: BucketSize
): { componentByRev: Map<string, string>; components: Array<{ bucket: string; componentId: string; revIds: string[] }> } {
  const bucketByRev = new Map(points.map((point) => [point.revId, bucketKey(point.ts, bucketSize)]));
  const adjacency = new Map<string, Set<string>>();
  edges.forEach((edge) => {
    const parentBucket = bucketByRev.get(edge.parentRevId);
    const childBucket = bucketByRev.get(edge.childRevId);
    if (!parentBucket || parentBucket !== childBucket) {
      return;
    }
    const parentSet = adjacency.get(edge.parentRevId) ?? new Set();
    parentSet.add(edge.childRevId);
    adjacency.set(edge.parentRevId, parentSet);
    const childSet = adjacency.get(edge.childRevId) ?? new Set();
    childSet.add(edge.parentRevId);
    adjacency.set(edge.childRevId, childSet);
  });

  const componentByRev = new Map<string, string>();
  const components: Array<{ bucket: string; componentId: string; revIds: string[] }> = [];
  const indexByBucket = new Map<string, number>();

  points.forEach((point) => {
    if (componentByRev.has(point.revId)) {
      return;
    }
    const bucket = bucketByRev.get(point.revId) ?? "unknown";
    const queue = [point.revId];
    const revIds: string[] = [];
    componentByRev.set(point.revId, "pending");

    while (queue.length > 0) {
      const current = queue.shift();
      if (!current) {
        continue;
      }
      revIds.push(current);
      const neighbors = adjacency.get(current);
      neighbors?.forEach((neighbor) => {
        if (componentByRev.has(neighbor)) {
          return;
        }
        if (bucketByRev.get(neighbor) !== bucket) {
          return;
        }
        componentByRev.set(neighbor, "pending");
        queue.push(neighbor);
      });
    }

    const nextIndex = (indexByBucket.get(bucket) ?? 0) + 1;
    indexByBucket.set(bucket, nextIndex);
    const componentId = `${bucket}:${nextIndex}`;
    revIds.forEach((revId) => componentByRev.set(revId, componentId));
    components.push({ bucket, componentId, revIds });
  });

  return { componentByRev, components };
}

/** Compute OHLC candles for each component. */
export function computeCandles(points: MetricPoint[], components: Array<{ bucket: string; componentId: string; revIds: string[] }>): Candle[] {
  const pointByRev = new Map(points.map((point) => [point.revId, point]));
  return components.map((component) => {
    const componentPoints = component.revIds
      .map((revId) => pointByRev.get(revId))
      .filter((point): point is MetricPoint => Boolean(point));
    const sorted = [...componentPoints].sort((a, b) => a.ts.getTime() - b.ts.getTime());
    const open = sorted[0]?.value ?? 0;
    const close = sorted[sorted.length - 1]?.value ?? 0;
    const low = Math.min(...componentPoints.map((point) => point.value));
    const high = Math.max(...componentPoints.map((point) => point.value));
    const minTs = sorted[0]?.ts ?? new Date(0);
    const maxTs = sorted[sorted.length - 1]?.ts ?? new Date(0);
    const mid = new Date((minTs.getTime() + maxTs.getTime()) / 2);
    return {
      bucket: component.bucket,
      componentId: component.componentId,
      x: mid,
      open,
      close,
      low,
      high,
    };
  });
}

/** Compute component-to-component links across buckets. */
export function computeComponentLinks(
  edges: Array<{ parentRevId: string; childRevId: string }>,
  componentByRev: Map<string, string>,
  candles: Candle[]
): ComponentEdgeXY[] {
  const candleByComponent = new Map(candles.map((candle) => [candle.componentId, candle]));
  const seen = new Set<string>();
  const links: ComponentEdgeXY[] = [];
  edges.forEach((edge) => {
    const from = componentByRev.get(edge.parentRevId);
    const to = componentByRev.get(edge.childRevId);
    if (!from || !to || from === to) {
      return;
    }
    const key = `${from}->${to}`;
    if (seen.has(key)) {
      return;
    }
    const fromCandle = candleByComponent.get(from);
    const toCandle = candleByComponent.get(to);
    if (!fromCandle || !toCandle) {
      return;
    }
    seen.add(key);
    links.push({
      x1: fromCandle.x,
      y1: (fromCandle.open + fromCandle.close) / 2,
      x2: toCandle.x,
      y2: (toCandle.open + toCandle.close) / 2,
    });
  });
  return links;
}
