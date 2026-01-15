import { mkdir, readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { Worker } from "node:worker_threads";
import * as duckdb from "@duckdb/duckdb-wasm";

const dbPath = process.argv[2];
if (!dbPath) {
  console.error("usage: node smoke_test.mjs <duckdb-file>");
  process.exit(1);
}

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const duckdbDist = path.join(__dirname, "node_modules", "@duckdb", "duckdb-wasm", "dist");
// DuckDB-WASM tries to install extensions under $HOME/.duckdb by default.
// Redirect HOME into the repo to avoid sandboxed writes to the real home dir.
const duckdbHome = path.join(__dirname, ".duckdb_home");
const extensionDir = path.join(duckdbHome, ".duckdb", "extensions");
await mkdir(extensionDir, { recursive: true });
process.env.HOME = duckdbHome;
process.env.DUCKDB_EXTENSION_DIRECTORY = extensionDir;

const absolutePath = path.resolve(dbPath);
const buffer = await readFile(absolutePath);

const bundle = {
  mainModule: path.join(duckdbDist, "duckdb-mvp.wasm"),
  mainWorker: path.join(duckdbDist, "duckdb-node-mvp.worker.cjs"),
  pthreadWorker: null,
};
const workerEntry = path.join(duckdbDist, "duckdb-node.cjs");
const worker = wrapWorker(
  new Worker(workerEntry, {
    workerData: {
      mod: bundle.mainWorker,
    },
  }),
);
const logger = new duckdb.ConsoleLogger();
const db = new duckdb.AsyncDuckDB(logger, worker);

try {
  await db.instantiate(bundle.mainModule, bundle.pthreadWorker);
  await db.registerFileBuffer("cogni.duckdb", buffer);
  await db.open({ path: "cogni.duckdb", accessMode: duckdb.DuckDBAccessMode.READ_ONLY });
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

function wrapWorker(nodeWorker) {
  const listeners = new Map();
  const dispatch = (type, event) => {
    const handlers = listeners.get(type);
    if (!handlers) return;
    for (const handler of handlers) {
      handler(event);
    }
  };
  nodeWorker.on("message", (data) => dispatch("message", { data }));
  nodeWorker.on("error", (error) => dispatch("error", error));
  nodeWorker.on("exit", () => dispatch("close", {}));
  return {
    addEventListener(type, handler) {
      if (!listeners.has(type)) {
        listeners.set(type, new Set());
      }
      listeners.get(type).add(handler);
    },
    removeEventListener(type, handler) {
      const handlers = listeners.get(type);
      if (handlers) {
        handlers.delete(handler);
      }
    },
    postMessage(message, transfer) {
      nodeWorker.postMessage(message, transfer);
    },
    terminate() {
      return nodeWorker.terminate();
    },
  };
}
