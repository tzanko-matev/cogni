import * as duckdb from "@duckdb/duckdb-wasm";
import * as vg from "@uwdata/vgplot";
import "./style.css";

type StatusLevel = "idle" | "loading" | "ready" | "error";

interface UIHandles {
  root: HTMLElement;
  status: HTMLElement;
  details: HTMLElement;
  chart: HTMLElement;
}

const DB_FILE_NAME = "cogni.duckdb";
const DB_URL = "/data/db.duckdb";

const DUCKDB_BUNDLES = duckdb.getJsDelivrBundles();

// buildShell constructs the basic report layout in the page.
function buildShell(): UIHandles {
  const root = document.getElementById("app");
  if (!root) {
    throw new Error("Cogni report: #app container not found.");
  }

  const shell = document.createElement("main");
  shell.className = "report-shell";

  const header = document.createElement("header");
  header.className = "report-header";

  const title = document.createElement("h1");
  title.textContent = "Cogni Report";

  const status = document.createElement("span");
  status.className = "status";
  status.dataset.level = "idle";
  status.textContent = "Idle";

  header.append(title, status);

  const details = document.createElement("p");
  details.className = "details";
  details.textContent = "Waiting to load report data.";

  const chart = document.createElement("div");
  chart.className = "chart";
  chart.id = "chart";

  shell.append(header, details, chart);
  root.appendChild(shell);

  return { root, status, details, chart };
}

// setStatus updates the status pill text and state.
function setStatus(target: HTMLElement, level: StatusLevel, message: string): void {
  target.dataset.level = level;
  target.textContent = message;
}

// initDuckDB loads DuckDB-WASM and returns a ready AsyncDuckDB instance.
async function initDuckDB(): Promise<duckdb.AsyncDuckDB> {
  const bundle = await duckdb.selectBundle(DUCKDB_BUNDLES);
  const workerScript = `importScripts("${bundle.mainWorker}");`;
  const workerBlob = new Blob([workerScript], { type: "text/javascript" });
  const worker = new Worker(URL.createObjectURL(workerBlob));
  const logger = new duckdb.ConsoleLogger();
  const db = new duckdb.AsyncDuckDB(logger, worker);
  await db.instantiate(bundle.mainModule, bundle.pthreadWorker);
  return db;
}

// loadReportData registers the DuckDB file and prepares a points table.
async function loadReportData(db: duckdb.AsyncDuckDB, url: string): Promise<void> {
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`Failed to fetch DuckDB file: ${response.status} ${response.statusText}`);
  }
  const buffer = new Uint8Array(await response.arrayBuffer());
  await db.registerFileBuffer(DB_FILE_NAME, buffer);

  const conn = await db.connect();
  try {
    await conn.query(`ATTACH '${DB_FILE_NAME}' AS cogni (READ_ONLY)`);
    await conn.query(`
      CREATE OR REPLACE TABLE points AS
      SELECT ts, value
      FROM cogni.main.v_points
      WHERE value IS NOT NULL
      ORDER BY ts
      LIMIT 200
    `);
  } finally {
    await conn.close();
  }
}

// renderChart uses vgplot to render a basic dot chart from the points table.
function renderChart(db: duckdb.AsyncDuckDB, target: HTMLElement): void {
  const coordinator = vg.coordinator();
  coordinator.databaseConnector(vg.wasmConnector({ duckdb: db }));

  const plot = vg.plot(
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

  target.appendChild(plot);
}

// bootstrap initializes the report UI and data pipeline.
async function bootstrap(): Promise<void> {
  const ui = buildShell();
  try {
    setStatus(ui.status, "loading", "Loading DuckDB");
    ui.details.textContent = "Starting DuckDB-WASM and loading the report database.";

    const db = await initDuckDB();
    setStatus(ui.status, "loading", "Loading data");
    ui.details.textContent = "Fetching /data/db.duckdb and preparing the points view.";

    await loadReportData(db, DB_URL);
    setStatus(ui.status, "loading", "Rendering chart");
    ui.details.textContent = "Rendering a basic metric chart from the DuckDB file.";

    renderChart(db, ui.chart);
    setStatus(ui.status, "ready", "Ready");
    ui.details.textContent = "Hello world report rendered from v_points.";
  } catch (error: unknown) {
    setStatus(ui.status, "error", "Error");
    const message = error instanceof Error ? error.message : "Unknown error.";
    ui.details.textContent = message;
    throw error;
  }
}

bootstrap().catch(() => {});
