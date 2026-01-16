import * as vg from "@uwdata/vgplot";

/** Build a candlestick plot from metric_candles and component_edge_xy tables. */
export function buildCandlesPlot(metricLabel: string, linksAvailable: boolean): HTMLElement {
  const marks = [] as Array<ReturnType<typeof vg.ruleX> | ReturnType<typeof vg.link>>;
  if (linksAvailable) {
    marks.push(
      vg.link(vg.from("component_edge_xy"), {
        x1: "x1",
        y1: "y1",
        x2: "x2",
        y2: "y2",
        strokeOpacity: 0.35,
      })
    );
  }
  marks.push(
    vg.ruleX(vg.from("metric_candles"), {
      x: "x",
      y1: "low",
      y2: "high",
      stroke: "component_id",
      strokeOpacity: 0.7,
    })
  );
  marks.push(
    vg.ruleX(vg.from("metric_candles"), {
      x: "x",
      y1: "open",
      y2: "close",
      stroke: "component_id",
      strokeWidth: 4,
    })
  );
  return vg.plot(...marks, vg.xLabel("Commit time"), vg.yLabel(metricLabel));
}
