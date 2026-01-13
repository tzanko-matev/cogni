## Memo: Generalized Line Chart for Cogni Reporting — Feasibility with Mosaic/vgplot

**Date:** 2026-01-14
**Audience:** Cogni reporting / visualization implementers
**Status:** Research synthesis + proposed implementation approach

---

### 1) Task summary

We want to implement a core reporting visualization in Cogni: a **generalized line chart over a commit graph**.

**Base visualization semantics**

* **X axis:** time (timestamp associated with each git commit / jj change).
* **Y axis:** measured metric value.
* **Vertices:** each measured commit/change is plotted as a **dot** at `(ts, value)`.
* **Edges:** draw a line between commits **A → B** iff:

  * A is an ancestor of B, and
  * there is **no measured** commit C such that A is ancestor of C and C is ancestor of B
    (i.e., edges are the “minimal” edges over the induced subgraph of measured nodes; effectively a transitive reduction restricted to measured nodes).

**High-density special case**

* When there are many commits (points too close in time), group by time bucket (initial idea: **per day**).
* For each day:

  1. take all measured commits that fall on that day,
  2. split them into **connected components** (with connectivity defined on the measured-commit graph for that day),
  3. draw **one candlestick (open/close/high/low)** per component, with distinct component colors,
  4. draw thin lines connecting “consecutive components” to show linkage across components/days, using a representative “vertex” per component (e.g., midpoint).

---

### 2) Key findings from Mosaic / vgplot documentation

#### Mosaic architecture is a fit for interactive, database-backed plotting

* Mosaic is a framework for linking interactive visual components (plots, tables, widgets) while leveraging a database (notably DuckDB) for scalable processing; Mosaic clients publish queries that a coordinator manages and can optimize. ([UW Interactive Data Lab][1])
* The documentation explicitly emphasizes scaling to **millions and even billions** of data points via database-backed execution, caching, and (where possible) automatic query optimization / pre-aggregation (“materialized views”). ([UW Interactive Data Lab][1])
* Mosaic is described as **an active research project** and the docs state it is **not yet “production-ready.”** ([UW Interactive Data Lab][1])

#### vgplot can draw the marks we need (dots, links, candlestick-like rules)

* vgplot marks are “graphical primitives” that act as chart layers; **each mark is a Mosaic client** that produces queries for needed data, with encoding channels (`x`, `y`, `fill`, `stroke`, …). ([UW Interactive Data Lab][2])
* Basic marks (including `dot`, `rect`, `rule`, etc.) are documented as mirroring Observable Plot counterparts. ([UW Interactive Data Lab][2])
* The vgplot **Marks API** includes:

  * `link`: draws straight lines between `[x1,y1]` and `[x2,y2]` ([UW Interactive Data Lab][3])
  * `ruleX` / `ruleY`: rule marks for vertical/horizontal segments ([UW Interactive Data Lab][3])
  * `line` / `area`: connected marks with optional M4 optimization ([UW Interactive Data Lab][3])

#### Candlestick (OHLC) rendering is directly supported via rule marks

* Observable Plot’s rule mark supports `y1` / `y2` (for `ruleX`) and can be used to encode low/high ranges; Plot’s docs include a **candlestick example** drawn using two `ruleX` layers (low→high and open→close with thicker stroke). ([Observable][4])
* Because vgplot basic marks mirror Observable Plot and use its semantics, the same candlestick construction pattern should carry over using `vg.ruleX`. ([UW Interactive Data Lab][2])

#### Declarative specs (JSON/YAML) are available and portable

* Mosaic supports declarative application specs via `mosaic-spec` as **JSON or YAML**, and the docs show these specs can generate JS code / running apps; the Jupyter widget uses this format to pass specs to the browser. ([UW Interactive Data Lab][5])

#### Handling “lots of points”: M4 exists, but our main reduction is structural

* vgplot can apply **M4 optimization** to line/area marks to reduce samples “to only a few sample points per pixel,” maintaining perceptual fidelity. This is particularly useful for large time series overview plots. ([UW Interactive Data Lab][1])
* However, our generalized chart is a **DAG of dots+links**, not a single series line: M4 helps if we add an overview line/area mode, but it does not replace our planned **time-bucket + component + candle** aggregation strategy.

#### Important limitation: vgplot won’t compute ancestry / components for us

* vgplot/Plot describes *rendering* + query-backed aggregation; it does not provide git-ancestry or graph algorithms out of the box. We should plan to **precompute**:

  * the “minimal edges” between measured commits, and
  * the per-day connected components and per-component OHLC stats,
    and then treat those as tables that vgplot queries.

---

### 3) Proposed data model (what Cogni should produce)

To use vgplot cleanly, we should emit a small set of tables (Parquet/Arrow/DuckDB tables, etc.):

#### A) Commit measurements

`measurements(id, ts, value, …)`

* `id`: commit hash / jj change id
* `ts`: chosen timestamp (author date? committer date? merge-aware time?)
* `value`: metric value

#### B) Minimal edges between measured commits

`edges(parent_id, child_id)`

* Already filtered to satisfy the “no measured C between A and B” rule.

To draw, we typically join edges to measurements to get endpoint coordinates:
`edge_xy(parent_id, child_id, x1, y1, x2, y2)`.

#### C) Daily component mapping (for dense views)

`commit_components(id, day, component_id)`

* `day`: date bucket (UTC day or repo-local day — must be defined)
* `component_id`: id of connected component within that day’s measured-commit subgraph

#### D) Component candles (OHLC per component)

`candles(day, component_id, x, open, close, low, high)`

* `x`: representative x-position (e.g., min ts / median ts / midpoint ts) for drawing the candle and for connecting to other components
* open/close: values of the earliest/latest commit in that component by `ts`
* low/high: min/max values within the component

#### E) Inter-component “thin link” edges (optional but matches your design)

`component_edges(from_day, from_component_id, to_day, to_component_id)`

* Precomputed based on how you define “consecutive components that are linked.”

---

### 4) Rendering approach in vgplot

#### A) Base commit graph view (dots + minimal ancestry links)

* Use `vg.dot` for measured commits.
* Use `vg.link` for the minimal edges (`edge_xy`).

This is supported directly by vgplot’s mark set (dot + link). ([UW Interactive Data Lab][3])

**Code sketch (JS API)**

```js
import * as vg from "@uwdata/vgplot";

await vg.coordinator().exec([
  vg.loadParquet("measurements", "commit-metrics.parquet"),
  vg.loadParquet("edges",        "commit-edges.parquet"),
  `
  CREATE VIEW edge_xy AS
  SELECT
    e.parent_id, e.child_id,
    p.ts    AS x1, p.value AS y1,
    c.ts    AS x2, c.value AS y2
  FROM edges e
  JOIN measurements p ON p.id = e.parent_id
  JOIN measurements c ON c.id = e.child_id
  `
]);

export default vg.plot(
  vg.link(vg.from("edge_xy"), {x1:"x1", y1:"y1", x2:"x2", y2:"y2", strokeOpacity: 0.25}),
  vg.dot(vg.from("measurements"), {x:"ts", y:"value", r: 2}),
  vg.xLabel("Time"),
  vg.yLabel("Metric")
);
```

**Why this works in Mosaic terms**

* Marks are query-backed “clients”; you keep all heavy lifting (joins/aggregation) in DuckDB. ([UW Interactive Data Lab][2])

#### B) Dense mode: per-day connected components → component candles

Candlestick construction is typically:

* wick: `ruleX` from `low` to `high`
* body: `ruleX` from `open` to `close` with thicker stroke

Observable Plot’s docs show this candlestick pattern explicitly, using `ruleX` with `y1`/`y2`. ([Observable][4])
vgplot includes `ruleX` and mirrors Plot semantics. ([UW Interactive Data Lab][2])

**Code sketch (JS API)**

```js
export default vg.plot(
  // thin links between component “vertices” (optional)
  vg.link(vg.from("component_links"), {x1:"x1", y1:"y1", x2:"x2", y2:"y2", strokeOpacity: 0.35}),

  // wick
  vg.ruleX(vg.from("candles"), {x:"x", y1:"low",  y2:"high", stroke:"component_id", strokeOpacity: 0.7}),

  // body
  vg.ruleX(vg.from("candles"), {x:"x", y1:"open", y2:"close", stroke:"component_id", strokeWidth: 4}),

  vg.xLabel("Time"),
  vg.yLabel("Metric")
);
```

---

### 5) How this fits Cogni reporting output formats

If Cogni prefers to output **declarative chart specs** (rather than JS), Mosaic supports JSON/YAML via `mosaic-spec` and can generate JS / run directly in supported clients (including the Jupyter widget). ([UW Interactive Data Lab][5])

This suggests a viable path where Cogni emits:

* a bundle of data files (Parquet),
* and a `mosaic-spec` YAML/JSON file describing the plot layers + filters,
  as a self-contained “report artifact”.

---

### 6) Key risks / caveats to track

1. **Graph algorithms are on us.**
   Mosaic/vgplot provides scalable query-backed visualization, but it does not provide git ancestry traversal, transitive reduction over measured commits, or connected components per day. Plan to precompute those tables in Cogni or via a preprocessing step.

2. **Interactive filtering requires database-backed data sources.**
   vgplot can render marks from explicit arrays, but the docs warn that interactive filtering is not supported if you bypass the database and pass data directly to a mark. ([UW Interactive Data Lab][2])

3. **Production readiness.**
   Mosaic’s docs explicitly state it is not yet “production-ready” and may have bugs / documentation gaps. If Cogni will ship this as a core feature, we should treat Mosaic as a candidate dependency to evaluate carefully (version pinning, fallback rendering, etc.). ([UW Interactive Data Lab][1])

4. **Rendering cost of many links.**
   Even if data querying scales, drawing *many* `link` segments in SVG may become heavy. Our daily-component candle fallback reduces visual density and should also reduce rendered primitives.

5. **Time semantics need to be pinned down.**
   “Reasonable timestamp” must be consistent (author vs committer date; timezone; merge commits). This affects day-binning and “open/close” selection.

---

### 7) Recommended next steps for Cogni

1. **Prototype with a real repo**

   * Create a minimal dataset: `measurements` + `edges` for a few thousand commits.
   * Render the base dot+link graph in a Mosaic example page or Observable Framework.

2. **Implement dense-mode pipeline**

   * Define day bucketing rules.
   * Implement per-day connected components.
   * Emit `candles` and `component_edges` and test candlestick rendering with `ruleX`.

3. **Decide on report packaging**

   * JS-driven vgplot app vs `mosaic-spec` YAML/JSON output for portability. ([UW Interactive Data Lab][5])

4. **Define “consecutive components” precisely**

   * Document how you build `component_edges` so the thin lines encode meaningful lineage (and don’t mislead).

---

### References consulted

* Mosaic documentation: “What is Mosaic?” ([UW Interactive Data Lab][1])
* Mosaic documentation: “Why Mosaic?” ([UW Interactive Data Lab][6])
* Mosaic documentation: “Mosaic vgplot” (marks as clients; DB-bypass warning; Plot semantics) ([UW Interactive Data Lab][2])
* Mosaic vgplot API reference: “Marks” (link, ruleX, line M4) ([UW Interactive Data Lab][3])
* Mosaic documentation: “Mosaic Declarative Specifications” (mosaic-spec JSON/YAML) ([UW Interactive Data Lab][5])
* Observable Plot documentation: “Rule mark” (y1/y2 + candlestick pattern using `ruleX`) ([Observable][4])
* Observable Framework docs: “Mosaic vgplot” (vgplot built on Observable Plot; DuckDB-backed; browser init example) ([observablehq.observablehq.cloud][7])

[1]: https://idl.uw.edu/mosaic/what-is-mosaic/ "What is Mosaic? | Mosaic"
[2]: https://idl.uw.edu/mosaic/vgplot/ "Mosaic vgplot | Mosaic"
[3]: https://idl.uw.edu/mosaic/api/vgplot/marks.html "Marks | Mosaic"
[4]: https://observablehq.com/plot/marks/rule "Rule mark | Plot"
[5]: https://idl.uw.edu/mosaic/spec/ "Mosaic Declarative Specifications | Mosaic"
[6]: https://idl.uw.edu/mosaic/why-mosaic/ "Why Mosaic? | Mosaic"
[7]: https://observablehq.observablehq.cloud/framework/lib/mosaic "Mosaic vgplot  | Observable Framework"
