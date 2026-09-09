import { useEffect, useState } from "react";
import { api, KeelMeshError, requestID } from "./api";
import type { AgentSnapshot, CapacityCostEvidenceV1, CoordinationOverviewV1, EvalCandidate, EvalRun, FleetSnapshotV2, InvestigationRun, MemorySnapshotV1, PlatformDrillReceiptV1, PlatformProofSummaryV1, PlatformSnapshot, ReplayResult } from "./types";

type Props = { value: AgentSnapshot; platform:PlatformSnapshot|null; memory:MemorySnapshotV1|null; coordination:CoordinationOverviewV1|null; onChange:(next:AgentSnapshot)=>void; onOpenSystem:()=>void; onError:(message:string)=>void };

export function EngineerView({ value, platform, memory, coordination, onChange, onOpenSystem, onError }: Props) {
  const [busy, setBusy] = useState(false);
	const [proof, setProof] = useState<PlatformProofSummaryV1|null>(null);
	const [capacity, setCapacity] = useState<CapacityCostEvidenceV1|null>(null);
	const [cost, setCost] = useState<CapacityCostEvidenceV1|null>(null);
	const [drills, setDrills] = useState<PlatformDrillReceiptV1[]>([]);
	const [fleet, setFleet] = useState<FleetSnapshotV2|null>(null);
	const [drillBusy, setDrillBusy] = useState(false);
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
	const trace = proof?.latest_trace ?? platform?.real_trace;
	useEffect(()=>{
		let active=true;
		const load=()=>Promise.all([api<PlatformProofSummaryV1>("/api/v6/platform/summary"),api<CapacityCostEvidenceV1>("/api/v6/platform/capacity"),api<CapacityCostEvidenceV1>("/api/v6/platform/cost-model"),api<{drills:PlatformDrillReceiptV1[]}>("/api/v6/platform/drills"),api<FleetSnapshotV2>("/api/v2/fleet")]).then(([nextProof,nextCapacity,nextCost,nextDrills,nextFleet])=>{if(active){setProof(nextProof);setCapacity(nextCapacity);setCost(nextCost);setDrills(nextDrills.drills);setFleet(nextFleet)}}).catch(()=>undefined);
		void load();
		const timer=window.setInterval(load,10000);
		return()=>{active=false;window.clearInterval(timer)};
	},[]);
	async function runWorkerRecoveryDrill(){
		const target=platform?.workers.find(worker=>worker.id==="worker-2"&&worker.state==="running")??platform?.workers.find(worker=>worker.state==="running");
		if(!platform||!target){onError("No healthy ingestion worker is available for the bounded recovery drill.");return}
		if(!window.confirm(`Run the bounded data-pipeline recovery drill against ${target.id}? The worker child is stopped, its supervisor remains active, and edge mission execution is protected.`))return;
		setDrillBusy(true);onError("");
		try{
			const receipt=await api<PlatformDrillReceiptV1>("/api/v6/platform/drills",{method:"POST",body:JSON.stringify({request_id:requestID("platform-drill"),idempotency_key:requestID("platform-drill-idem"),expected_platform_state_version:platform.state_version,type:"data_pipeline_worker_recovery",target_id:target.id,actor_identity:"demo-engineer",confirmed:true})});
			setDrills(current=>[receipt,...current.filter(item=>item.id!==receipt.id)]);
		}catch(error){onError(error instanceof Error?error.message:String(error))}finally{setDrillBusy(false)}
	}

  return <section className="engineer-view" aria-label="AI Lab workspace">
    <header className="engineer-hero">
      <div><small>AI LAB · INVESTIGATE → REPLAY → EVALUATE</small><h1>{incident.title}</h1><p>{incident.summary}</p></div>
      <button className="engineer-system-link" onClick={onOpenSystem}>View live system →</button>
      <div className={`ai-health ${value.available ? "ready" : "degraded"}`}><span />{value.available ? "AI Lab ready" : "AI degraded"}<small>Mission authority independent</small></div>
    </header>
	<div className="platform-proof-ribbon" aria-label="Platform plane health">{(proof?.planes??[]).map(plane=><div key={plane.id} className={plane.state}><i/><span><small>{plane.label}</small><b>{plane.state.replaceAll("_"," ")}</b><em>{plane.detail}</em></span></div>)}{!proof&&<p>Loading live platform evidence…</p>}</div>
	<div className="platform-slo-strip" aria-label="Platform service objectives">{(proof?.slos??[]).map(item=><div key={item.id} className={item.state} title={`${item.source} · ${item.workload} · ${item.window}`}><small>{item.label}</small><b>{item.measured?`${Math.round(item.value)} ${item.unit}`:"NOT MEASURED"}</b><em>{item.comparison==="eq"?"=":"≤"} {item.objective} {item.unit}</em></div>)}</div>
	<section className="platform-drill-runner" aria-label="Bounded platform drills"><header><div><small>CONTROLLED FAILURE</small><h2>SLO drill runner</h2></div><span>Exact target · explicit confirmation · durable receipt</span></header><div><button disabled={drillBusy||!platform?.workers.some(worker=>worker.state==="running")} onClick={runWorkerRecoveryDrill}><b>Data pipeline recovery</b><span>Stops one worker child; supervisor rollback remains armed.</span></button><article><b>Coordinator failure</b><span>Run only after signed leader preflight and node-service rollback are available.</span><em>PROTECTED</em></article><article><b>Radio quorum partition</b><span>Radio interface only; management and provider paths cannot be targets.</span><em>PROTECTED</em></article></div>{drills[0]&&<footer><b>{drills[0].type.replaceAll("_"," ")}</b><span>{drills[0].state} · {drills[0].outcome}</span><code>{drills[0].evidence_hash?.slice(0,16)??"collecting evidence"}</code></footer>}</section>

    <div className="engineer-grid">
      <article className="engineer-card coordination-card"><header><span>M12</span><div><small>REAL CONSENSUS</small><h2>Quorum-backed node authority</h2></div><strong>{cells.length ? `${cells.length} cells` : "offline"}</strong></header>
        <div className="memory-evidence-strip">{cells.map(cell=><div key={cell.id}><small>CELL {cell.id} · TERM {cell.leader?.term??"—"}</small><b>{cell.leader?.leader_node_id??"electing"}</b><em>{cell.leader?.reachable_voters??0}/{cell.leader?.quorum_required??4} proof quorum · index {cell.leader?.commit_index??0}</em></div>)}</div>
        <div className="memory-hit-list">{cells.map(cell=><div key={cell.id}><span className={cell.leader?.state==="leader"?"verified":"inferred"}>{cell.leader?.state??"unavailable"}</span><b>epoch {cell.leader?.authority_epoch??0}</b><em>{cell.leader?.last_election_ms??0} ms election · {cell.leader?.state_hash?.slice(0,12)??"no checksum"}</em></div>)}{!cells.length&&<p>Raft telemetry is unavailable; simulated authority remains the rollback path.</p>}</div>
        <p>Radio-plane Raft and application signatures are independent of management, AI, memory, and telemetry.</p>
      </article>
      <article className="engineer-card execution-card"><header><span>M14</span><div><small>EDGE EXECUTION</small><h2>Full-program autonomy</h2></div><strong>{fleet?.missions.filter(mission=>mission.execution?.complete_program_onboard).length??0} active</strong></header>
        <div className="memory-evidence-strip">{(fleet?.missions??[]).filter(mission=>mission.execution).slice(0,3).map(mission=><div key={mission.id}><small>{mission.name}</small><b>R{mission.execution!.active_revision} · {mission.execution!.total_segments} segments</b><em>{Math.ceil(mission.execution!.authorized_time_remaining_seconds/60)} min authority · {mission.execution!.terminal_contingency.replaceAll("_"," ")}</em></div>)}{!fleet?.missions.some(mission=>mission.execution)&&<div><small>PROGRAM STORE</small><b>NO ACTIVE PROGRAM</b><em>Idle vessels hold without execution authority.</em></div>}</div>
        <p>Each assigned node stores the complete finite program before readiness. Group or isolated-node adaptations remain inside the same signed geography, reserve, PNT, separation, and expiry envelope.</p>
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

      <article className="engineer-card trace-card"><header><span>06</span><div><small>REAL CROSS-PROCESS TRACE</small><h2>OTLP telemetry waterfall</h2></div><code>{trace?.trace_id.slice(0,12) ?? "waiting"}</code></header>
        <div className="waterfall">{(trace?.spans ?? []).map((span,index)=><div key={span.span_id}><span>{span.service}</span><b>{span.name}</b><i style={{marginLeft:`${Math.min(index*7,28)}%`,width:`${Math.max(12,Math.min(72,span.duration_ms/20))}%`}}/><em>{Math.round(span.duration_ms)} ms</em></div>)}{!trace && <p>Waiting for backend OTLP spans. Collection is private and non-blocking.</p>}</div>
      </article>
    </div>
	<details className="platform-capacity-drawer"><summary>Capacity and cost evidence <span>12 measured · 100 / 1,000 projected</span></summary><div><section><h3>Capacity</h3>{(capacity?.capacity??[]).map(item=><p key={item.asset_count}><b>{item.asset_count} assets</b><span className={item.evidence_class}>{item.evidence_class}</span><em>{Math.round(item.events_per_second)} events/s · {item.worker_count} workers · lag {item.consumer_lag}</em></p>)}</section><section><h3>Monthly planning model</h3>{(cost?.cost??[]).map(item=><p key={item.asset_count}><b>{item.asset_count} assets</b><span className="projected">projected</span><em>${item.estimated_monthly_usd.toFixed(0)} · assumptions visible in API receipt</em></p>)}</section></div></details>
    <div className="engineer-action"><div><small>CURRENT PHASE</small><strong>{value.phase.replaceAll("_"," ")}</strong><span>{value.summary}</span></div><button disabled={busy || (!value.available && !value.investigation)} onClick={primary.run}>{busy ? "Working…" : primary.label}</button></div>
  </section>;
}
