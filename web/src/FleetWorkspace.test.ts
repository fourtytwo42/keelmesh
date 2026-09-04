import { describe, expect, it } from "vitest";
import type { CommandSceneV1 } from "./types";
import { restorableCommandScenes } from "./FleetWorkspace";

const scene = (id: string, overrides: Partial<CommandSceneV1> = {}): CommandSceneV1 => ({
  schema_version: 1,
  id,
  actor_identity: "demo-operator",
  session_id: "session-test",
  type: "operational_brief",
  title: id,
  summary: id,
  spoken_summary: id,
  state: "active",
  pinned: false,
  critical: false,
  pending_approval: false,
  catalog_id: "keelmesh-operations-v1",
  workspace_version: 1,
  primary_surface: { id: `${id}-surface`, role: "primary", title: id, sequence: 1, messages: [] },
  supporting_surfaces: [],
  bindings: [],
  map_annotations: [],
  suggested_actions: [],
  receipts: [],
  created_at: "2026-09-04T00:00:00Z",
  updated_at: "2026-09-04T00:00:00Z",
  ...overrides,
});

describe("restorableCommandScenes", () => {
  it("restores only the newest ordinary scene plus protected scene lifecycles", () => {
    const restored = restorableCommandScenes([
      scene("newest"),
      scene("older"),
      scene("critical", { critical: true }),
      scene("approval", { pending_approval: true }),
      scene("pinned", { pinned: true, state: "replaced" }),
      scene("dismissed", { state: "dismissed" }),
    ]);

    expect(restored.map((item) => item.id)).toEqual(["newest", "critical", "approval", "pinned"]);
  });
});
