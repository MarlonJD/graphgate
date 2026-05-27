import React, { useEffect, useState } from "react";
import { createRoot } from "react-dom/client";
import "./style.css";

const tabs = ["Dashboard", "Operations", "Schema", "Manifest", "Diff", "Test Runs", "Reports"];

function App() {
  const [state, setState] = useState(null);
  const [tab, setTab] = useState("Dashboard");

  useEffect(() => {
    fetch("/api/state")
      .then((response) => response.json())
      .then(setState)
      .catch((error) => setState({ error: error.message }));
  }, []);

  if (!state) return <main className="shell">Loading GraphGate...</main>;
  if (state.error) return <main className="shell">{state.error}</main>;

  const ok = state.validation.issues.length === 0;
  return (
    <main className="shell">
      <header className="topbar">
        <div>
          <h1>GraphGate</h1>
          <p>Local GraphQL contract gate</p>
        </div>
        <div className={`status ${ok ? "pass" : "fail"}`}>{ok ? "PASS" : "FAIL"}</div>
      </header>
      <nav className="tabs">
        {tabs.map((item) => (
          <button key={item} className={tab === item ? "active" : ""} onClick={() => setTab(item)}>
            {item}
          </button>
        ))}
      </nav>
      {tab === "Dashboard" && <Dashboard data={state} />}
      {tab === "Operations" && <Operations data={state} />}
      {tab === "Schema" && <Schema data={state} />}
      {tab === "Manifest" && <pre>{JSON.stringify(state.manifest, null, 2)}</pre>}
      {tab === "Diff" && <Placeholder text="Run graphgate diff --base <schema> to generate impacted operation reports." />}
      {tab === "Test Runs" && <Placeholder text="Run graphgate test --env local to execute fixture-based contract tests." />}
      {tab === "Reports" && <Reports data={state} />}
    </main>
  );
}

function Dashboard({ data }) {
  return (
    <>
      <div className="grid">
        <Metric label="Operations" value={data.validation.operations.length} />
        <Metric label="Operation files" value={data.validation.operationFiles.length} />
        <Metric label="Issues" value={data.validation.issues.length} />
        <Metric label="Schema types" value={data.schema.types?.length || 0} />
      </div>
      <section className="panel">
        <h2>Latest Validation</h2>
        {data.validation.issues.length === 0 ? <p>No issues found.</p> : data.validation.issues.map((issue) => <p key={`${issue.code}-${issue.message}`}>{issue.message}</p>)}
      </section>
    </>
  );
}

function Operations({ data }) {
  return data.validation.operations.map((op) => (
    <div className="row" key={op.id}>
      <strong>{op.name}</strong>
      <code>{op.id}</code>
      <span>{op.file}</span>
    </div>
  ));
}

function Schema({ data }) {
  return (data.schema.types || []).map((type) => (
    <section className="panel" key={type.name}>
      <h2>{type.name}</h2>
      {(type.fields || []).map((field) => (
        <div key={field.name}>
          <code>{field.name}</code>: {field.type}
        </div>
      ))}
    </section>
  ));
}

function Reports({ data }) {
  return (
    <section className="panel">
      {(data.reports || []).map((report) => (
        <p key={report.path}>
          <a href={report.path}>{report.name}</a>
        </p>
      ))}
    </section>
  );
}

function Metric({ label, value }) {
  return (
    <div className="metric">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function Placeholder({ text }) {
  return (
    <section className="panel">
      <p>{text}</p>
    </section>
  );
}

createRoot(document.getElementById("root")).render(<App />);
