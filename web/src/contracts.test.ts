import { describe, expect, it } from "vitest";
import fixtureJSON from "../../contracts/fixtures/mission-intent-v1.json";
import resilienceJSON from "../../contracts/fixtures/resilience-snapshot-v1.json";
import platformJSON from "../../contracts/fixtures/platform-snapshot-v1.json";
import agentJSON from "../../contracts/fixtures/agent-snapshot-v1.json";
import quietFleetJSON from "../../contracts/fixtures/quiet-fleet-snapshot-v1.json";
import memoryJSON from "../../contracts/fixtures/memory-snapshot-v1.json";
import platformProofJSON from "../../contracts/fixtures/platform-proof-summary-v1.json";
import platformDrillJSON from "../../contracts/fixtures/platform-drill-receipt-v1.json";
import executionV7JSON from "../../contracts/fixtures/execution-v7.json";
import type { AgentSnapshot, MemorySnapshotV1, MissionIntent, PlatformDrillReceiptV1, PlatformProofSummaryV1, PlatformSnapshot, QuietFleetSnapshot, ResilienceSnapshot } from "./types";

describe("shared contract fixtures", () => {
  it("reads MissionIntentV1 using the TypeScript contract", () => {
    const fixture = fixtureJSON as MissionIntent;
    expect(fixture.schema_version).toBe(1);
    expect(fixture.requested_asset_count).toBe(6);
    expect(fixture.area.type).toBe("Polygon");
    expect(fixture.constraints.avoid_zones).toEqual(["exclusion-2"]);
  });

  it("reads ResilienceSnapshotV1 using the TypeScript contract", () => {
    const fixture = resilienceJSON as ResilienceSnapshot;
    expect(fixture.scenario_id).toBe("resilient-edge-v1");
    expect(fixture.incident_node_id).toBe("vessel-04");
    expect(fixture.active_path).toEqual(["operator", "vessel-04"]);
  });

  it("reads PlatformSnapshotV1 using the TypeScript contract", () => {
    const fixture = platformJSON as PlatformSnapshot;
    expect(fixture.active_run?.vessel_count).toBe(1000);
    expect(fixture.topics[0].partitions).toBe(12);
    expect(fixture.metrics.attempted).toBe(fixture.metrics.unique_inserted + fixture.metrics.duplicates_suppressed + fixture.metrics.quarantined);
  });

  it("reads AgentSnapshotV1 using the TypeScript contract", () => {
    const fixture = agentJSON as AgentSnapshot;
    expect(fixture.incidents[0].scenario_seed).toBe(42042);
    expect(fixture.provider.models.at(-1)).toBe("openrouter/free");
    expect(fixture.security_denials).toBe(1);
  });

  it("reads QuietFleetSnapshotV1 using the TypeScript contract", () => {
    const fixture = quietFleetJSON as QuietFleetSnapshot;
    expect(fixture.contract.quorum).toBe(3);
    expect(fixture.metrics.quorum_count).toBe(3);
    expect(fixture.metrics.affected_armed).toBe(3);
    expect(fixture.decisions[0].reason_code).toBe("SPEED_ENVELOPE_EXCEEDED");
  });

  it("reads MemorySnapshotV1 using the TypeScript contract", () => {
    const fixture = memoryJSON as MemorySnapshotV1;
    expect(fixture.embedding_version).toBe("all-MiniLM-L6-v2-onnx-v1");
    expect(fixture.sync[0].central_watermark).toBe(42);
    expect(fixture.memory_lab.enabled).toBe(true);
  });

  it("reads M13 platform proof and drill receipts", () => {
    const proof = platformProofJSON as PlatformProofSummaryV1;
    const drill = platformDrillJSON as PlatformDrillReceiptV1;
    expect(proof.planes).toHaveLength(4);
    expect(proof.slos.find((item)=>item.id==="duplicate-effects")?.measured).toBe(false);
    expect(drill.outcome).toBe("passed");
    expect(drill.expected_invariants).toContain("committed edge missions continue");
  });
  it("reads M14 full-program execution fixtures", () => {
    const fixture = executionV7JSON as { authority: { complete_program_onboard: boolean; authorized_time_remaining_seconds: number }; decision: { decision_node_id: string; decision_scope: string }; install_receipt: { state: string } };
    expect(fixture.authority.complete_program_onboard).toBe(true);
    expect(fixture.authority.authorized_time_remaining_seconds).toBeGreaterThan(60);
    expect(fixture.decision).toMatchObject({ decision_node_id: "node-a-01", decision_scope: "group" });
    expect(fixture.install_receipt.state).toBe("installed");
  });
});
