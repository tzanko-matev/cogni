import type { BucketSize, MetricDef, StatusLevel, ViewMode } from "./types";

/** Handles for updating the report UI. */
export interface UIHandles {
  root: HTMLElement;
  status: HTMLElement;
  details: HTMLElement;
  chart: HTMLElement;
  metricSelect: HTMLSelectElement;
  pointsButton: HTMLButtonElement;
  candlesButton: HTMLButtonElement;
  bucketSelect: HTMLSelectElement;
}

/** Build the report shell and control surface. */
export function buildShell(): UIHandles {
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

  const controls = document.createElement("div");
  controls.className = "report-controls";

  const metricGroup = document.createElement("label");
  metricGroup.className = "control";
  metricGroup.textContent = "Metric";

  const metricSelect = document.createElement("select");
  metricSelect.className = "control-select";
  metricSelect.name = "metric";
  metricSelect.setAttribute("aria-label", "Metric selector");
  metricGroup.appendChild(metricSelect);

  const viewGroup = document.createElement("div");
  viewGroup.className = "control";

  const viewLabel = document.createElement("span");
  viewLabel.className = "control-label";
  viewLabel.textContent = "View";

  const viewToggle = document.createElement("div");
  viewToggle.className = "control-toggle";

  const pointsButton = document.createElement("button");
  pointsButton.type = "button";
  pointsButton.dataset.view = "points";
  pointsButton.textContent = "Points";

  const candlesButton = document.createElement("button");
  candlesButton.type = "button";
  candlesButton.dataset.view = "candles";
  candlesButton.textContent = "Candles";

  viewToggle.append(pointsButton, candlesButton);
  viewGroup.append(viewLabel, viewToggle);

  const bucketGroup = document.createElement("label");
  bucketGroup.className = "control";
  bucketGroup.textContent = "Bucket";

  const bucketSelect = document.createElement("select");
  bucketSelect.className = "control-select";
  bucketSelect.name = "bucket";
  bucketSelect.setAttribute("aria-label", "Bucket size selector");
  bucketGroup.appendChild(bucketSelect);

  controls.append(metricGroup, viewGroup, bucketGroup);

  const details = document.createElement("p");
  details.className = "details";
  details.textContent = "Waiting to load report data.";

  const chart = document.createElement("div");
  chart.className = "chart";
  chart.id = "chart";

  shell.append(header, controls, details, chart);
  root.appendChild(shell);

  return { root, status, details, chart, metricSelect, pointsButton, candlesButton, bucketSelect };
}

/** Update the status pill. */
export function setStatus(target: HTMLElement, level: StatusLevel, message: string): void {
  target.dataset.level = level;
  target.textContent = message;
}

/** Update the details line text. */
export function setDetails(target: HTMLElement, message: string): void {
  target.textContent = message;
}

/** Populate the metric selector options. */
export function setMetricOptions(select: HTMLSelectElement, metrics: MetricDef[], selected: string | null): void {
  select.innerHTML = "";
  metrics.forEach((metric) => {
    const option = document.createElement("option");
    option.value = metric.name;
    option.textContent = metric.name;
    if (selected && metric.name === selected) {
      option.selected = true;
    }
    select.appendChild(option);
  });
  select.disabled = metrics.length === 0;
}

/** Populate bucket size options. */
export function setBucketOptions(select: HTMLSelectElement, selected: BucketSize): void {
  const options: Array<{ value: BucketSize; label: string }> = [
    { value: "day", label: "Day" },
    { value: "week", label: "Week" },
    { value: "month", label: "Month" },
  ];
  select.innerHTML = "";
  options.forEach((option) => {
    const element = document.createElement("option");
    element.value = option.value;
    element.textContent = option.label;
    if (option.value === selected) {
      element.selected = true;
    }
    select.appendChild(element);
  });
}

/** Toggle the active view button state. */
export function setViewMode(pointsButton: HTMLButtonElement, candlesButton: HTMLButtonElement, view: ViewMode): void {
  const isPoints = view === "points";
  pointsButton.classList.toggle("active", isPoints);
  pointsButton.setAttribute("aria-pressed", String(isPoints));
  candlesButton.classList.toggle("active", !isPoints);
  candlesButton.setAttribute("aria-pressed", String(!isPoints));
}

/** Enable or disable candle-only controls. */
export function setCandlesEnabled(candlesButton: HTMLButtonElement, bucketSelect: HTMLSelectElement, enabled: boolean): void {
  candlesButton.disabled = !enabled;
  bucketSelect.disabled = !enabled;
}

/** Remove any existing plot from the chart container. */
export function clearChart(target: HTMLElement): void {
  target.innerHTML = "";
}
