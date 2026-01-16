# Report UI Design (v1)

This document defines the UI layout and user-visible behavior.

## Layout

Top to bottom inside the existing `report-shell`:

1) **Header** (existing): title + status pill.
2) **Controls row** (new):
   - Metric selector (dropdown).
   - View toggle (segmented buttons: Points | Candles).
3) **Details line** (existing): status / warnings / empty-state messages.
4) **Chart panel** (existing): vgplot renders here.

Keep the look aligned with the existing palette and typography in
`web/src/style.css`.

## Controls

### Metric selector

- Default to the first numeric metric (alphabetical by name) if no previous
  selection exists.
- Show the metric name; optionally show unit or description in a tooltip or
  secondary text.
- Only include metrics where `physical_type IN ('DOUBLE','BIGINT')`.

### View toggle

- Two options: **Points** and **Candles**.
- Default to **Points**.
- Switching view re-renders the chart without reloading the DB.

## Chart behavior

### Point view (dots + edges)

- Plot each measured commit as a dot at `(ts, value)`.
- If `v_edges` is available:
  - draw links between `x1,y1` and `x2,y2` with low opacity.
- If `v_edges` is missing:
  - render dots only and show a warning in the details line.

### Candlestick view

- Use `ruleX` for the wick (low -> high).
- Use `ruleX` for the body (open -> close) with thicker stroke.
- Color by component id (or lineage id, if present).
- Candles are grouped by a **bucket window** (v1: UTC day). The computation is
  client-side, so future week/month grouping only requires changing the bucket
  size and recomputing components.
- If component links are computed (requires `revision_parents`):
  - draw thin links between component midpoints.
- If candles are missing or empty:
  - show an empty-state message in the details line and clear the chart.

## Status + empty states

- **Loading:** status pill shows "Loading"; details explain what is loading.
- **Ready:** status pill shows "Ready"; details show the current metric + view.
- **Empty:** if no numeric metrics, show "No numeric metrics found" and keep
  the chart empty.
- **Missing tables:** mention which view data is unavailable (edges/candles).
- **Error:** status pill shows "Error"; details show a concise error message.

## Accessibility

- Metric selector and view toggle must be keyboard accessible.
- Ensure focus outlines are visible with the existing palette.
- Avoid color-only encoding for warnings; include text in the details line.

Next: `integration.md`
