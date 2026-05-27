const state = {
  tab: "dashboard",
  data: null,
};

const view = document.querySelector("#view");
const status = document.querySelector("#status");

document.querySelectorAll("[data-tab]").forEach((button) => {
  button.addEventListener("click", () => {
    state.tab = button.dataset.tab;
    document.querySelectorAll("[data-tab]").forEach((item) => item.classList.toggle("active", item === button));
    render();
  });
});

async function load() {
  const response = await fetch("/api/state");
  state.data = await response.json();
  render();
}

function render() {
  const data = state.data;
  if (!data) {
    view.innerHTML = "<div class='panel'>Loading project state...</div>";
    return;
  }
  const ok = data.validation?.issues?.length === 0;
  status.textContent = ok ? "PASS" : "FAIL";
  status.className = `status ${ok ? "pass" : "fail"}`;

  if (state.tab === "dashboard") renderDashboard(data);
  if (state.tab === "operations") renderOperations(data);
  if (state.tab === "schema") renderSchema(data);
  if (state.tab === "manifest") renderManifest(data);
  if (state.tab === "diff") renderPlaceholder("Diff", "Run graphgate diff --base <schema> to generate impacted operation reports.");
  if (state.tab === "tests") renderPlaceholder("Test Runs", "Run graphgate test --env local to execute fixture-based contract tests.");
  if (state.tab === "reports") renderReports(data);
}

function renderDashboard(data) {
  view.innerHTML = `
    <div class="grid">
      ${metric("Operations", data.validation.operations.length)}
      ${metric("Operation files", data.validation.operationFiles.length)}
      ${metric("Issues", data.validation.issues.length)}
      ${metric("Schema types", data.schema.types?.length || 0)}
    </div>
    <div class="panel">
      <h2>Latest Validation</h2>
      ${data.validation.issues.length === 0 ? "<p>No issues found.</p>" : issueList(data.validation.issues)}
    </div>
  `;
}

function renderOperations(data) {
  view.innerHTML = data.validation.operations.map((op) => `
    <div class="row">
      <strong>${escapeHTML(op.name)}</strong>
      <code>${escapeHTML(op.id)}</code>
      <span class="muted">${escapeHTML(op.file)}</span>
    </div>
  `).join("") || "<div class='panel'>No operations found.</div>";
}

function renderSchema(data) {
  view.innerHTML = (data.schema.types || []).map((type) => `
    <div class="panel">
      <h2>${escapeHTML(type.name)} <span class="muted">${escapeHTML(type.kind)}</span></h2>
      ${(type.fields || []).map((field) => `<div><code>${escapeHTML(field.name)}</code>: ${escapeHTML(field.type)}</div>`).join("") || "<p>No fields.</p>"}
    </div>
  `).join("") || "<div class='panel'>No schema summary available.</div>";
}

function renderManifest(data) {
  view.innerHTML = `<pre>${escapeHTML(JSON.stringify(data.manifest, null, 2))}</pre>`;
}

function renderReports(data) {
  view.innerHTML = `
    <div class="panel">
      <h2>Reports</h2>
      ${(data.reports || []).map((report) => `<p><a href="${report.path}">${escapeHTML(report.name)}</a></p>`).join("")}
    </div>
  `;
}

function renderPlaceholder(title, text) {
  view.innerHTML = `<div class="panel"><h2>${title}</h2><p>${text}</p></div>`;
}

function metric(label, value) {
  return `<div class="metric"><span class="muted">${label}</span><strong>${value}</strong></div>`;
}

function issueList(issues) {
  return issues.map((issue) => `<p><code>${escapeHTML(issue.code)}</code> ${escapeHTML(issue.file || "")}: ${escapeHTML(issue.message)}</p>`).join("");
}

function escapeHTML(value) {
  return String(value ?? "").replace(/[&<>"']/g, (char) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    "\"": "&quot;",
    "'": "&#039;",
  })[char]);
}

load().catch((error) => {
  status.textContent = "ERROR";
  status.className = "status fail";
  view.innerHTML = `<div class="panel">${escapeHTML(error.message)}</div>`;
});
