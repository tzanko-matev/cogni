import "./style.css";

const root = document.getElementById("app");

if (root) {
  root.innerHTML = `
    <main class="report-shell">
      <h1>Cogni Report</h1>
      <p>Report assets loaded. Interactive charts will render here.</p>
    </main>
  `;
} else {
  console.warn("Cogni report: #app container not found.");
}
