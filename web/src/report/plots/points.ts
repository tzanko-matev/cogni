import * as vg from "@uwdata/vgplot";

/** Build a points plot from the metric_points temp view. */
export function buildPointsPlot(metricLabel: string, edgesAvailable: boolean): HTMLElement {
  const marks = [] as Array<ReturnType<typeof vg.dot> | ReturnType<typeof vg.link>>;
  if (edgesAvailable) {
    marks.push(
      vg.link(vg.from("edge_xy"), {
        x1: "x1",
        y1: "y1",
        x2: "x2",
        y2: "y2",
        strokeOpacity: 0.25,
      })
    );
  }
  marks.push(
    vg.dot(vg.from("metric_points"), {
      x: "ts",
      y: "value",
      r: 4,
      fill: "#ff7a59",
      stroke: "#1d1a16",
    })
  );
  return vg.plot(...marks, vg.xLabel("Commit time"), vg.yLabel(metricLabel));
}
