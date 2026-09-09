import { useState } from "react";
import { api, requestID } from "./api";
import type { ResilienceSnapshot } from "./types";

const actions = [
  ["fail_starlink", "Fail Starlink", "Route around the direct uplink"],
  ["partition_vessel4", "Isolate Vessel 4", "Continue past one minute from its complete onboard program"],
  ["inject_gnss_spoof", "Inject GNSS spoof", "Reject the fix and enter safe hold"],
  ["restore_contact", "Restore contact", "Reconcile position, revision, epoch, and progress"],
] as const;

type Props = { value: ResilienceSnapshot; onChange: (value: ResilienceSnapshot) => void; onError: (message: string) => void };

export function ResilienceDrill({ value, onChange, onError }: Props) {
  const [busy, setBusy] = useState(false);
  const incident = value.nodes.find((node) => node.id === value.incident_node_id);

  async function fault(kind: string, version = value.state_version) {
    const id = requestID(kind);
    return api<ResilienceSnapshot>("/api/v1/faults", { method: "POST", body: JSON.stringify({ schema_version: 1, kind, target_id: "vessel-04", scenario_tick: value.mission_tick, request_id: id, idempotency_key: id, expected_state_version: version }) });
  }
  async function run(kind: string) {
    setBusy(true); onError("");
    try { onChange(await fault(kind)); } catch (error) { onError(error instanceof Error ? error.message : String(error)); }
    finally { setBusy(false); }
  }
  async function autoRun() {
    setBusy(true); onError("");
    try {
      let current = value;
      if (current.phase !== "ready") {
        const id = requestID("reset");
        current = await api<ResilienceSnapshot>("/api/v1/scenarios/resilient-edge:reset", { method: "POST", body: JSON.stringify({ request_id: id, idempotency_key: id, expected_state_version: current.state_version }) });
        onChange(current);
      }
      for (const [kind] of actions) {
        await new Promise((resolve) => window.setTimeout(resolve, 900));
        const id = requestID(kind);
        current = await api<ResilienceSnapshot>("/api/v1/faults", { method: "POST", body: JSON.stringify({ schema_version: 1, kind, target_id: "vessel-04", scenario_tick: current.mission_tick, request_id: id, idempotency_key: id, expected_state_version: current.state_version }) });
        onChange(current);
      }
    } catch (error) { onError(error instanceof Error ? error.message : String(error)); }
    finally { setBusy(false); }
  }

  return <aside className="resilience-drill" aria-label="Resilience drill">
    <header><div><small>RESILIENCE DRILL · VESSEL 4</small><h2>{value.phase.replaceAll("_", " ")}</h2></div><span>T+{value.mission_tick}s</span></header>
    <p className="resilience-summary">{value.summary}</p>
    <div className="resilience-metrics">
      <div><small>ONBOARD PROGRAM</small><strong className={incident?.execution?.complete_program_onboard ? "trusted" : "unsafe"}>{incident?.execution?.complete_program_onboard ? "COMPLETE" : "NONE"}</strong></div>
      <div><small>PNT INTEGRITY</small><strong className={incident?.pnt.integrity}>{incident?.pnt.integrity ?? "trusted"}</strong></div>
      <div><small>UNCERTAINTY</small><strong>{incident?.pnt.uncertainty_m ?? 0}m</strong></div>
      <div><small>BUFFERED</small><strong>{incident?.buffered_events ?? 0}</strong></div>
    </div>
    <div className="resilience-evidence"><span>{incident?.execution ? `${Math.ceil(incident.execution.authorized_time_remaining_seconds / 60)} min authority remaining` : "No active authority"}</span><span>{incident?.execution ? `${incident.execution.decision_scope} decisions · epoch ${incident.execution.decision_epoch}` : "No decision scope"}</span><span>{value.hop_receipts?.length ? `${value.hop_receipts.length} signed hop receipts` : "Direct authority path"}</span><span>{incident?.pnt.excluded_sources?.includes("gnss") ? "GNSS excluded" : `${incident?.pnt.contributing_sources?.length ?? 0} PNT sources`}</span></div>
    <div className="fault-actions">{actions.map(([kind, label, detail], index) => {
      const enabled = value.next_action === kind && !busy;
      return <button key={kind} disabled={!enabled} onClick={() => run(kind)}><b>{index + 1}</b><span><strong>{label}</strong><small>{detail}</small></span></button>;
    })}</div>
    <footer><span>HWM {incident?.execution_watermark ?? -1} · {value.active_path.length ? value.active_path.join(" → ") : "NO ROUTE"}</span><button disabled={busy} onClick={autoRun}>{busy ? "Running…" : "Auto-run"}</button></footer>
  </aside>;
}
