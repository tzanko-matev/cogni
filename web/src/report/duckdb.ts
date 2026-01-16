import * as duckdb from "@duckdb/duckdb-wasm";

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

/**
 * Create a minimal points table for the initial placeholder chart.
 */
export async function createSamplePointsView(conn: duckdb.AsyncDuckDBConnection): Promise<void> {
  await conn.query(`
    CREATE OR REPLACE TABLE points AS
    SELECT ts, value
    FROM cogni.main.v_points
    WHERE value IS NOT NULL
    ORDER BY ts
    LIMIT 200
  `);
}
