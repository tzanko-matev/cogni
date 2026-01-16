import * as duckdb from "@duckdb/duckdb-wasm";
import { tableFromArrays } from "apache-arrow";
import { buildMetricPointsSelectSQL, buildMetricPointsViewSQL, sqlStringLiteral } from "./sql";
import { parseMetricDefRows, parseMetricPointRows, parseParentEdgeRows } from "./types";
import type { Candle, ComponentEdgeXY, EdgeXY } from "./types";

const DB_FILE_NAME = "cogni.duckdb";
const DUCKDB_BUNDLES = duckdb.getJsDelivrBundles();

/** Initialize DuckDB-WASM and return the async instance. */
export async function initDuckDB(): Promise<duckdb.AsyncDuckDB> {
  const bundle = await duckdb.selectBundle(DUCKDB_BUNDLES);
  const workerScript = `importScripts("${bundle.mainWorker}");`;
  const workerBlob = new Blob([workerScript], { type: "text/javascript" });
  const worker = new Worker(URL.createObjectURL(workerBlob));
  const logger = new duckdb.ConsoleLogger();
  const db = new duckdb.AsyncDuckDB(logger, worker);
  await db.instantiate(bundle.mainModule, bundle.pthreadWorker);
  return db;
}

/**
 * Attach the report database and return a live connection.
 */
export async function attachReportDatabase(
  db: duckdb.AsyncDuckDB,
  url: string
): Promise<duckdb.AsyncDuckDBConnection> {
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`Failed to fetch DuckDB file: ${response.status} ${response.statusText}`);
  }
  const buffer = new Uint8Array(await response.arrayBuffer());
  await db.registerFileBuffer(DB_FILE_NAME, buffer);

  const conn = await db.connect();
  await conn.query(`ATTACH '${DB_FILE_NAME}' AS cogni (READ_ONLY)`);
  return conn;
}

/** Query numeric metric definitions for the selector. */
export async function listNumericMetrics(conn: duckdb.AsyncDuckDBConnection) {
  const table = await conn.query(`
    SELECT name, description, unit, physical_type
    FROM cogni.main.metric_defs
    WHERE physical_type IN ('DOUBLE','BIGINT')
    ORDER BY name
  `);
  return parseMetricDefRows(table.toArray());
}

/** Check whether revision_parents exists in the report DB. */
export async function hasRevisionParents(conn: duckdb.AsyncDuckDBConnection): Promise<boolean> {
  const table = await conn.query(`
    SELECT COUNT(*) AS count
    FROM information_schema.tables
    WHERE table_schema = 'main'
      AND table_name = 'revision_parents'
  `);
  const row = table.toArray()[0] as { count?: number } | undefined;
  return Boolean(row && row.count && row.count > 0);
}

/** Create or replace the metric_points temp view. */
export async function createMetricPointsView(
  conn: duckdb.AsyncDuckDBConnection,
  metric: string
): Promise<void> {
  await conn.query(buildMetricPointsViewSQL(metric));
}

/** Fetch metric points for the current selection. */
export async function fetchMetricPoints(conn: duckdb.AsyncDuckDBConnection) {
  const table = await conn.query(buildMetricPointsSelectSQL());
  return parseMetricPointRows(table.toArray());
}

/** Fetch parent edges for a specific repo. */
export async function fetchRevisionParents(
  conn: duckdb.AsyncDuckDBConnection,
  repoId: string
) {
  const repoLiteral = sqlStringLiteral(repoId);
  const table = await conn.query(`
    SELECT child_rev_id, parent_rev_id
    FROM cogni.main.revision_parents
    WHERE repo_id = ${repoLiteral}
  `);
  return parseParentEdgeRows(table.toArray());
}

/** Replace the edge_xy temp table with new link data. */
export async function replaceEdgeXYTable(
  conn: duckdb.AsyncDuckDBConnection,
  edges: EdgeXY[]
): Promise<void> {
  await conn.query(
    "CREATE OR REPLACE TABLE edge_xy (x1 TIMESTAMP, y1 DOUBLE, x2 TIMESTAMP, y2 DOUBLE)"
  );
  if (edges.length === 0) {
    return;
  }
  const table = tableFromArrays({
    x1: edges.map((edge) => edge.x1),
    y1: edges.map((edge) => edge.y1),
    x2: edges.map((edge) => edge.x2),
    y2: edges.map((edge) => edge.y2),
  });
  await conn.insertArrowTable(table, { name: "edge_xy", create: false });
}

/** Replace the metric_candles temp table with new candle data. */
export async function replaceCandlesTable(
  conn: duckdb.AsyncDuckDBConnection,
  candles: Candle[]
): Promise<void> {
  await conn.query(
    "CREATE OR REPLACE TABLE metric_candles (bucket VARCHAR, component_id VARCHAR, x TIMESTAMP, open DOUBLE, close DOUBLE, low DOUBLE, high DOUBLE)"
  );
  if (candles.length === 0) {
    return;
  }
  const table = tableFromArrays({
    bucket: candles.map((candle) => candle.bucket),
    component_id: candles.map((candle) => candle.componentId),
    x: candles.map((candle) => candle.x),
    open: candles.map((candle) => candle.open),
    close: candles.map((candle) => candle.close),
    low: candles.map((candle) => candle.low),
    high: candles.map((candle) => candle.high),
  });
  await conn.insertArrowTable(table, { name: "metric_candles", create: false });
}

/** Replace the component_edge_xy temp table with link data. */
export async function replaceComponentEdgeTable(
  conn: duckdb.AsyncDuckDBConnection,
  edges: ComponentEdgeXY[]
): Promise<void> {
  await conn.query(
    "CREATE OR REPLACE TABLE component_edge_xy (x1 TIMESTAMP, y1 DOUBLE, x2 TIMESTAMP, y2 DOUBLE)"
  );
  if (edges.length === 0) {
    return;
  }
  const table = tableFromArrays({
    x1: edges.map((edge) => edge.x1),
    y1: edges.map((edge) => edge.y1),
    x2: edges.map((edge) => edge.x2),
    y2: edges.map((edge) => edge.y2),
  });
  await conn.insertArrowTable(table, { name: "component_edge_xy", create: false });
}
