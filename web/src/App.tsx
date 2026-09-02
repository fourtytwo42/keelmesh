import { useCallback, useEffect, useMemo, useState } from "react";
import { api, KeelMeshError, requestID, streamURL } from "./api";
import { MissionMap } from "./MissionMap";
import { PlatformCutaway } from "./PlatformCutaway";
import { ResilienceDrill } from "./ResilienceDrill";
import type { AuditEvent, Bootstrap, FleetSnapshot, Lease, MissionIntent, PlanCandidate, PlatformSnapshot, Point, Polygon, Preview, StreamMessage } from "./types";
import "./app.css";

const defaultCommand = "Search this area with six vessels. Maintain 30% reserve and avoid the exclusion zone.";

export default function App() {
  const [bootstrap, setBootstrap] = useState<Bootstrap | null>(null);
  const [snapshot, setSnapshot] = useState<FleetSnapshot | null>(null);
  const [area, setArea] = useState<Polygon | null>(null);
  const [command, setCommand] = useState(defaultCommand);
  const [intent, setIntent] = useState<MissionIntent | null>(null);
  const [plans, setPlans] = useState<PlanCandidate[]>([]);
  const [selectedID, setSelectedID] = useState("");
  const [preview, setPreview] = useState<Preview | null>(null);
  const [previewSecond, setPreviewSecond] = useState(0);
  const [lease, setLease] = useState<Lease | null>(null);
  const [audit, setAudit] = useState<AuditEvent[]>([]);
  const [drawNonce, setDrawNonce] = useState(0);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [connected, setConnected] = useState(false);
  const [view, setView] = useState<"operator" | "cutaway">("operator");
  const [platform, setPlatform] = useState<PlatformSnapshot | null>(null);

  const refresh = useCallback(async () => {
    const value = await api<Bootstrap>("/api/v1/bootstrap");
    setBootstrap(value); setSnapshot(value.snapshot); setAudit(value.audit);
  }, []);

  useEffect(() => { refresh().catch((e) => setError(e.message)); }, [refresh]);
  useEffect(() => {
    let cancelled = false;
    const refresh = () => api<PlatformSnapshot>("/api/v1/platform").then((next) => {
      if (!cancelled) setPlatform(next);
    }).catch(() => undefined);
    void refresh();
    const timer = window.setInterval(refresh, 1_000);
    return () => { cancelled = true; window.clearInterval(timer); };
  }, []);
  useEffect(() => {
    let socket: WebSocket | null = null; let stopped = false; let retry = 500;
    const connect = () => {
      if (stopped) return; socket = new WebSocket(streamURL());
      socket.onopen = () => { setConnected(true); retry = 500; refresh().catch(() => undefined); };
      socket.onmessage = (event) => { const message = JSON.parse(event.data) as StreamMessage; if (message.snapshot) setSnapshot(message.snapshot); if (message.resilience) setSnapshot((current) => current ? { ...current, state_version: message.resilience!.state_version, resilience: message.resilience } : current); if (message.platform) setPlatform(message.platform); if (message.audit) setAudit((current) => [...current.slice(-19), message.audit!]); };
      socket.onclose = () => { setConnected(false); if (!stopped) window.setTimeout(connect, retry); retry = Math.min(5000, retry * 2); };
    };
    connect(); return () => { stopped = true; socket?.close(); };
  }, [refresh]);

  const selectedPlan = useMemo(() => plans.find((p) => p.id === selectedID) ?? null, [plans, selectedID]);
  const previewPositions = preview ? preview.samples[Math.min(previewSecond, preview.samples.length - 1)]?.positions ?? null : null;
  const stateVersion = snapshot?.state_version ?? 0;

  async function createPlans() {
    if (!area) { setError("Select the suggested area or draw a polygon first."); return; }
    setBusy(true); setError(""); setPreview(null); setLease(null);
    try {
      const compiled = await api<MissionIntent>("/api/v1/intents:compile", { method: "POST", body: JSON.stringify({ request_id: requestID("intent"), expected_state_version: stateVersion, text: command, area }) });
      setIntent(compiled);
      const result = await api<{ plans: PlanCandidate[] }>("/api/v1/plans", { method: "POST", body: JSON.stringify({ request_id: requestID("plans"), expected_state_version: compiled.source_state_version, intent_id: compiled.id }) });
      setPlans(result.plans); setSelectedID((result.plans.find((p) => p.recommended) ?? result.plans[0]).id);
    } catch (e) { setError(e instanceof KeelMeshError ? `${e.code}: ${e.message}` : String(e)); await refresh(); }
    finally { setBusy(false); }
  }

  async function previewPlan() {
    if (!selectedPlan) return; setBusy(true); setError("");
    try { const value = await api<Preview>(`/api/v1/plans/${selectedPlan.id}:preview`, { method: "POST", body: JSON.stringify({ request_id: requestID("preview"), expected_state_version: selectedPlan.source_state_version }) }); setPreview(value); setPreviewSecond(0); }
    catch (e) { setError(e instanceof Error ? e.message : String(e)); } finally { setBusy(false); }
  }

  async function authorize() {
    if (!selectedPlan || !intent) return; setBusy(true); setError("");
    try { setLease(await api<Lease>(`/api/v1/plans/${selectedPlan.id}:authorize`, { method: "POST", body: JSON.stringify({ request_id: requestID("authorize"), expected_state_version: intent.source_state_version, plan_hash: selectedPlan.content_hash, operator_id: "demo-operator" }) })); }
    catch (e) { setError(e instanceof KeelMeshError ? `${e.code}: ${e.message}` : String(e)); } finally { setBusy(false); }
  }

  async function startMission() {
    if (!lease || !intent) return; setBusy(true); setError(""); setPreviewSecond(0);
    try { const state = await api<FleetSnapshot["mission"]>(`/api/v1/missions/${lease.mission_id}:start`, { method: "POST", body: JSON.stringify({ request_id: requestID("start"), expected_state_version: intent.source_state_version, lease_id: lease.id, plan_hash: lease.plan_hash, idempotency_key: `start-${lease.id}` }) }); setSnapshot((current) => current ? { ...current, mission: state, state_version: current.state_version + 1 } : current); }
    catch (e) { setError(e instanceof KeelMeshError ? `${e.code}: ${e.message}` : String(e)); await refresh(); } finally { setBusy(false); }
  }

  if (!bootstrap || !snapshot) return <main className="loading"><span className="brand-mark">KM</span><p>Loading KeelMesh mission appliance…</p></main>;
  const phase = snapshot.mission.phase;
  const canCreate = !busy && !!area && !["executing"].includes(phase);

  return <main className="app-shell">
    <header className="topbar">
      <div className="brand"><span className="brand-mark">KM</span><span><strong>KeelMesh</strong><small>Fleet Intent Control</small></span></div>
      <div className="scenario"><span className="sim-label">SIMULATION</span><span>{snapshot.scenario_name}</span><span>×{snapshot.simulation_rate}</span></div>
      <div className="view-switch" role="group" aria-label="Application view"><button className={view === "operator" ? "active" : ""} onClick={() => setView("operator")}>Operator</button><button className={view === "cutaway" ? "active" : ""} onClick={() => setView("cutaway")}>Cutaway</button></div>
      <div className={`connection ${connected ? "online" : "offline"}`}><span />{connected ? "Live" : "Reconnecting"}</div>
    </header>

    <section className="workspace">
      {view === "cutaway" && platform && <PlatformCutaway value={platform} fleet={snapshot} onError={setError} />}
      <div className={view === "cutaway" ? "operator-hidden" : "operator-layer"}>
      <MissionMap snapshot={snapshot} boundary={bootstrap.boundary} exclusion={bootstrap.exclusion_zone} holding={bootstrap.holding_area} area={area} plan={selectedPlan} previewPositions={phase === "previewing" ? previewPositions : null} drawNonce={drawNonce} onAreaDrawn={(polygon) => { setArea(polygon); setError(""); setPlans([]); setPreview(null); }} />

      {snapshot.resilience && <ResilienceDrill value={snapshot.resilience} onChange={(resilience) => setSnapshot((current) => current ? { ...current, state_version: resilience.state_version, resilience } : current)} onError={setError} />}

      <aside className={`plan-panel ${plans.length ? "open" : ""}`} aria-label="Plan comparison">
        <div className="panel-heading"><div><small>PLAN COMPARISON</small><h1>{plans.length ? "Choose the approach" : "Mission workspace"}</h1></div>{plans.length > 0 && <span>{plans.length} OPTIONS</span>}</div>
        {plans.length === 0 ? <div className="empty-state"><div className="empty-orbit">◎</div><h2>Define the objective</h2><p>Select an area, describe the mission, and KeelMesh will compute two constrained plans.</p><ol><li>Choose the search area</li><li>Review the typed intent</li><li>Compare before anything moves</li></ol></div> : <>
          <div className="plan-list">{plans.map((plan) => <button key={plan.id} className={`plan-card ${selectedID === plan.id ? "selected" : ""}`} onClick={() => { setSelectedID(plan.id); setPreview(null); setLease(null); }} aria-pressed={selectedID === plan.id}>
            <div className="plan-title"><span>{plan.name}</span>{plan.recommended && <em>RECOMMENDED</em>}</div><p>{plan.summary}</p>
            <dl><div><dt>Coverage</dt><dd>{plan.metrics.coverage_percent.toFixed(1)}%</dd></div><div><dt>Duration</dt><dd>{plan.metrics.duration_minutes.toFixed(1)}m</dd></div><div><dt>Min reserve</dt><dd>{Math.round(plan.metrics.minimum_reserve * 100)}%</dd></div><div><dt>Score</dt><dd>{plan.score.total.toFixed(1)}</dd></div></dl>
            <span className={`policy ${plan.policy.status}`}>{plan.policy.status === "approval_required" ? "Ready for approval" : "Policy blocked"}</span>
          </button>)}</div>
          {selectedPlan && <div className="primary-actions">
            {!preview && <button className="primary" disabled={busy || selectedPlan.policy.status === "prohibited"} onClick={previewPlan}>Preview {selectedPlan.name}</button>}
            {preview && !lease && <><div className="safety-note"><strong>Nothing has been sent yet.</strong><span>Previewing a deterministic simulation of plan <code>{selectedPlan.content_hash.slice(7, 19)}</code>.</span></div><button className="primary authorize" disabled={busy} onClick={authorize}>Authorize exact plan</button></>}
            {lease && phase !== "executing" && phase !== "completed" && <><div className="lease-summary"><span>LEASE READY</span><strong>{lease.asset_ids.length} vessels · expires {new Date(lease.expires_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}</strong><code>{lease.plan_hash.slice(0, 26)}…</code></div><button className="primary start" disabled={busy} onClick={startMission}>Start authorized mission</button></>}
            {phase === "executing" && <div className="execution"><span>MISSION EXECUTING</span><strong>{Math.round(snapshot.mission.progress * 100)}%</strong><div><i style={{ width: `${snapshot.mission.progress * 100}%` }} /></div></div>}
            {phase === "completed" && <div className="execution complete"><span>MISSION COMPLETE</span><strong>100%</strong><div><i style={{ width: "100%" }} /></div></div>}
          </div>}
        </>}
      </aside>

      {preview && phase === "previewing" && <div className="preview-timeline"><button onClick={() => setPreviewSecond((s) => s >= preview.duration_seconds ? 0 : s + 1)} aria-label="Play preview">▶</button><input aria-label="Preview time" type="range" min="0" max={preview.duration_seconds} value={previewSecond} onChange={(e) => setPreviewSecond(Number(e.target.value))} /><span>{Math.floor(previewSecond / 60)}:{String(previewSecond % 60).padStart(2, "0")} / {Math.floor(preview.duration_seconds / 60)}:{String(preview.duration_seconds % 60).padStart(2, "0")}</span></div>}

      <section className="command-dock" aria-label="Mission command">
        <div className="dock-top"><span>WHAT SHOULD THE FLEET DO?</span><div className="area-actions"><button onClick={() => { setArea(bootstrap.suggested_area.geometry); setPlans([]); setPreview(null); setError(""); }}>Use suggested area</button><button onClick={() => setDrawNonce((n) => n + 1)}>Draw area</button></div></div>
        <div className="command-row"><span className="prompt">›</span><input value={command} onChange={(e) => setCommand(e.target.value)} onKeyDown={(e) => { if (e.key === "Enter" && canCreate) createPlans(); }} aria-label="Mission intent" /><button className="create" disabled={!canCreate} onClick={createPlans}>{busy ? "Working…" : "Create plans"}</button></div>
        <div className="dock-status"><span className={area ? "ready" : "waiting"}>{area ? "Area ready" : "Choose an area"}</span><span>6 vessels available</span><span>30% reserve floor</span><span>Offline deterministic mode</span></div>
        {error && <div className="error" role="alert">{error}</div>}
      </section>

      <aside className="audit-strip" aria-label="Audit timeline"><header><span>LIVE AUDIT</span><strong>{audit.length}</strong></header>{audit.slice(-5).reverse().map((event) => <div key={event.id}><time>{new Date(event.at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" })}</time><span>{event.summary}</span></div>)}</aside>
      </div>
    </section>
    <div className="sr-live" aria-live="polite">Mission phase: {phase}. {error}</div>
  </main>;
}
