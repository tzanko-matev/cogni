import * as vg from "@uwdata/vgplot";
import {
  attachReportDatabase,
  createMetricPointsView,
  fetchMetricPoints,
  hasRevisionParents,
  initDuckDB,
  listNumericMetrics,
} from "./duckdb";
import { formatMetricLabel } from "./format";
import { buildPointsPlot } from "./plots/points";
import { normalizeBucketSize, normalizeViewMode, selectMetricName } from "./state";
import type { BucketSize, ViewMode } from "./types";
import {
  buildShell,
  clearChart,
  setBucketOptions,
  setCandlesEnabled,
  setDetails,
  setMetricOptions,
  setStatus,
  setViewMode,
} from "./ui";

const DB_URL = "/data/db.duckdb";

/**
 * Bootstrap the report UI and render the initial points view.
 */
export async function bootstrapReport(): Promise<void> {
  const ui = buildShell();
  try {
    setStatus(ui.status, "loading", "Loading DuckDB");
    setDetails(ui.details, "Starting DuckDB-WASM and loading the report database.");

    const db = await initDuckDB();
    setStatus(ui.status, "loading", "Loading data");
    setDetails(ui.details, "Fetching /data/db.duckdb and preparing the report views.");

    const conn = await attachReportDatabase(db, DB_URL);

    const coordinator = vg.coordinator();
    coordinator.databaseConnector(vg.wasmConnector({ duckdb: db }));

    const metrics = await listNumericMetrics(conn);
    let selectedMetric = selectMetricName(metrics, null);
    let viewMode: ViewMode = normalizeViewMode("points");
    let bucketSize: BucketSize = normalizeBucketSize("day");

    setMetricOptions(ui.metricSelect, metrics, selectedMetric);
    setBucketOptions(ui.bucketSelect, bucketSize);
    setViewMode(ui.pointsButton, ui.candlesButton, viewMode);

    const parentsAvailable = await hasRevisionParents(conn);
    setCandlesEnabled(ui.candlesButton, ui.bucketSelect, parentsAvailable);

    if (!selectedMetric) {
      setStatus(ui.status, "ready", "Ready");
      setDetails(ui.details, "No numeric metrics found in the report.");
      return;
    }

    let renderToken = 0;
    const render = async (): Promise<void> => {
      renderToken += 1;
      const token = renderToken;

      setStatus(ui.status, "loading", "Rendering chart");
      await createMetricPointsView(conn, selectedMetric ?? "");
      const points = await fetchMetricPoints(conn);

      if (token !== renderToken) {
        return;
      }

      clearChart(ui.chart);
      const metricLabel = formatMetricLabel(
        metrics.find((metric) => metric.name === selectedMetric) ?? metrics[0]
      );
      ui.chart.appendChild(buildPointsPlot(metricLabel));

      if (points.length === 0) {
        setDetails(ui.details, `Metric: ${selectedMetric} (no data).`);
      } else {
        setDetails(
          ui.details,
          `Metric: ${selectedMetric} · View: ${viewMode === "points" ? "Points" : "Candles"}`
        );
      }
      setStatus(ui.status, "ready", "Ready");
    };

    ui.metricSelect.addEventListener("change", () => {
      const nextMetric = ui.metricSelect.value;
      if (nextMetric && nextMetric !== selectedMetric) {
        selectedMetric = nextMetric;
      }
      void render();
    });

    ui.pointsButton.addEventListener("click", () => {
      viewMode = "points";
      setViewMode(ui.pointsButton, ui.candlesButton, viewMode);
      void render();
    });

    ui.candlesButton.addEventListener("click", () => {
      if (ui.candlesButton.disabled) {
        return;
      }
      viewMode = "candles";
      setViewMode(ui.pointsButton, ui.candlesButton, viewMode);
      void render();
    });

    ui.bucketSelect.addEventListener("change", () => {
      bucketSize = normalizeBucketSize(ui.bucketSelect.value);
      if (viewMode === "candles") {
        void render();
      }
    });

    await render();
  } catch (error: unknown) {
    setStatus(ui.status, "error", "Error");
    const message = error instanceof Error ? error.message : "Unknown error.";
    setDetails(ui.details, message);
    throw error;
  }
}
