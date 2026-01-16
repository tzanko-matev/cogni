import * as vg from "@uwdata/vgplot";

/** Build a points plot from the `metric_points` temp view. */
export function buildPointsPlot(metricLabel: string): HTMLElement {
  return vg.plot(
    vg.dot(vg.from("metric_points"), {
      x: "ts",
      y: "value",
      r: 4,
      fill: "#ff7a59",
      stroke: "#1d1a16",
    }),
    vg.xLabel("Commit time"),
    vg.yLabel(metricLabel)
  );
}
