import * as vg from "@uwdata/vgplot";

/** Build a simple points plot from the `points` table. */
export function buildPointsPlot(): HTMLElement {
  return vg.plot(
    vg.dot(vg.from("points"), {
      x: "ts",
      y: "value",
      r: 4,
      fill: "#ff7a59",
      stroke: "#1d1a16",
    }),
    vg.xLabel("Commit time"),
    vg.yLabel("Metric value")
  );
}
