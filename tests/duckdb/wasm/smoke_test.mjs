import { Worker } from "node:worker_threads";
import { readFile } from "node:fs/promises";
import path from "node:path";
import * as duckdb from "@duckdb/duckdb-wasm";

const dbPath = process.argv[2];
if (!dbPath) {
  console.error("usage: node smoke_test.mjs <duckdb-file>");
  process.exit(1);
}

const absolutePath = path.resolve(dbPath);
const buffer = await readFile(absolutePath);

const bundles = duckdb.getJsDelivrBundles();
const bundle = await duckdb.selectBundle(bundles);
const worker = new Worker(bundle.mainWorker);
const logger = new duckdb.ConsoleLogger();
const db = new duckdb.AsyncDuckDB(logger, worker);

try {
  await db.instantiate(bundle.mainModule, bundle.pthreadWorker);
  await db.registerFileBuffer("cogni.duckdb", buffer);
  const conn = await db.connect();
  await conn.query("SELECT COUNT(*) FROM v_points");
  await conn.query("SELECT spec->>'$.model' FROM agents LIMIT 1");
  await conn.close();
} catch (err) {
  console.error("duckdb wasm smoke test failed:", err);
  process.exitCode = 1;
} finally {
  await db.terminate();
  await worker.terminate();
}
