# Plan: Serve browser report from DuckDB (2026-01-15)

## Goal
Add a new CLI command `cogni serve <db.duckdb>` that starts a local HTTP server
(on 127.0.0.1:5000 by default) and serves a basic “hello world” report in the
browser. The report must load the DuckDB database in the browser (DuckDB-WASM)
so that all querying happens client-side. The plan must also establish the
infrastructure needed for future, more complex reports:
- a TypeScript client build pipeline,
- embedded static assets in the Go binary by default,
- and an optional external assets base URL for SaaS deployments.

## References
- spec/inbox/vgplot-research.md
- spec/features/output-report-html.feature
- spec/features/cli.feature

## Steps
1) CLI command + server skeleton
   - Add `serve` command registration in `internal/cli/cli.go`.
   - Implement `internal/cli/serve.go` with flags:
     - positional arg: `<db.duckdb>` (required)
     - `--addr` default `127.0.0.1:5000`
     - `--assets-base-url` optional (CDN/SaaS override)
   - Validate DB path exists before starting the server.
   - Start a server that responds to `/` and `/data/db.duckdb`.
   - Tests (with explicit timeouts):
     - `nix develop -c go test ./internal/cli -timeout 10s`
     - Add a unit test for flag parsing and missing DB path behavior.

2) HTTP handlers + asset resolution layer
   - Create a small `internal/reportserver` package that owns routing and HTML rendering.
   - Add an asset resolver that:
     - uses embedded assets by default, and
     - switches to `--assets-base-url` when provided.
   - Introduce an asset manifest (JSON) to map logical names to hashed files.
   - Add HTTP tests for `/` and `/data/db.duckdb`.
   - Tests (with explicit timeouts):
     - `nix develop -c go test ./internal/reportserver -timeout 10s`

3) TypeScript client build pipeline
   - Create `web/` with `package.json`, `tsconfig.json`, and build tooling.
   - Use a bundler that emits a `manifest.json` with hashed filenames (e.g., Vite).
   - Output to `web/dist` with `index.html` replaced by Go rendering (we only need JS/CSS).
   - Add `just` targets (or scripts) for `web-build` and integrate into the Go build flow.
   - Update `flake.nix` to include `nodejs` (and the chosen package manager if needed).
   - Tests:
     - `nix develop -c npm --prefix web run build` (or equivalent) and ensure `web/dist/manifest.json` exists.

4) “Hello world” report in the browser
   - TypeScript entry point:
     - load DuckDB-WASM from CDN,
     - fetch `/data/db.duckdb`,
     - query `v_points` for a small result set,
     - render a basic vgplot dot plot (or a table fallback if no data).
   - Keep the UI minimal but structured so more charts can be added later.
   - Tests:
     - Build pipeline check from Step 3 is sufficient for now.

5) BDD + docs updates
   - Add a new feature file (e.g., `spec/features/output-report-serve.feature`) describing:
     - `cogni serve <db.duckdb>` starts a server and responds with HTML.
     - The server serves `/data/db.duckdb`.
     - `--assets-base-url` switches asset URLs in the HTML.
   - Update `spec/features/cli.feature` to list the new `serve` command in help output.
   - Add Godog test harness if needed for the new feature file.
   - Tests (with explicit timeouts):
     - `nix develop -c go test ./tests/... -tags=cucumber -timeout 10s` (or the specific test package).

## Completion
Mark this plan and status file as DONE when all steps and tests are complete.

Status: IN PROGRESS
