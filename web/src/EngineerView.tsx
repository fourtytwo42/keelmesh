import { useState } from "react";
import { api, KeelMeshError, requestID } from "./api";
import type { AgentSnapshot, CoordinationOverviewV1, EvalCandidate, EvalRun, InvestigationRun, MemorySnapshotV1, ReplayResult } from "./types";

type Props = { value: AgentSnapshot; memory:MemorySnapshotV1|null; coordination:CoordinationOverviewV1|null; onChange:(next:AgentSnapshot)=>void; onOpenSystem:()=>void; onError:(message:string)=>void };

export function EngineerView({ value, memory, coordination, onChange, onOpenSystem, onError }: Props) {
  const [busy, setBusy] = useState(false);
  const incident = value.incidents[0];
  const receipts = value.investigation?.tool_receipts ?? [];
  const citations = value.investigation?.citations ?? [];
  const attempts = value.investigation?.provider_attempts ?? [];
  const refresh = async () => onChange(await api<AgentSnapshot>("/api/v1/ai"));
  const mutation = (prefix:string) => ({ request_id:requestID(prefix), idempotency_key:requestID(`${prefix}-idem`), expected_ai_state_version:value.state_version });
  const act = async (work:()=>Promise<unknown>) => { setBusy(true); onError(""); try { await work(); await refresh(); } catch (error) { onError(error instanceof KeelMeshError ? `${error.code}: ${error.message}` : String(error)); await refresh().catch(()=>undefined); } finally { setBusy(false); } };
  const investigate = () => act(()=>api<InvestigationRun>(`/api/v1/incidents/${incident.id}:investigate`,{method:"POST",body:JSON.stringify(mutation("investigate"))}));
  const replay = () => act(()=>api<ReplayResult>(`/api/v1/investigations/${value.investigation!.id}:replay`,{method:"POST",body:JSON.stringify(mutation("replay"))}));
  const approve = () => act(()=>api<EvalCandidate>(`/api/v1/eval-candidates/${value.candidate!.id}:approve`,{method:"POST",body:JSON.stringify({...mutation("approve"),candidate_hash:value.candidate!.candidate_hash,operator_identity:"demo-engineer"})}));
  const evaluate = () => act(()=>api<EvalRun>("/api/v1/evaluations/runs",{method:"POST",body:JSON.stringify({...mutation("eval"),candidate_id:value.candidate!.id})}));
  const fault = (kind:string) => act(()=>api("/api/v1/ai/faults",{method:"POST",body:JSON.stringify({...mutation("fault"),kind})}));
  const reset = () => act(()=>api("/api/v1/scenarios/ai-tooling:reset",{method:"POST",body:JSON.stringify(mutation("reset"))}));
  const primary = !value.investigation ? {label:"Investigate incident",run:investigate} : !value.investigation.replay ? {label:"Run isolated replay",run:replay} : value.candidate?.state !== "approved" ? {label:"Approve exact candidate hash",run:approve} : !value.evaluation ? {label:"Run versioned regression",run:evaluate} : {label:"Reset AI workflow",run:reset};
  const cells = Object.entries(coordination?.cells ?? {}).map(([id,nodes]) => ({id,nodes,leader:nodes.find(node=>node.state==="leader")??nodes.find(node=>node.leader_node_id===node.local_node_id)}));

  return <section className="engineer-view" aria-label="AI Lab workspace">
    <header className="engineer-hero">
      <div><small>AI LAB · INVESTIGATE → REPLAY → EVALUATE</small><h1>{incident.title}</h1><p>{incident.summary}</p></div>
      <button className="engineer-system-link" onClick={onOpenSystem}>View live system →</button>
      <div className={`ai-health ${value.available ? "ready" : "degraded"}`}><span />{value.available ? "AI Lab ready" : "AI degraded"}<small>Mission authority independent</small></div>
    </header>

    <div className="engineer-grid">
      <article className="engineer-card coordination-card"><header><span>M12</span><div><small>REAL CONSENSUS</small><h2>Quorum-backed node authority</h2></div><strong>{cells.length ? `${cells.length} cells` : "offline"}</strong></header>
        <div className="memory-evidence-strip">{cells.map(cell=><div key={cell.id}><small>CELL {cell.id} · TERM {cell.leader?.term??"—"}</small><b>{cell.leader?.leader_node_id??"electing"}</b><em>{cell.leader?.reachable_voters??0}/{cell.leader?.quorum_required??4} proof quorum · index {cell.leader?.commit_index??0}</em></div>)}</div>
        <div className="memory-hit-list">{cells.map(cell=><div key={cell.id}><span className={cell.leader?.state==="leader"?"verified":"inferred"}>{cell.leader?.state??"unavailable"}</span><b>epoch {cell.leader?.authority_epoch??0}</b><em>{cell.leader?.last_election_ms??0} ms election · {cell.leader?.state_hash?.slice(0,12)??"no checksum"}</em></div>)}{!cells.length&&<p>Raft telemetry is unavailable; simulated authority remains the rollback path.</p>}</div>
        <p>Radio-plane Raft and application signatures are independent of management, AI, memory, and telemetry.</p>
      </article>
      <article className="engineer-card memory-card"><header><span>M11</span><div><small>RETRIEVAL EVIDENCE</small><h2>Scoped context assembly</h2></div><strong>{memory?.retrieval_mode ?? "offline"}</strong></header>
        <div className="memory-evidence-strip"><div><small>EXACT TURNS</small><b>{memory?.last_context?.recent_turns.length??0}</b></div><div><small>SEMANTIC</small><b>{memory?.last_context?.semantic_memories.length??0}</b></div><div><small>RUNBOOK</small><b>{memory?.last_context?.procedural_chunks.length??0}</b></div><div><small>EPISODES</small><b>{memory?.last_context?.operational_episodes.length??0}</b></div></div>
        <div className="memory-hit-list">{(memory?.last_receipt?.hits??[]).slice(0,3).map(hit=><div key={hit.item_id}><span className={hit.trust}>{hit.trust}</span><b>{hit.kind.replaceAll("_"," ")}</b><em>{hit.scope.kind} · {Math.round(hit.combined_score*100)}%</em></div>)}{!memory?.last_receipt?.hits.length&&<p>No retrieval receipt yet. Run the investigation to assemble cited context.</p>}</div>
        <p>{memory?.last_context ? `${memory.last_context.estimated_tokens}/${memory.last_context.token_budget} estimated tokens · receipt ${memory.last_context.retrieval_receipt_id.slice(0,12)}` : "No context has been assembled for this investigation yet."}</p>
      </article>
      <article className="engineer-card evidence-card"><header><span>01</span><div><small>IMMUTABLE INCIDENT</small><h2>Bounded evidence</h2></div><code>{incident.state_checksum.slice(7,19)}</code></header>
        <div className="incident-track">{incident.evidence.map((item)=><div key={item.id}><b>{item.tick ?? "—"}</b><i/><span><strong>{item.kind}</strong>{item.summary}</span></div>)}</div>
        <footer><span>Seed {incident.scenario_seed}</span><span>{incident.classification}</span><span>fixture provenance</span></footer>
      </article>

      <article className="engineer-card provider-card"><header><span>02</span><div><small>PROVIDER EVIDENCE</small><h2>Accepted response and failover</h2></div></header>
        <div className="provider-route"><div className="route-node cloud"><b>OPENROUTER</b><span>{value.provider.models.length} ranked free models</span></div><i>→</i><div className="route-node"><b>LOCAL</b><span>{value.provider.local_enabled ? "configured" : "standby"}</span></div><i>→</i><div className="route-node mock"><b>MOCK</b><span>deterministic</span></div></div>
        <div className="fault-row"><button disabled={busy} onClick={()=>fault("fail_cloud_next")}>Fail cloud next</button><button disabled={busy} onClick={()=>fault("fail_local_next")}>Fail local next</button></div>
        <div className="attempt-list">{attempts.map((attempt,index)=><div key={`${attempt.provider}-${attempt.model}-${index}`} className={attempt.state}><b>{attempt.provider}</b><span>{attempt.model}</span><em>{attempt.state} · {attempt.latency_ms} ms</em></div>)}{attempts.length === 0 && <p>No provider request yet. Evidence collection happens first.</p>}</div>
      </article>

      <article className="engineer-card mcp-card"><header><span>03</span><div><small>PRIVATE MCP BOUNDARY</small><h2>Actual scoped tool receipts</h2></div><strong>{receipts.length}/8</strong></header>
        <div className="tool-grid">{receipts.map((receipt)=><div key={receipt.id}><span>✓</span><b>{receipt.tool}</b><small>{receipt.duration_ms} ms · {receipt.result_hash.slice(7,15)}</small></div>)}{receipts.length === 0 && <p>Read, replay, and draft capabilities only. No command or authorization tools exist.</p>}</div>
      </article>

      <article className="engineer-card diagnosis-card"><header><span>04</span><div><small>GROUNDED FINDING</small><h2>Diagnosis & citations</h2></div>{value.investigation && <strong>{Math.round(value.investigation.confidence*100)}%</strong>}</header>
        {value.investigation ? <><p className="diagnosis">{value.investigation.diagnosis || "Collecting bounded evidence through MCP…"}</p><div className="citation-list">{citations.map((citation)=><div key={citation.chunk_id}><span className={citation.trust}>{citation.trust}</span><b>{citation.title}</b><p>{citation.excerpt}</p><code>{citation.chunk_id}</code></div>)}</div></> : <p>Agent findings appear only after MCP evidence, cited retrieval, and deterministic replay are validated.</p>}
      </article>

      <article className="engineer-card eval-card"><header><span>05</span><div><small>HUMAN-GATED DATA FLYWHEEL</small><h2>Incident → regression</h2></div></header>
        <div className="flywheel"><span className={value.investigation ? "done":"active"}>Investigate</span><i>›</i><span className={value.investigation?.replay ? "done":""}>Replay</span><i>›</i><span className={value.candidate?.state === "approved" ? "done":""}>Approve hash</span><i>›</i><span className={value.evaluation ? "done":""}>Regression</span></div>
        {value.candidate && <div className="candidate"><div><small>CANDIDATE V{value.candidate.version}</small><code>{value.candidate.candidate_hash}</code></div><ul>{value.candidate.assertions.map((assertion)=><li key={assertion}>✓ {assertion.replaceAll("_"," ")}</li>)}</ul><b>{value.candidate.state}</b></div>}
        {value.evaluation && <div className="eval-results">{value.evaluation.results.map((result)=><div key={`${result.provider}-${result.model}`}><strong>{result.provider}</strong><span>{result.model || "not configured"}</span><em className={result.state}>{result.state}</em><b>{result.passed} pass · {result.skipped} skip · {result.failed} fail</b></div>)}</div>}
      </article>

      <article className="engineer-card trace-card"><header><span>06</span><div><small>TRACE CONTINUITY</small><h2>OpenTelemetry waterfall</h2></div><code>{value.trace?.trace_id.slice(0,12) ?? "waiting"}</code></header>
        <div className="waterfall">{(value.trace?.spans ?? []).map((span,index)=><div key={span.span_id}><span>{span.service}</span><b>{span.name}</b><i style={{marginLeft:`${Math.min(index*7,28)}%`,width:`${Math.max(12,Math.min(72,span.duration_ms/20))}%`}}/><em>{Math.round(span.duration_ms)} ms</em></div>)}{!value.trace && <p>The waterfall is populated from investigation and tool events, never a canned animation.</p>}</div>
      </article>
    </div>
    <div className="engineer-action"><div><small>CURRENT PHASE</small><strong>{value.phase.replaceAll("_"," ")}</strong><span>{value.summary}</span></div><button disabled={busy || (!value.available && !value.investigation)} onClick={primary.run}>{busy ? "Working…" : primary.label}</button></div>
  </section>;
}
