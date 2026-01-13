const measurements = [
  { id: "a1", label: "alpha", ts: "2025-10-29T09:12:00Z", value: 72, parents: [] },
  { id: "b1", label: "beta", ts: "2025-11-02T14:02:00Z", value: 68, parents: ["a1"] },
  { id: "c1", label: "gamma", ts: "2025-11-06T09:48:00Z", value: 75, parents: ["b1"] },
  { id: "d1", label: "delta", ts: "2025-11-09T16:35:00Z", value: 63, parents: ["b1"] },
  { id: "e1", label: "epsilon", ts: "2025-11-12T08:15:00Z", value: 70, parents: ["c1", "d1"] },
  { id: "f1", label: "zeta", ts: "2025-11-16T11:10:00Z", value: 66, parents: ["e1"] },
  { id: "g1", label: "eta", ts: "2025-11-18T17:40:00Z", value: 73, parents: ["e1"] },
  { id: "h1", label: "theta", ts: "2025-11-21T10:04:00Z", value: 69, parents: ["f1", "g1"] },
  { id: "i1", label: "iota", ts: "2025-11-25T13:19:00Z", value: 78, parents: ["h1"] },
  { id: "j1", label: "kappa", ts: "2025-11-28T08:05:00Z", value: 64, parents: ["h1"] },
  { id: "k1", label: "lambda", ts: "2025-12-01T19:42:00Z", value: 71, parents: ["i1", "j1"] },
  { id: "l1", label: "mu", ts: "2025-12-05T12:30:00Z", value: 67, parents: ["k1"] },
];

const chart = document.getElementById("chart");
const width = chart.clientWidth;
const height = chart.clientHeight;
const margin = { top: 32, right: 28, bottom: 50, left: 56 };
const innerW = width - margin.left - margin.right;
const innerH = height - margin.top - margin.bottom;

const nodeById = new Map(measurements.map((node) => [node.id, node]));

// Return a set of ancestor ids for the given node id.
const ancestorCache = new Map();
function collectAncestors(id) {
  if (ancestorCache.has(id)) {
    return ancestorCache.get(id);
  }
  const node = nodeById.get(id);
  const result = new Set();
  if (node) {
    node.parents.forEach((parentId) => {
      result.add(parentId);
      const parentAncestors = collectAncestors(parentId);
      parentAncestors.forEach((ancestorId) => result.add(ancestorId));
    });
  }
  ancestorCache.set(id, result);
  return result;
}

// Build a list of edges between direct measured ancestors.
function computeEdges(nodes) {
  const edges = [];
  nodes.forEach((node) => {
    const ancestors = collectAncestors(node.id);
    ancestors.forEach((ancestorId) => {
      let isDirect = true;
      ancestors.forEach((candidateId) => {
        if (candidateId === ancestorId) {
          return;
        }
        const candidateAncestors = collectAncestors(candidateId);
        if (candidateAncestors.has(ancestorId)) {
          isDirect = false;
        }
      });
      if (isDirect) {
        edges.push({ from: ancestorId, to: node.id });
      }
    });
  });
  return edges;
}

// Map timestamps and metric values to chart coordinates.
function buildScales(nodes) {
  const times = nodes.map((node) => new Date(node.ts).getTime());
  const values = nodes.map((node) => node.value);
  const minTime = Math.min(...times);
  const maxTime = Math.max(...times);
  const minVal = Math.min(...values) - 4;
  const maxVal = Math.max(...values) + 4;
  const scaleX = (time) => margin.left + ((time - minTime) / (maxTime - minTime)) * innerW;
  const scaleY = (value) => margin.top + (1 - (value - minVal) / (maxVal - minVal)) * innerH;
  return { scaleX, scaleY, minTime, maxTime, minVal, maxVal };
}

const edges = computeEdges(measurements);
const { scaleX, scaleY, minTime, maxTime, minVal, maxVal } = buildScales(measurements);

const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
svg.setAttribute("viewBox", `0 0 ${width} ${height}`);
svg.setAttribute("width", width);
svg.setAttribute("height", height);
chart.appendChild(svg);

// Add an SVG line and return it for further styling.
function addLine(x1, y1, x2, y2, stroke, widthPx, dash) {
  const line = document.createElementNS("http://www.w3.org/2000/svg", "line");
  line.setAttribute("x1", x1);
  line.setAttribute("y1", y1);
  line.setAttribute("x2", x2);
  line.setAttribute("y2", y2);
  line.setAttribute("stroke", stroke);
  line.setAttribute("stroke-width", widthPx);
  if (dash) {
    line.setAttribute("stroke-dasharray", dash);
  }
  svg.appendChild(line);
  return line;
}

// Add a text label to the SVG.
function addText(x, y, text, anchor, size, fill) {
  const label = document.createElementNS("http://www.w3.org/2000/svg", "text");
  label.setAttribute("x", x);
  label.setAttribute("y", y);
  label.setAttribute("fill", fill);
  label.setAttribute("font-size", size);
  label.setAttribute("font-family", "IBM Plex Mono, monospace");
  label.setAttribute("text-anchor", anchor);
  label.textContent = text;
  svg.appendChild(label);
  return label;
}

// Grid lines and axes.
const gridLines = 5;
for (let i = 0; i <= gridLines; i += 1) {
  const y = margin.top + (innerH / gridLines) * i;
  addLine(margin.left, y, width - margin.right, y, "rgba(29,26,22,0.1)", 1, "4 6");
}
addLine(margin.left, margin.top, margin.left, height - margin.bottom, "#1d1a16", 2);
addLine(margin.left, height - margin.bottom, width - margin.right, height - margin.bottom, "#1d1a16", 2);

for (let i = 0; i <= gridLines; i += 1) {
  const value = minVal + ((maxVal - minVal) / gridLines) * (gridLines - i);
  const y = margin.top + (innerH / gridLines) * i + 4;
  addText(margin.left - 12, y, value.toFixed(0), "end", 12, "#6b6258");
}

const timeFormat = new Intl.DateTimeFormat("en-US", { month: "short", day: "2-digit" });
for (let i = 0; i <= 4; i += 1) {
  const time = minTime + ((maxTime - minTime) / 4) * i;
  const x = margin.left + (innerW / 4) * i;
  addText(x, height - margin.bottom + 28, timeFormat.format(new Date(time)), "middle", 12, "#6b6258");
}

addText(margin.left, margin.top - 12, "Effort score", "start", 13, "#1d1a16");
addText(width - margin.right, height - 12, "Commit time", "end", 13, "#1d1a16");

// Draw edges (direct ancestor links).
edges.forEach((edge) => {
  const from = nodeById.get(edge.from);
  const to = nodeById.get(edge.to);
  if (!from || !to) {
    return;
  }
  const x1 = scaleX(new Date(from.ts).getTime());
  const y1 = scaleY(from.value);
  const x2 = scaleX(new Date(to.ts).getTime());
  const y2 = scaleY(to.value);
  const line = addLine(x1, y1, x2, y2, "#2c8c99", 2);
  line.style.opacity = 0;
  line.style.animation = "rise 0.7s ease forwards";
  line.style.animationDelay = "0.1s";
});

// Draw nodes.
measurements.forEach((node, index) => {
  const cx = scaleX(new Date(node.ts).getTime());
  const cy = scaleY(node.value);
  const circle = document.createElementNS("http://www.w3.org/2000/svg", "circle");
  circle.setAttribute("cx", cx);
  circle.setAttribute("cy", cy);
  circle.setAttribute("r", 6);
  circle.setAttribute("fill", "#ff7a59");
  circle.setAttribute("stroke", "#1d1a16");
  circle.setAttribute("stroke-width", "1.5");
  circle.style.opacity = 0;
  circle.style.animation = "rise 0.5s ease forwards";
  circle.style.animationDelay = `${0.05 * index}s`;
  const title = document.createElementNS("http://www.w3.org/2000/svg", "title");
  title.textContent = `${node.label} (${node.id}) | ${node.ts} | value ${node.value}`;
  circle.appendChild(title);
  svg.appendChild(circle);

  const tag = document.createElementNS("http://www.w3.org/2000/svg", "text");
  tag.setAttribute("x", cx + 8);
  tag.setAttribute("y", cy - 10);
  tag.setAttribute("font-size", "11");
  tag.setAttribute("fill", "#1d1a16");
  tag.setAttribute("font-family", "IBM Plex Mono, monospace");
  tag.textContent = node.label;
  tag.style.opacity = 0;
  tag.style.animation = "rise 0.5s ease forwards";
  tag.style.animationDelay = `${0.05 * index + 0.1}s`;
  svg.appendChild(tag);
});
