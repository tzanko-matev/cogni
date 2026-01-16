import * as vg from "@uwdata/vgplot";
import { buildPointsPlot } from "./plots/points";
import { attachReportDatabase, createSamplePointsView, initDuckDB } from "./duckdb";
import { buildShell, clearChart, setDetails, setStatus } from "./ui";

const DB_URL = "/data/db.duckdb";

/**
 * Bootstrap the report UI and render a placeholder plot.
 */
export async function bootstrapReport(): Promise<void> {
  const ui = buildShell();
  try {
    setStatus(ui.status, "loading", "Loading DuckDB");
    setDetails(ui.details, "Starting DuckDB-WASM and loading the report database.");

    const db = await initDuckDB();
    setStatus(ui.status, "loading", "Loading data");
    setDetails(ui.details, "Fetching /data/db.duckdb and preparing the points view.");

    const conn = await attachReportDatabase(db, DB_URL);
    await createSamplePointsView(conn);

    const coordinator = vg.coordinator();
    coordinator.databaseConnector(vg.wasmConnector({ duckdb: db }));

    clearChart(ui.chart);
    ui.chart.appendChild(buildPointsPlot());

    setStatus(ui.status, "ready", "Ready");
    setDetails(ui.details, "Hello world report rendered from v_points.");
  } catch (error: unknown) {
    setStatus(ui.status, "error", "Error");
    const message = error instanceof Error ? error.message : "Unknown error.";
    setDetails(ui.details, message);
    throw error;
  }
}
