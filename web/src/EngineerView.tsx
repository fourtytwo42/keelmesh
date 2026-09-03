import { useState } from "react";
import { api, KeelMeshError, requestID } from "./api";
import type { AgentSnapshot, EvalCandidate, EvalRun, InvestigationRun, MemorySnapshotV1, ReplayResult } from "./types";

type Props = { value: AgentSnapshot; memory:MemorySnapshotV1|null; onChange:(next:AgentSnapshot)=>void; onError:(message:string)=>void };

export function EngineerView({ value, memory, onChange, onError }: Props) {
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

  return <section className="engineer-view" aria-label="Autonomy engineer workspace">
    <header className="engineer-hero">
      <div><small>AUTONOMY ENGINEER · INCIDENT → EVALUATION</small><h1>{incident.title}</h1><p>{incident.summary}</p></div>
      <div className={`ai-health ${value.available ? "ready" : "degraded"}`}><span />{value.available ? "Agent ready" : "AI degraded"}<small>M1–M3 independent</small></div>
    </header>

    <div className="engineer-grid">
      <article className="engineer-card memory-card"><header><span>M11</span><div><small>DISTRIBUTED AGENT MEMORY</small><h2>Scoped context assembly</h2></div><strong>{memory?.retrieval_mode ?? "offline"}</strong></header>
        <div className="tool-grid"><div><span>✓</span><b>{memory?.conversation_turns ?? 0} durable turns</b><small>latest 12 exact turns per assembly</small></div><div><span>✓</span><b>{memory?.committed_items ?? 0} committed memories</b><small>{memory?.pending_candidates ?? 0} awaiting review · {memory?.tombstones ?? 0} tombstones</small></div><div><span>✓</span><b>{memory?.embedding_state ?? "unavailable"} embeddings</b><small>{memory?.embedding_version ?? "keyword fallback"}</small></div></div>
        <p>{memory?.last_context ? `${memory.last_context.estimated_tokens}/${memory.last_context.token_budget} estimated context tokens · ${memory.last_context.semantic_memories.length} semantic · ${memory.last_context.procedural_chunks.length} runbook · ${memory.last_context.operational_episodes.length} episode` : memory?.summary ?? "Memory state is unavailable; mission authority remains independent."}</p>
      </article>
      <article className="engineer-card evidence-card"><header><span>01</span><div><small>IMMUTABLE INCIDENT</small><h2>Bounded evidence</h2></div><code>{incident.state_checksum.slice(7,19)}</code></header>
        <div className="incident-track">{incident.evidence.map((item)=><div key={item.id}><b>{item.tick ?? "—"}</b><i/><span><strong>{item.kind}</strong>{item.summary}</span></div>)}</div>
        <footer><span>Seed {incident.scenario_seed}</span><span>{incident.classification}</span><span>fixture provenance</span></footer>
      </article>

      <article className="engineer-card provider-card"><header><span>02</span><div><small>ADAPTIVE PROVIDER ROUTER</small><h2>Cloud → local → mock</h2></div></header>
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
