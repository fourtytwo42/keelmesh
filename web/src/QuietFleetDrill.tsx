import { useState } from "react";
import { api, requestID } from "./api";
import type { QuietFleetSnapshot } from "./types";

const labels: Record<string, [string,string]> = {
  enter_mode: ["Enter Quiet Fleet", "Open the signed low-duty cell"],
  inject_slowdown: ["Slow Vessel 4", "Evaluate the first redistribution"],
  submit_revision: ["Revise proposal", "Respect every local speed envelope"],
  commit_proposal: ["Commit exact hash", "Arm a future tape boundary"],
  advance_to_activation: ["Advance to activation", "Switch all assignments atomically"],
};

type Props = { value:QuietFleetSnapshot; onChange:(value:QuietFleetSnapshot)=>void; onError:(message:string)=>void };

export function QuietFleetDrill({value,onChange,onError}:Props) {
  const [busy,setBusy] = useState(false);
  async function command(kind:string) {
    setBusy(true); onError("");
    try {
      const id=requestID(`quiet-${kind}`);
      const next=await api<QuietFleetSnapshot>("/api/v1/quiet-fleet/commands",{method:"POST",body:JSON.stringify({schema_version:1,kind,request_id:id,idempotency_key:id,expected_state_version:value.state_version,proposal_hash:kind==="commit_proposal"?value.proposal?.content_hash:undefined})});
      onChange(next);
    } catch(error) { onError(error instanceof Error?error.message:String(error)); }
    finally { setBusy(false); }
  }
  const [label,detail]=labels[value.next_action??""]??["Coordination complete","Revision is active"];
  return <aside className="quiet-fleet" aria-label="Quiet Fleet adaptation">
    <header><div><small>QUIET FLEET · CELL 2–5</small><h2>{value.phase.replaceAll("_"," ")}</h2></div><span>T+{value.mission_tick}s</span></header>
    <p>{value.summary}</p>
    <div className="quiet-metrics"><span><small>QUORUM</small><b>{value.metrics.quorum_count}/{value.metrics.quorum_required}</b></span><span><small>AFFECTED ARMED</small><b>{value.metrics.affected_armed}/{value.metrics.affected_required}</b></span><span><small>RADIO</small><b>{value.metrics.bytes_sent}/{value.metrics.byte_budget} B</b></span><span><small>ROUNDS</small><b>{value.metrics.rounds}/{value.contract.maximum_rounds}</b></span></div>
    <div className="node-votes">{value.contract.members.map((id)=>{const decision=value.decisions.find((d)=>d.node_id===id);return <span key={id} className={decision?.decision??"waiting"}><b>{id.replace("vessel-0","V")}</b><i>{decision?.decision??"waiting"}</i>{decision?.reason_code&&<small>{decision.reason_code}</small>}</span>})}</div>
    {value.proposal&&<div className="hash-row"><span>PROPOSAL {value.proposal.revision}</span><code>{value.proposal.content_hash.slice(7,19)}</code></div>}
    {value.commit&&<div className="activation"><span>FUTURE COMMIT</span><b>T+{value.commit.activation_tick}s</b><code>{value.commit.content_hash.slice(7,19)}</code></div>}
    <div className="quiet-action"><button disabled={busy||!value.next_action} onClick={()=>value.next_action&&command(value.next_action)}><strong>{busy?"Coordinating…":label}</strong><small>{detail}</small></button><button className="quiet-auto" disabled={busy||value.phase!=="ready"} onClick={()=>command("auto_run")}>Auto-run</button></div>
    <footer><span>{value.inference_label}</span><span>{value.contract.bulk_traffic_suppressed?`${value.metrics.bulk_messages_suppressed} bulk messages suppressed`:"normal traffic"}</span></footer>
  </aside>;
}
