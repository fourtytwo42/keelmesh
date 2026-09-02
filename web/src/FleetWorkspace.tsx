import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { api, KeelMeshError, requestID } from "./api";
import type {
  AgentSnapshot,
  ArenaSnapshotV1,
  Bootstrap,
  CommandDraftV2,
  FleetLeaseV2,
  FleetPlanV2,
  FleetPreviewV2,
  FleetSnapshotV2,
  MissionWaypointV2,
  MissionWorkspaceV2,
  PlatformSnapshot,
  Point,
  ReachabilityV2,
  VesselProfileV2,
  VoiceV2,
} from "./types";
import { OperationsMap, type WaypointColor } from "./OperationsMap";
import { WindowManager, type WindowDefinition } from "./WindowManager";
import { EngineerView } from "./EngineerView";
import { PlatformCutaway } from "./PlatformCutaway";
import { ResilienceDrill } from "./ResilienceDrill";
import { QuietFleetDrill } from "./QuietFleetDrill";
import { ArenaView } from "./ArenaView";
import {
  Anchor,
  Ban,
  Bot,
  BoxSelect,
  CheckCircle2,
  Eye,
  GripVertical,
  ListFilter,
  MapPinned,
  Mic,
  MousePointer2,
  Network,
  Pause,
  Pencil,
  Play,
  Plus,
  Radio,
  Route,
  Save,
  Search,
  ShieldCheck,
  Ship,
  Skull,
  SlidersHorizontal,
  Sparkles,
  Swords,
  Trash2,
  Users,
  Volume2,
  Waves,
  X,
} from "lucide-react";

type Tool = "select" | "box" | "waypoint" | "include" | "exclude";
const formations = [
  "column",
  "line_abreast",
  "wedge",
  "echelon_left",
  "echelon_right",
  "parallel_columns",
  "dispersed_screen",
  "ring",
  "search_grid",
];

export function FleetWorkspace() {
  const [fleet, setFleet] = useState<FleetSnapshotV2 | null>(null),
    [legacy, setLegacy] = useState<Bootstrap | null>(null),
    [platform, setPlatform] = useState<PlatformSnapshot | null>(null),
    [agent, setAgent] = useState<AgentSnapshot | null>(null),
    [arena, setArena] = useState<ArenaSnapshotV1 | null>(null),
    [voices, setVoices] = useState<VoiceV2[]>([]),
    [voice, setVoice] = useState("morgan"),
    [speechState, setSpeechState] = useState("ready"),
    [selected, setSelected] = useState<Set<string>>(new Set()),
    [activeMissionID, setActiveMissionID] = useState<string>(""),
    [activeGroupID, setActiveGroupID] = useState<string>(""),
    [tool, setTool] = useState<Tool>("select"),
    [search, setSearch] = useState(""),
    [command, setCommand] = useState(
      "Search the selected area in a dispersed screen, avoid shallow water, and keep 35% reserve",
    ),
    [draft, setDraft] = useState<CommandDraftV2 | null>(null),
    [plans, setPlans] = useState<FleetPlanV2[]>([]),
    [planID, setPlanID] = useState(""),
    [preview, setPreview] = useState<FleetPreviewV2 | null>(null),
    [lease, setLease] = useState<FleetLeaseV2 | null>(null),
    [reachability, setReachability] = useState<ReachabilityV2 | null>(null),
    [windows, setWindows] = useState<Set<string>>(
      () =>
        new Set(
          window.location.search.includes("arena=1") ? ["arena"] : ["fleet"],
        ),
    ),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [connected, setConnected] = useState(true),
    [pendingDeleteID, setPendingDeleteID] = useState("");
  const audio = useRef<HTMLAudioElement | null>(null),
    speechAbort = useRef<AbortController | null>(null),
    recorder = useRef<MediaRecorder | null>(null),
    recordingStream = useRef<MediaStream | null>(null),
    recordingChunks = useRef<BlobPart[]>([]),
    stopRequested = useRef(false);
  const [inspectVesselID, setInspectVesselID] = useState(""),
    [windowActivations, setWindowActivations] = useState<
      Record<string, number>
    >({});
  const [pirate, setPirate] = useState(
    () => localStorage.getItem("keelmesh.theme") === "pirate",
  );
  const words = pirate
    ? {
        subtitle: "PIRATE FLEET COMMAND",
        simulation: "SEA TRIALS",
        vessels: "SHIPS",
        groups: "CREWS",
        active: "UNDERWAY",
        fleet: "Flotilla",
        mission: "Voyage",
        arena: "High Seas",
        resilience: "Storm Drill",
        quiet: "Silent Running",
        engineer: "Shipwright",
        cutaway: "Below Deck",
        selected: "MUSTERED",
        newMission: "New voyage",
        commandMission: "Command voyage",
        generate: "PLOT COURSES",
        noMission: "NO VOYAGE SELECTED",
        selectAssets: "Muster ships and chart a voyage",
        authority: "CAPTAIN'S AUTHORITY VERIFIED",
        fixture: "CHARTED-WATERS FIXTURE",
        advisory: "SHIP'S AI: ADVISORY",
      }
    : {
        subtitle: "MISSION OPERATIONS",
        simulation: "SIMULATION",
        vessels: "VESSELS",
        groups: "GROUPS",
        active: "ACTIVE",
        fleet: "Fleet",
        mission: "Mission",
        arena: "Fleet Arena",
        resilience: "Resilience",
        quiet: "Quiet Fleet",
        engineer: "Engineer",
        cutaway: "Cutaway",
        selected: "SELECTED",
        newMission: "New mission",
        commandMission: "Command mission",
        generate: "GENERATE OPTIONS",
        noMission: "NO MISSION SELECTED",
        selectAssets: "Select assets and create a mission",
        authority: "AUTHORITY HEALTHY",
        fixture: "NOAA-DERIVED FIXTURE",
        advisory: "NODE AI: ADVISORY",
      };
  useEffect(() => {
    localStorage.setItem("keelmesh.theme", pirate ? "pirate" : "navy");
    document.documentElement.dataset.theme = pirate ? "pirate" : "navy";
  }, [pirate]);
  const refresh = useCallback(async () => {
    const value = await api<FleetSnapshotV2>("/api/v2/fleet");
    setFleet(value);
  }, []);
  useEffect(() => {
    void refresh().catch((e) => setError(String(e)));
    Promise.allSettled([
      api<Bootstrap>("/api/v1/bootstrap").then(setLegacy),
      api<PlatformSnapshot>("/api/v1/platform").then(setPlatform),
      api<AgentSnapshot>("/api/v1/ai").then(setAgent),
      api<ArenaSnapshotV1>("/api/v3/arena?faction=A").then(setArena),
      api<{ voices: VoiceV2[] }>("/api/v2/voices").then((v) =>
        setVoices(v.voices),
      ),
    ]);
    const t = window.setInterval(() => {
      refresh()
        .then(() => setConnected(true))
        .catch(() => setConnected(false));
      api<ArenaSnapshotV1>("/api/v3/arena?faction=A")
        .then(setArena)
        .catch(() => {});
    }, 1000);
    return () => window.clearInterval(t);
  }, [refresh]);
  const rawMission =
    fleet?.missions.find((m) => m.id === activeMissionID) ??
    fleet?.missions[0] ??
    null;
  const mission = rawMission
    ? {
        ...rawMission,
        geometry: {
          ...rawMission.geometry,
          included_areas: rawMission.geometry.included_areas ?? [],
          exclusion_areas: rawMission.geometry.exclusion_areas ?? [],
          waypoints: rawMission.geometry.waypoints ?? [],
          waypoint_details:
            rawMission.geometry.waypoint_details ??
            (rawMission.geometry.waypoints ?? []).map((position, index) => ({
              id: `legacy-waypoint-${index + 1}`,
              position,
              color: "amber" as const,
              sequence: index + 1,
            })),
          pois: rawMission.geometry.pois ?? [],
        },
      }
    : null;
  const pendingDeleteMission =
    fleet?.missions.find((item) => item.id === pendingDeleteID) ?? null;
  useEffect(() => {
    if (!activeMissionID && fleet?.missions[0])
      setActiveMissionID(fleet.missions[0].id);
  }, [fleet, activeMissionID]);
  const activePlan =
    plans.find((p) => p.id === planID) ??
    plans.find((p) => p.recommended) ??
    null;
  const vesselsByID = useMemo(
    () => new Map(fleet?.vessels.map((v) => [v.id, v]) ?? []),
    [fleet],
  );
  const dragEndedAt = useRef(0);
  useEffect(() => {
    const mark = () => {
      dragEndedAt.current = performance.now();
    };
    document.addEventListener("dragend", mark, true);
    return () => document.removeEventListener("dragend", mark, true);
  }, []);
  const open = useCallback((id: string) => {
    setWindows((current) => new Set(current).add(id));
    setWindowActivations((current) => ({
      ...current,
      [id]: (current[id] ?? 0) + 1,
    }));
  }, []);
  const revealFleet = useCallback(() => open("fleet"), [open]);
  const select = useCallback(
    (ids: string[], mode: "replace" | "toggle") => {
      if (ids.length > 0) revealFleet();
      setSelected((old) => {
        if (mode === "replace") return new Set(ids);
        if (performance.now() - dragEndedAt.current < 250) return old;
        const next = new Set(old);
        for (const id of ids) next.has(id) ? next.delete(id) : next.add(id);
        return next;
      });
    },
    [revealFleet],
  );
  const selectGroup = useCallback(
    (gid: string) => {
      const group = fleet?.groups.find((g) => g.id === gid);
      if (group) {
        revealFleet();
        setSelected(new Set(group.member_ids));
        setActiveGroupID(gid);
      }
    },
    [fleet, revealFleet],
  );
  const filtered = useMemo(() => {
    const q = search.toLowerCase().trim();
    return (
      fleet?.vessels.filter(
        (v) =>
          !q ||
          [
            v.callsign,
            v.designation,
            v.group_code,
            v.class.name,
            v.telemetry.mode,
            v.telemetry.mission_id,
          ].some((x) => x?.toLowerCase().includes(q)),
      ) ?? []
    );
  }, [fleet, search]);
  const selectedGroup = useMemo(
    () =>
      fleet?.groups.find(
        (g) =>
          g.member_ids.length === selected.size &&
          g.member_ids.every((id) => selected.has(id)),
      ) ?? null,
    [fleet, selected],
  );
  const selectionMatchesMission =
    !!mission &&
    selected.size === mission.target_ids.length &&
    mission.target_ids.every((id) => selected.has(id));
  const inspectedVessel = inspectVesselID
    ? vesselsByID.get(inspectVesselID)
    : undefined;
  useEffect(() => {
    if (!inspectedVessel) {
      setReachability(null);
      return;
    }
    api<ReachabilityV2>(`/api/v2/vessels/${inspectedVessel.id}/reachability`)
      .then(setReachability)
      .catch(() => setReachability(null));
  }, [inspectedVessel]);
  async function mutate<T>(fn: () => Promise<T>) {
    setBusy(true);
    setError("");
    try {
      return await fn();
    } catch (e) {
      setError(
        e instanceof KeelMeshError ? `${e.code}: ${e.message}` : String(e),
      );
      throw e;
    } finally {
      setBusy(false);
    }
  }
  async function createMissionFor(
    targetIDs: string[],
    namingMode: "operator" | "ai" = "operator",
  ) {
    if (targetIDs.length === 0) {
      setError(
        pirate
          ? "Muster one or more ships first."
          : "Select one or more vessels first.",
      );
      return null;
    }
    const current = await api<FleetSnapshotV2>("/api/v2/fleet");
    const group = current.groups.find(
      (g) =>
        g.member_ids.length === targetIDs.length &&
        g.member_ids.every((id) => targetIDs.includes(id)),
    );
    const name =
      namingMode === "operator" && group
        ? `${group.code} · ${group.name} ${pirate ? "Voyage" : "Mission"}`
        : "";
    const objective =
      namingMode === "ai"
        ? command
        : group
          ? `${pirate ? "Crew" : "Operational group"} ${group.code} task`
          : pirate
            ? "New fleet undertaking"
            : "New fleet task";
    const m = await mutate(() =>
      api<MissionWorkspaceV2>("/api/v2/missions", {
        method: "POST",
        body: JSON.stringify({
          request_id: requestID("mission"),
          idempotency_key: requestID("mission-key"),
          expected_version: current.fleet_version,
          name,
          naming_mode: namingMode,
          objective,
          target_ids: targetIDs,
        }),
      }),
    ).catch(() => null);
    if (m) {
      await refresh();
      setActiveMissionID(m.id);
      setPlans([]);
      setDraft(null);
      setPreview(null);
      setLease(null);
      open("planner");
    }
    return m;
  }
  async function createMission() {
    await createMissionFor([...selected]);
  }
  async function createGroupFor(memberIDs: string[], name?: string) {
    if (memberIDs.length === 0) return null;
    const current = await api<FleetSnapshotV2>("/api/v2/fleet");
    const palette = [
      "#e5a23a",
      "#59bdd1",
      "#62c58e",
      "#b895d8",
      "#e36e62",
      "#ece8dc",
    ];
    const g = await mutate(() =>
      api<FleetSnapshotV2["groups"][number]>("/api/v2/groups", {
        method: "POST",
        body: JSON.stringify({
          request_id: requestID("group"),
          idempotency_key: requestID("group-key"),
          expected_version: current.fleet_version,
          name:
            name?.trim() ||
            `${pirate ? "Crew" : "Task Group"} ${current.groups.length - 7}`,
          color: palette[current.groups.length % palette.length],
          pattern: "chevron",
          member_ids: memberIDs,
        }),
      }),
    ).catch(() => null);
    if (g) {
      setActiveGroupID(g.id);
      await refresh();
    }
    return g;
  }
  async function createGroup() {
    const g = await createGroupFor([...selected]);
    if (g) open("group-manager");
  }
  async function createGroupFromVessel(vesselID: string, name: string) {
    const g = await createGroupFor([vesselID], name);
    if (g) open("group-manager");
  }
  async function patchGroup(
    id: string,
    values: { name: string; color: string; pattern: string },
  ) {
    const group = (await api<FleetSnapshotV2>("/api/v2/fleet")).groups.find(
      (g) => g.id === id,
    );
    if (!group) return;
    await mutate(() =>
      api(`/api/v2/groups/${id}`, {
        method: "PATCH",
        body: JSON.stringify({
          request_id: requestID("group-patch"),
          idempotency_key: requestID("group-patch-key"),
          expected_version: group.revision,
          ...values,
        }),
      }),
    );
    await refresh();
  }
  async function renameVessel(id: string, callsign: string) {
    const current = await api<FleetSnapshotV2>("/api/v2/fleet"),
      vessel = current.vessels.find((v) => v.id === id);
    if (!vessel || vessel.callsign === callsign.trim()) return;
    await mutate(() =>
      api(`/api/v2/vessels/${id}`, {
        method: "PATCH",
        body: JSON.stringify({
          request_id: requestID("vessel-rename"),
          idempotency_key: requestID("vessel-rename-key"),
          expected_version: current.fleet_version,
          callsign: callsign.trim(),
        }),
      }),
    );
    await refresh();
  }
  async function moveVessel(vesselID: string, groupID: string) {
    const current = await api<FleetSnapshotV2>("/api/v2/fleet");
    const group = current.groups.find((g) => g.id === groupID),
      vessel = current.vessels.find((v) => v.id === vesselID);
    if (!group || !vessel || vessel.group_id === groupID) return;
    await mutate(() =>
      api(`/api/v2/groups/${groupID}/members:move`, {
        method: "POST",
        body: JSON.stringify({
          request_id: requestID("group-move"),
          idempotency_key: requestID("group-move-key"),
          expected_version: group.revision,
          vessel_id: vesselID,
        }),
      }),
    );
    await refresh();
  }
  async function updateMission(patch: Record<string, unknown>) {
    if (!mission) return;
    await mutate(() =>
      api<MissionWorkspaceV2>(`/api/v2/missions/${mission.id}`, {
        method: "PATCH",
        body: JSON.stringify({
          request_id: requestID("mission-patch"),
          idempotency_key: requestID("mission-patch-key"),
          expected_version: mission.version,
          ...patch,
        }),
      }),
    );
    await refresh();
  }
  async function renameMission(id: string, name: string) {
    const current = (await api<FleetSnapshotV2>("/api/v2/fleet")).missions.find(
      (item) => item.id === id,
    );
    if (!current || current.name === name.trim()) return;
    await mutate(() =>
      api(`/api/v2/missions/${id}`, {
        method: "PATCH",
        body: JSON.stringify({
          request_id: requestID("mission-rename"),
          idempotency_key: requestID("mission-rename-key"),
          expected_version: current.version,
          name: name.trim(),
        }),
      }),
    );
    await refresh();
  }
  async function setMissionStatus(id: string, status: "paused" | "executing") {
    const current = (await api<FleetSnapshotV2>("/api/v2/fleet")).missions.find(
      (item) => item.id === id,
    );
    if (!current) return;
    await mutate(() =>
      api<MissionWorkspaceV2>(`/api/v2/missions/${id}`, {
        method: "PATCH",
        body: JSON.stringify({
          request_id: requestID(`mission-${status}`),
          idempotency_key: requestID(`mission-${status}-key`),
          expected_version: current.version,
          status,
        }),
      }),
    );
    await refresh();
  }
  function deleteMission(id: string) {
    setPendingDeleteID(id);
  }
  async function confirmMissionDelete() {
    const id = pendingDeleteID;
    if (!id) return;
    const snapshot = await api<FleetSnapshotV2>("/api/v2/fleet"),
      current = snapshot.missions.find((item) => item.id === id);
    if (!current) {
      setPendingDeleteID("");
      await refresh();
      return;
    }
    try {
      await mutate(() =>
        api(`/api/v2/missions/${id}`, {
          method: "DELETE",
          body: JSON.stringify({
            request_id: requestID("mission-delete"),
            idempotency_key: requestID("mission-delete-key"),
            expected_version: current.version,
          }),
        }),
      );
    } catch {
      return;
    }
    const updated = await api<FleetSnapshotV2>("/api/v2/fleet"),
      deletedActive = id === activeMissionID || id === mission?.id;
    setPendingDeleteID("");
    setFleet(updated);
    if (deletedActive) {
      setActiveMissionID(updated.missions[0]?.id ?? "");
      setPlans([]);
      setDraft(null);
      setPreview(null);
      setLease(null);
      if (updated.missions.length === 0)
        setWindows((value) => {
          const next = new Set(value);
          next.delete("planner");
          return next;
        });
    }
  }
  function waypointEntries(value: MissionWorkspaceV2) {
    return value.geometry.waypoints.map(
      (position, index) =>
        value.geometry.waypoint_details?.[index] ?? {
          id: `waypoint-${value.id}-${index + 1}`,
          position,
          color: "amber" as const,
          sequence: index + 1,
        },
    );
  }
  async function saveWaypoints(
    target: MissionWorkspaceV2,
    entries: MissionWaypointV2[],
  ) {
    const current = (await api<FleetSnapshotV2>("/api/v2/fleet")).missions.find(
      (m) => m.id === target.id,
    );
    if (!current) return null;
    const normalized = entries.map((entry, index) => ({
      ...entry,
      position: entry.position,
      sequence: index + 1,
    }));
    const updated = await mutate(() =>
      api<MissionWorkspaceV2>(`/api/v2/missions/${current.id}/geometry`, {
        method: "POST",
        body: JSON.stringify({
          request_id: requestID("geometry"),
          idempotency_key: requestID("geometry-key"),
          expected_version: current.version,
          included_areas: current.geometry.included_areas,
          exclusion_areas: current.geometry.exclusion_areas,
          waypoints: normalized.map((entry) => entry.position),
          waypoint_details: normalized,
          pois: current.geometry.pois,
        }),
      }),
    ).catch(() => null);
    if (updated) {
      setPlans([]);
      setDraft(null);
      setPreview(null);
      setLease(null);
      await refresh();
    }
    return updated;
  }
  async function missionForWaypoint() {
    if (mission) return mission;
    if (selected.size > 0) return createMissionFor([...selected]);
    setError(
      pirate
        ? "Muster ships before dropping a bearing mark."
        : "Select vessels or create a mission before adding a waypoint.",
    );
    return null;
  }
  async function addWaypoint(p: Point, color: WaypointColor) {
    const target = await missionForWaypoint();
    if (!target) return;
    const entries = waypointEntries(target);
    entries.push({
      id: requestID("waypoint"),
      position: p,
      color,
      sequence: entries.length + 1,
    });
    await saveWaypoints(target, entries);
  }
  async function deleteWaypoint(index: number) {
    if (!mission) return;
    const entries = waypointEntries(mission);
    if (index < 0 || index >= entries.length) return;
    entries.splice(index, 1);
    await saveWaypoints(mission, entries);
  }
  async function clearWaypoints(color?: WaypointColor) {
    if (!mission) return;
    const entries = color
      ? waypointEntries(mission).filter((entry) => entry.color !== color)
      : [];
    await saveWaypoints(mission, entries);
  }
  async function goToLocation(p: Point, color: WaypointColor) {
    let target: MissionWorkspaceV2 | null = selectionMatchesMission
      ? mission
      : null;
    if (!target && selected.size > 0)
      target = await createMissionFor([...selected]);
    if (!target) {
      setError(
        pirate
          ? "Muster ships before ordering a course."
          : "Select one or more vessels before choosing Go to location.",
      );
      return;
    }
    const entry: MissionWaypointV2 = {
      id: requestID("waypoint"),
      position: p,
      color,
      sequence: 1,
    };
    const updated = await saveWaypoints(target, [entry]);
    if (!updated) return;
    const text = `Navigate the selected assets to the ${color} waypoint 1, then hold position.`;
    setCommand(text);
    await createPlans(updated, text);
  }
  async function useWaypointColor(color: WaypointColor) {
    if (!mission) return;
    const text = `Navigate the selected assets through the ${color} waypoints in numbered order.`;
    setCommand(text);
    await createPlans(mission, text);
  }
  async function addPolygon(kind: "include" | "exclude", poly: Point[]) {
    if (!mission) return;
    await mutate(() =>
      api(`/api/v2/missions/${mission.id}/geometry`, {
        method: "POST",
        body: JSON.stringify({
          request_id: requestID("geometry"),
          idempotency_key: requestID("geometry-key"),
          expected_version: mission.version,
          included_areas:
            kind === "include"
              ? [...mission.geometry.included_areas, poly]
              : mission.geometry.included_areas,
          exclusion_areas:
            kind === "exclude"
              ? [...mission.geometry.exclusion_areas, poly]
              : mission.geometry.exclusion_areas,
          waypoints: mission.geometry.waypoints,
          waypoint_details: mission.geometry.waypoint_details,
          pois: mission.geometry.pois,
        }),
      }),
    );
    await refresh();
  }
  async function createPlans(
    targetMission: MissionWorkspaceV2 | null,
    intent = command,
  ) {
    if (!targetMission) return;
    const compiled = await mutate(() =>
      api<CommandDraftV2>(
        `/api/v2/missions/${targetMission.id}/commands:compile`,
        {
          method: "POST",
          body: JSON.stringify({
            request_id: requestID("compile"),
            idempotency_key: requestID("compile-key"),
            expected_version: targetMission.version,
            text: intent,
            target_ids: targetMission.target_ids,
            formation: targetMission.formation,
          }),
        },
      ),
    ).catch(() => null);
    if (!compiled) return;
    setDraft(compiled);
    const current = (await api<FleetSnapshotV2>("/api/v2/fleet")).missions.find(
      (m) => m.id === targetMission.id,
    );
    if (!current) return;
    const result = await mutate(() =>
      api<{ plans: FleetPlanV2[] }>(
        `/api/v2/missions/${targetMission.id}/plans`,
        {
          method: "POST",
          body: JSON.stringify({
            request_id: requestID("plans"),
            idempotency_key: requestID("plans-key"),
            expected_version: current.version,
            draft_id: compiled.id,
          }),
        },
      ),
    ).catch(() => null);
    if (result) {
      setPlans(result.plans);
      setPlanID(
        (result.plans.find((p) => p.recommended) ?? result.plans[0]).id,
      );
      setPreview(null);
      setLease(null);
      open("planner");
      await refresh();
    }
  }
  async function generateOptions() {
    let target: MissionWorkspaceV2 | null = selectionMatchesMission
      ? mission
      : null;
    if (!target && selected.size > 0) {
      const targetIDs = [...selected];
      target =
        fleet?.missions.find(
          (m) =>
            m.status !== "completed" &&
            m.target_ids.length === targetIDs.length &&
            m.target_ids.every((id) => selected.has(id)),
        ) ?? (await createMissionFor(targetIDs, "ai"));
      if (target) setActiveMissionID(target.id);
    }
    if (!target) {
      setError(
        pirate
          ? "Muster ships or select a voyage first."
          : "Select vessels or an existing mission first.",
      );
      return;
    }
    await createPlans(target);
  }
  async function previewPlan() {
    if (!mission || !activePlan) return;
    const current = (await api<FleetSnapshotV2>("/api/v2/fleet")).missions.find(
      (m) => m.id === mission.id,
    )!;
    const value = await mutate(() =>
      api<FleetPreviewV2>(
        `/api/v2/missions/${mission.id}/plans/${activePlan.id}:preview`,
        {
          method: "POST",
          body: JSON.stringify({
            request_id: requestID("preview"),
            idempotency_key: requestID("preview-key"),
            expected_version: current.version,
          }),
        },
      ),
    ).catch(() => null);
    if (value) setPreview(value);
  }
  async function authorize() {
    if (!mission || !activePlan) return;
    const current = (await api<FleetSnapshotV2>("/api/v2/fleet")).missions.find(
      (m) => m.id === mission.id,
    )!;
    const value = await mutate(() =>
      api<FleetLeaseV2>(
        `/api/v2/missions/${mission.id}/plans/${activePlan.id}:authorize`,
        {
          method: "POST",
          body: JSON.stringify({
            request_id: requestID("authorize"),
            idempotency_key: requestID("authorize-key"),
            expected_version: current.version,
            plan_hash: activePlan.content_hash,
            operator_id: "demo-operator",
          }),
        },
      ),
    ).catch(() => null);
    if (value) {
      setLease(value);
      await refresh();
    }
  }
  async function start() {
    if (!mission || !activePlan || !lease) return;
    const current = (await api<FleetSnapshotV2>("/api/v2/fleet")).missions.find(
      (m) => m.id === mission.id,
    )!;
    await mutate(() =>
      api(`/api/v2/missions/${mission.id}/plans/${activePlan.id}:start`, {
        method: "POST",
        body: JSON.stringify({
          request_id: requestID("start"),
          idempotency_key: `start-${lease.id}`,
          expected_version: current.version,
          plan_hash: activePlan.content_hash,
          lease_id: lease.id,
        }),
      }),
    );
    await refresh();
  }
  async function speak(text: string) {
    speechAbort.current?.abort();
    audio.current?.pause();
    const controller = new AbortController();
    speechAbort.current = controller;
    setSpeechState("synthesizing");
    try {
      const response = await fetch("/api/v2/speech:synthesize", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ request_id: requestID("speech"), voice, text }),
        signal: controller.signal,
      });
      if (!response.ok)
        throw new Error(
          "Pocket TTS unavailable — visible text remains active.",
        );
      const blob = await response.blob();
      const player = new Audio(URL.createObjectURL(blob));
      audio.current = player;
      player.onended = () => setSpeechState("ready");
      setSpeechState("speaking");
      await player.play();
    } catch (e) {
      if ((e as Error).name !== "AbortError") {
        setSpeechState("text fallback");
        setError((e as Error).message);
      }
    }
  }
  async function beginTranscription() {
    if (recorder.current?.state === "recording") return;
    stopRequested.current = false;
    setError("");
    setSpeechState("requesting microphone");
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          channelCount: 1,
          echoCancellation: true,
          noiseSuppression: true,
          autoGainControl: true,
          sampleRate: { ideal: 48000 },
          sampleSize: { ideal: 16 },
        },
      });
      if (stopRequested.current) {
        stream.getTracks().forEach((track) => track.stop());
        setSpeechState("ready");
        return;
      }
      recordingStream.current = stream;
      recordingChunks.current = [];
      const preferred = ["audio/webm;codecs=opus", "audio/webm"].find(
        MediaRecorder.isTypeSupported,
      );
      const active = new MediaRecorder(
        stream,
        preferred
          ? { mimeType: preferred, audioBitsPerSecond: 128000 }
          : undefined,
      );
      recorder.current = active;
      active.ondataavailable = (e) => {
        if (e.data.size) recordingChunks.current.push(e.data);
      };
      active.onstop = async () => {
        stream.getTracks().forEach((track) => track.stop());
        recordingStream.current = null;
        const blob = new Blob(recordingChunks.current, {
          type: active.mimeType || "audio/webm",
        });
        setSpeechState("transcribing on VM node");
        try {
          const response = await fetch(
            `/api/v2/transcription?request_id=${encodeURIComponent(requestID("stt"))}`,
            {
              method: "POST",
              headers: { "Content-Type": blob.type },
              body: blob,
            },
          );
          const result = (await response.json()) as {
            text?: string;
            route?: string;
            real_time_factor?: number;
            message?: string;
          };
          if (!response.ok || !result.text)
            throw new Error(
              result.message ??
                "No speech was detected; typed input remains available.",
            );
          setCommand(result.text);
          setSpeechState(
            `${result.route} · RTF ${result.real_time_factor ?? "—"}`,
          );
        } catch (e) {
          setSpeechState("typed fallback");
          setError((e as Error).message);
        }
      };
      active.start(200);
      setSpeechState("listening · release to transcribe");
    } catch {
      setSpeechState("typed fallback");
      setError(
        "Microphone access requires HTTPS. Use the ephemeral Cloudflare tunnel or typed input.",
      );
    }
  }
  function endTranscription() {
    stopRequested.current = true;
    if (recorder.current?.state === "recording") recorder.current.stop();
  }
  if (!fleet)
    return (
      <main className="m6-loading">
        <b>KM</b>
        <span>Loading operations workspace…</span>
      </main>
    );
  const defs: WindowDefinition[] = [];
  if (windows.has("fleet"))
    defs.push({
      id: "fleet",
      kind: "primary",
      title: pirate ? "Flotilla / Crews" : "Fleet / Groups",
      icon: <Ship />,
      activation: windowActivations.fleet,
      initial: { x: 10, y: 92, width: 320, height: 600 },
      content: (
        <FleetRail
          pirate={pirate}
          fleet={fleet}
          filtered={filtered}
          selected={selected}
          search={search}
          onSearch={setSearch}
          onSelect={select}
          onGroup={selectGroup}
          onManage={(id) => {
            setActiveGroupID(id);
            open("group-manager");
          }}
          onInspect={(id) => {
            setInspectVesselID(id);
            open("inspector");
          }}
          onMove={moveVessel}
          onCreateGroup={createGroup}
          onCreateGroupFromVessel={createGroupFromVessel}
          onCreateMission={createMission}
        />
      ),
    });
  const activeGroup = fleet.groups.find((g) => g.id === activeGroupID);
  if (windows.has("group-manager") && activeGroup)
    defs.push({
      id: "group-manager",
      kind: "context",
      activation: windowActivations["group-manager"],
      title: `${pirate ? "Crew" : "Group"} · ${activeGroup.code}`,
      icon: <Users />,
      initial: { x: 350, y: 120, width: 340, height: 390 },
      content: (
        <GroupManager
          key={`${activeGroup.id}-${activeGroup.revision}`}
          group={activeGroup}
          vessels={vesselsByID}
          onSave={(v) => patchGroup(activeGroup.id, v)}
        />
      ),
    });
  if (windows.has("inspector") && inspectedVessel)
    defs.push({
      id: "inspector",
      kind: "context",
      activation: windowActivations.inspector,
      title: inspectedVessel.display_name,
      icon: <Eye />,
      initial: { x: 350, y: 92, width: 350, height: 590 },
      content: (
        <VesselInspector
          vessel={inspectedVessel}
          reachability={reachability}
          lookup={vesselsByID}
          onRename={(name) => renameVessel(inspectedVessel.id, name)}
        />
      ),
    });
  if (windows.has("planner"))
    defs.push({
      id: "planner",
      kind: "primary",
      activation: windowActivations.planner,
      title: pirate ? "Voyage Plotter" : "Mission Planner",
      icon: <Route />,
      initial: { x: window.innerWidth - 440, y: 92, width: 420, height: 650 },
      content: (
        <Planner
          pirate={pirate}
          mission={mission}
          draft={draft}
          command={command}
          setCommand={setCommand}
          plans={plans}
          activePlan={activePlan}
          preview={preview}
          lease={lease}
          busy={busy}
          voices={voices}
          voice={voice}
          speechState={speechState}
          onVoice={setVoice}
          onSpeak={() =>
            speak(
              activePlan
                ? `${activePlan.name}. ${activePlan.maneuvers.join(". ")}. Minimum projected reserve ${Math.round(activePlan.minimum_reserve * 100)} percent.`
                : command,
            )
          }
          onFormation={(f) => updateMission({ formation: f })}
          onArea={(kind) => setTool(kind)}
          onTool={setTool}
          onCreate={() => createPlans(mission)}
          onPlan={setPlanID}
          onPreview={previewPlan}
          onAuthorize={authorize}
          onStart={start}
          onStatus={(status) => mission && setMissionStatus(mission.id, status)}
          onRename={(name) => mission && renameMission(mission.id, name)}
          onDelete={() => mission && deleteMission(mission.id)}
        />
      ),
    });
  if (windows.has("constraints") && mission)
    defs.push({
      id: "constraints",
      kind: "context",
      activation: windowActivations.constraints,
      title: pirate ? "Captain's Standing Orders" : "Effective Constraints",
      icon: <SlidersHorizontal />,
      initial: { x: 410, y: 130, width: 360, height: 570 },
      content: (
        <Constraints
          mission={mission}
          onSave={(c) => updateMission({ constraints: c })}
        />
      ),
    });
  if (windows.has("engineer") && agent)
    defs.push({
      id: "engineer",
      kind: "primary",
      activation: windowActivations.engineer,
      title: pirate ? "Fleet Shipwright" : "Autonomy Engineer",
      icon: <Bot />,
      initial: {
        x: 120,
        y: 82,
        width: Math.min(1000, window.innerWidth - 180),
        height: Math.min(650, window.innerHeight - 150),
      },
      content: (
        <EngineerView value={agent} onChange={setAgent} onError={setError} />
      ),
    });
  if (windows.has("cutaway") && platform && legacy)
    defs.push({
      id: "cutaway",
      kind: "primary",
      activation: windowActivations.cutaway,
      title: pirate ? "Below Deck Systems" : "Live Infrastructure Cutaway",
      icon: <Network />,
      initial: {
        x: 80,
        y: 82,
        width: Math.min(1180, window.innerWidth - 130),
        height: Math.min(650, window.innerHeight - 150),
      },
      content: (
        <PlatformCutaway
          value={platform}
          fleet={legacy.snapshot}
          onError={setError}
        />
      ),
    });
  if (windows.has("resilience") && legacy)
    defs.push({
      id: "resilience",
      kind: "primary",
      activation: windowActivations.resilience,
      title: pirate ? "Storm Drill" : "Resilience Drill",
      icon: <ShieldCheck />,
      initial: { x: 370, y: 110, width: 370, height: 580 },
      content: legacy.snapshot.resilience ? (
        <ResilienceDrill
          value={legacy.snapshot.resilience}
          onChange={(v) =>
            setLegacy((l) =>
              l ? { ...l, snapshot: { ...l.snapshot, resilience: v } } : l,
            )
          }
          onError={setError}
        />
      ) : (
        <div className="window-empty">
          Start the M1 compatibility mission to initialize the deterministic
          Vessel 4 drill.
        </div>
      ),
    });
  if (windows.has("quiet") && legacy)
    defs.push({
      id: "quiet",
      kind: "primary",
      activation: windowActivations.quiet,
      title: pirate ? "Silent Running" : "Quiet Fleet",
      icon: <Radio />,
      initial: { x: 420, y: 120, width: 390, height: 530 },
      content: legacy.snapshot.quiet_fleet ? (
        <QuietFleetDrill
          value={legacy.snapshot.quiet_fleet}
          onChange={(v) =>
            setLegacy((l) =>
              l ? { ...l, snapshot: { ...l.snapshot, quiet_fleet: v } } : l,
            )
          }
          onError={setError}
        />
      ) : (
        <div className="window-empty">
          Start an authorized compatibility mission to initialize Quiet Fleet
          coordination.
        </div>
      ),
    });
  if (windows.has("arena") && arena)
    defs.push({
      id: "arena",
      kind: "primary",
      activation: windowActivations.arena,
      title: pirate
        ? "High Seas · Distributed Fleet"
        : "Fleet Arena · Distributed Node Fabric",
      icon: <Swords />,
      initial: {
        x: 55,
        y: 82,
        width: Math.min(1180, window.innerWidth - 100),
        height: Math.min(650, window.innerHeight - 150),
      },
      content: (
        <ArenaView
          pirate={pirate}
          value={arena}
          onChange={setArena}
          onError={setError}
        />
      ),
    });
  return (
    <main className="m6-shell">
      <header className="ops-bar">
        <div className="ops-brand">
          <b>{pirate ? <Skull /> : "KM"}</b>
          <span>
            <strong>KEELMESH</strong>
            <small>{words.subtitle}</small>
          </span>
        </div>
        <div className="status-cluster">
          <span className="sim">{words.simulation}</span>
          <span>
            {fleet.vessels.length} {words.vessels}
          </span>
          <span>
            {fleet.groups.length} {words.groups}
          </span>
          <span>
            {fleet.missions.filter((m) => m.status === "executing").length}{" "}
            {words.active}
          </span>
        </div>
        <nav>
          <button onClick={revealFleet}>
            <Ship />
            {words.fleet}
          </button>
          <button onClick={() => open("planner")}>
            <Route />
            {words.mission}
          </button>
          <button className="arena-nav" onClick={() => open("arena")}>
            <Swords />
            {words.arena}
          </button>
          <button onClick={() => open("resilience")}>
            <ShieldCheck />
            {words.resilience}
          </button>
          <button onClick={() => open("quiet")}>
            <Radio />
            {words.quiet}
          </button>
          <button onClick={() => open("engineer")}>
            <Bot />
            {words.engineer}
          </button>
          <button onClick={() => open("cutaway")}>
            <Network />
            {words.cutaway}
          </button>
        </nav>
        <button
          className="theme-toggle"
          aria-label={pirate ? "Return to navy mode" : "Enter pirate mode"}
          title={pirate ? "Return to Navy mode" : "Hoist the pirate colors"}
          onClick={() => setPirate((v) => !v)}
        >
          {pirate ? <Anchor /> : <Skull />}
        </button>
        <div className={`ops-live ${connected ? "on" : "off"}`}>
          <i />
          {connected
            ? pirate
              ? "SHIPSHAPE"
              : "LIVE"
            : pirate
              ? "SIGNAL LOST"
              : "RECONNECTING"}
        </div>
      </header>
      <div className="mission-tabs">
        <button
          className="new-tab"
          aria-label={words.newMission}
          onClick={createMission}
        >
          <Plus />
        </button>
        {fleet.missions.map((m) => (
          <div
            key={m.id}
            className={`mission-tab ${m.id === mission?.id ? "active" : ""}`}
          >
            <button
              className="mission-tab-main"
              onClick={() => {
                setActiveMissionID(m.id);
                setPlans([]);
                setDraft(null);
                setPreview(null);
                setLease(null);
                open("planner");
              }}
            >
              <i className={m.status} />
              <span>{m.name}</span>
              <small>{m.status}</small>
            </button>
            <div className="mission-tab-actions">
              <button
                aria-label={`${m.status === "paused" ? "Resume" : "Pause"} ${m.name}`}
                title={
                  m.status === "paused" ? "Resume mission" : "Pause mission"
                }
                disabled={m.status !== "executing" && m.status !== "paused"}
                onClick={() =>
                  void setMissionStatus(
                    m.id,
                    m.status === "paused" ? "executing" : "paused",
                  )
                }
              >
                {m.status === "paused" ? <Play /> : <Pause />}
              </button>
              <button
                className="delete"
                aria-label={`Delete ${m.name}`}
                title="Delete mission"
                onClick={() => void deleteMission(m.id)}
              >
                <Trash2 />
              </button>
            </div>
          </div>
        ))}
      </div>
      <OperationsMap
        pirate={pirate}
        fleet={fleet}
        mission={mission}
        selected={selected}
        activePlan={activePlan}
        tool={tool}
        onSelect={(ids, mode) => {
          select(ids, mode);
          if (ids.length === 1) {
            setInspectVesselID(ids[0]);
            open("inspector");
          }
        }}
        onGroup={selectGroup}
        onWaypoint={addWaypoint}
        onDeleteWaypoint={deleteWaypoint}
        onClearWaypoints={clearWaypoints}
        onGoTo={goToLocation}
        onUseWaypointColor={useWaypointColor}
        onArea={addPolygon}
        onToolDone={() => setTool("select")}
      />
      <div className="map-tools">
        <button
          className={tool === "select" ? "active" : ""}
          onClick={() => setTool("select")}
          title="Select"
        >
          <MousePointer2 />
        </button>
        <button
          className={tool === "box" ? "active" : ""}
          onClick={() => setTool("box")}
          title="Rectangle select"
        >
          <BoxSelect />
        </button>
        <button
          className={tool === "waypoint" ? "active" : ""}
          onClick={() => setTool("waypoint")}
          title="Add waypoint"
        >
          <MapPinned />
        </button>
        <button
          className={tool === "include" ? "active" : ""}
          onClick={() => setTool("include")}
          title="Drag operating area"
        >
          <Plus />
          <BoxSelect />
        </button>
        <button
          className={tool === "exclude" ? "active" : ""}
          onClick={() => setTool("exclude")}
          title="Drag exclusion area"
        >
          <Ban />
        </button>
      </div>
      <section
        className={`intent-dock ${selectedGroup && !selectionMatchesMission ? "target-pending" : ""}`}
      >
        <div>
          <small>
            {selectedGroup
              ? `${selectedGroup.code} ${selectedGroup.name.toUpperCase()} · ${selectedGroup.member_ids.length} ${pirate ? "CREW" : "GROUP ASSETS"}`
              : selected.size > 0
                ? `${selected.size} ${pirate ? "MUSTERED" : "SELECTED"} · ${pirate ? "COURSE TARGET" : "PLAN TARGET"}`
                : mission
                  ? `${mission.name.toUpperCase()} · ${mission.target_ids.length} ${pirate ? "HANDS" : "ASSETS"}`
                  : words.noMission}
          </small>
          <strong>
            {selectedGroup && !selectionMatchesMission
              ? pirate
                ? "Ready to plot courses for this crew"
                : "Ready to generate options for this operational group"
              : (mission?.objective ?? words.selectAssets)}
          </strong>
        </div>
        <Route />
        <input
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") void generateOptions();
          }}
        />
        <button
          className={`mic ${speechState.startsWith("listening") ? "recording" : ""}`}
          title="Hold to talk · routed to node-local STT"
          aria-label="Hold to talk"
          onPointerDown={() => void beginTranscription()}
          onPointerUp={endTranscription}
          onPointerCancel={endTranscription}
          onContextMenu={(e) => e.preventDefault()}
        >
          <Mic />
        </button>
        <button
          onClick={() => void generateOptions()}
          disabled={(!mission && selected.size === 0) || busy}
        >
          <Sparkles />
          {busy ? (pirate ? "PLOTTING…" : "WORKING…") : words.generate}
        </button>
      </section>
      {pendingDeleteMission && (
        <div className="mission-delete-backdrop">
          <section
            className="mission-delete-dialog"
            role="dialog"
            aria-modal="true"
            aria-labelledby="mission-delete-title"
          >
            <header>
              <Trash2 />
              <div>
                <small>{pirate ? "SCUTTLE VOYAGE" : "DELETE MISSION"}</small>
                <h2 id="mission-delete-title">
                  Delete {pendingDeleteMission.name}?
                </h2>
              </div>
            </header>
            <p>
              This permanently removes the mission workspace, ends its simulated
              movement authority, and releases{" "}
              {pendingDeleteMission.target_ids.length}{" "}
              {pendingDeleteMission.target_ids.length === 1
                ? "vessel"
                : "vessels"}
              .
            </p>
            <div>
              <button autoFocus onClick={() => setPendingDeleteID("")}>
                {pirate ? "Belay that" : "Cancel"}
              </button>
              <button
                className="danger"
                disabled={busy}
                onClick={() => void confirmMissionDelete()}
              >
                <Trash2 />
                {busy
                  ? pirate
                    ? "SCUTTLING…"
                    : "DELETING…"
                  : pirate
                    ? "Scuttle voyage"
                    : "Delete mission"}
              </button>
            </div>
          </section>
        </div>
      )}
      {error && (
        <div className="ops-error" role="alert">
          <b>!</b>
          <span>{error}</span>
          <button aria-label="Dismiss alert" onClick={() => setError("")}>
            <X />
          </button>
        </div>
      )}
      <WindowManager windows={defs} />
      <footer className="ops-status">
        <span>
          <i className="green" />
          <CheckCircle2 />
          {words.authority}
        </span>
        <span>
          <Radio />
          STARLINK + HALOW
        </span>
        <span>
          <Waves />
          {words.fixture}
        </span>
        <span>
          <Bot />
          {words.advisory}
        </span>
        <span>{new Date(fleet.generated_at).toLocaleTimeString()}</span>
      </footer>
    </main>
  );
}

type VesselGroupMenuState = { vessel: VesselProfileV2; x: number; y: number };
function VesselGroupContextMenu({
  pirate,
  state,
  groups,
  onMove,
  onCreate,
  onClose,
}: {
  pirate: boolean;
  state: VesselGroupMenuState;
  groups: FleetSnapshotV2["groups"];
  onMove: (vesselID: string, groupID: string) => void;
  onCreate: (vesselID: string, name: string) => void;
  onClose: () => void;
}) {
  const [creating, setCreating] = useState(false),
    [name, setName] = useState("");
  useEffect(() => {
    const close = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    window.addEventListener("keydown", close);
    return () => window.removeEventListener("keydown", close);
  }, [onClose]);
  const x = Math.min(state.x, window.innerWidth - 238),
    y = Math.min(state.y, window.innerHeight - 330);
  return createPortal(
    <div
      className="context-menu-scrim"
      onPointerDown={onClose}
      onContextMenu={(e) => {
        e.preventDefault();
        onClose();
      }}
    >
      <div
        className="vessel-group-menu"
        role="menu"
        aria-label={`Assign ${state.vessel.callsign} to group`}
        style={{ left: Math.max(6, x), top: Math.max(6, y) }}
        onPointerDown={(e) => e.stopPropagation()}
        onContextMenu={(e) => e.stopPropagation()}
      >
        <header>
          <strong>{pirate ? "ASSIGN SHIP" : "ASSIGN VESSEL"}</strong>
          <span>
            {state.vessel.callsign} · {state.vessel.designation}
          </span>
        </header>
        <div className="group-menu-list">
          {groups.map((group) => (
            <button
              role="menuitem"
              key={group.id}
              disabled={group.id === state.vessel.group_id}
              onClick={() => {
                onMove(state.vessel.id, group.id);
                onClose();
              }}
            >
              <i style={{ background: group.color }} />
              <span>
                <b>{group.code}</b>
                {group.name}
              </span>
              <small>
                {group.id === state.vessel.group_id
                  ? pirate
                    ? "current crew"
                    : "current group"
                  : group.member_ids.length}
              </small>
            </button>
          ))}
        </div>
        {creating ? (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              if (name.trim()) {
                onCreate(state.vessel.id, name.trim());
                onClose();
              }
            }}
          >
            <input
              autoFocus
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={pirate ? "New crew name" : "New group name"}
            />
            <button type="submit" disabled={!name.trim()}>
              {pirate ? "Form crew" : "Create"}
            </button>
          </form>
        ) : (
          <button
            className="create-group-menu"
            role="menuitem"
            onClick={() => setCreating(true)}
          >
            <Plus />
            {pirate
              ? "Form new crew with this ship"
              : "Create new group with this vessel"}
          </button>
        )}
      </div>
    </div>,
    document.body,
  );
}
function FleetRail({
  pirate,
  fleet,
  filtered,
  selected,
  search,
  onSearch,
  onSelect,
  onGroup,
  onManage,
  onInspect,
  onMove,
  onCreateGroup,
  onCreateGroupFromVessel,
  onCreateMission,
}: {
  pirate: boolean;
  fleet: FleetSnapshotV2;
  filtered: VesselProfileV2[];
  selected: Set<string>;
  search: string;
  onSearch: (s: string) => void;
  onSelect: (ids: string[], m: "replace" | "toggle") => void;
  onGroup: (id: string) => void;
  onManage: (id: string) => void;
  onInspect: (id: string) => void;
  onMove: (vesselID: string, groupID: string) => void;
  onCreateGroup: () => void;
  onCreateGroupFromVessel: (vesselID: string, name: string) => void;
  onCreateMission: () => void;
}) {
  const [dropGroup, setDropGroup] = useState(""),
    [menu, setMenu] = useState<VesselGroupMenuState | null>(null);
  const drop = (event: React.DragEvent, groupID: string) => {
    event.preventDefault();
    event.stopPropagation();
    setDropGroup("");
    const vesselID = event.dataTransfer.getData(
      "application/x-keelmesh-vessel",
    );
    if (vesselID) onMove(vesselID, groupID);
  };
  return (
    <div className="fleet-rail">
      <div className="rail-search">
        <span>
          <Search />
        </span>
        <input
          placeholder={
            pirate
              ? "Callsign, class, crew, bearing…"
              : "Callsign, class, group, status…"
          }
          value={search}
          onChange={(e) => onSearch(e.target.value)}
        />
      </div>
      <div className="rail-actions">
        <button
          onClick={() =>
            onSelect(
              filtered.map((v) => v.id),
              "replace",
            )
          }
        >
          <ListFilter />
          {pirate ? "Muster filtered" : "Select all filtered"}
        </button>
        <button onClick={() => onSelect([], "replace")}>
          <X />
          {pirate ? "Dismiss" : "Clear"}
        </button>
        <button onClick={onCreateGroup} disabled={selected.size === 0}>
          <Users />
          {pirate ? "New crew" : "New group"}
        </button>
      </div>
      <div className="group-list">
        {fleet.groups.map((g) => (
          <section
            key={g.id}
            className={dropGroup === g.id ? "drop-active" : ""}
            onDragOver={(e) => {
              e.preventDefault();
              e.dataTransfer.dropEffect = "move";
              setDropGroup(g.id);
            }}
            onDragLeave={(e) => {
              if (!e.currentTarget.contains(e.relatedTarget as Node))
                setDropGroup("");
            }}
            onDrop={(e) => drop(e, g.id)}
            title={`Drop a vessel into ${g.code} ${g.name}`}
          >
            <div className="group-row-wrap">
              <button
                aria-label={`${g.code} ${g.name}`}
                className="group-row"
                onDoubleClick={() => onGroup(g.id)}
                onClick={() => onGroup(g.id)}
              >
                <i style={{ background: g.color }} />
                <b>{g.code}</b>
                <span>{g.name}</span>
                <small>
                  {g.member_ids.filter((id) => selected.has(id)).length}/
                  {g.member_ids.length}
                </small>
              </button>
              <button
                className="group-view"
                aria-label={`View status of ${g.code} ${g.name}`}
                title={`View ${g.code} group status`}
                onClick={() => onManage(g.id)}
              >
                <Eye />
              </button>
            </div>
            {filtered
              .filter((v) => v.group_id === g.id)
              .map((v) => (
                <div
                  draggable
                  onDragStart={(e) => {
                    e.dataTransfer.effectAllowed = "move";
                    e.dataTransfer.setData(
                      "application/x-keelmesh-vessel",
                      v.id,
                    );
                  }}
                  onContextMenu={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    setMenu({ vessel: v, x: e.clientX, y: e.clientY });
                  }}
                  className={`fleet-vessel-row ${selected.has(v.id) ? "selected" : ""}`}
                  key={v.id}
                >
                  <input
                    aria-label={`Select ${v.callsign}`}
                    type="checkbox"
                    checked={selected.has(v.id)}
                    onChange={() => onSelect([v.id], "toggle")}
                  />
                  <i style={{ borderColor: v.group_color }}>
                    <img src={`/assets/vessels/${v.class.id}.png`} />
                  </i>
                  <span>
                    <b>{v.callsign}</b>
                    <small>
                      {v.designation} · {v.class.name}
                    </small>
                  </span>
                  <em>{Math.round(v.telemetry.reserve * 100)}%</em>
                  <button
                    className="vessel-view"
                    aria-label={`View status of ${v.callsign}`}
                    title={`View ${v.callsign} status`}
                    onClick={(event) => {
                      event.stopPropagation();
                      onInspect(v.id);
                    }}
                  >
                    <Eye />
                  </button>
                </div>
              ))}
          </section>
        ))}
      </div>
      <button
        className="wide amber"
        onClick={onCreateMission}
        disabled={selected.size === 0}
      >
        <Route />
        {pirate
          ? `Chart voyage for ${selected.size} mustered`
          : `Create mission from ${selected.size} selected`}
      </button>
      {menu && (
        <VesselGroupContextMenu
          pirate={pirate}
          state={menu}
          groups={fleet.groups}
          onMove={onMove}
          onCreate={onCreateGroupFromVessel}
          onClose={() => setMenu(null)}
        />
      )}
    </div>
  );
}
function SelectionDrawer({
  pirate,
  vessels,
  groups,
  onRemove,
  onInspectVessel,
  onInspectGroup,
  onMove,
  onCreateGroup,
}: {
  pirate: boolean;
  vessels: VesselProfileV2[];
  groups: FleetSnapshotV2["groups"];
  onRemove: (id: string) => void;
  onInspectVessel: (id: string) => void;
  onInspectGroup: (id: string) => void;
  onMove: (vesselID: string, groupID: string) => void;
  onCreateGroup: (vesselID: string, name: string) => void;
}) {
  const [dropGroup, setDropGroup] = useState(""),
    [menu, setMenu] = useState<VesselGroupMenuState | null>(null),
    represented = groups.filter((group) =>
      vessels.some((v) => v.group_id === group.id),
    ).length;
  const drop = (event: React.DragEvent, groupID: string) => {
    event.preventDefault();
    event.stopPropagation();
    setDropGroup("");
    const vesselID = event.dataTransfer.getData(
      "application/x-keelmesh-vessel",
    );
    if (vesselID) onMove(vesselID, groupID);
  };
  return (
    <div className="selection-drawer">
      <div className="drawer-heading">
        <span>
          {pirate
            ? "MUSTER MANIFEST · DRAG INTO ANY CREW"
            : "SELECTED ASSETS · DRAG INTO ANY GROUP"}
        </span>
        <small>
          {vessels.length} {pirate ? "ships" : "vessels"} · {represented}{" "}
          {pirate ? "crews" : "groups"}
        </small>
      </div>
      <div className="selection-scroll">
        {groups.map((group) => {
          const members = vessels.filter((v) => v.group_id === group.id);
          return (
            <section
              className={`selection-group ${members.length === 0 ? "empty" : ""} ${dropGroup === group.id ? "drop-active" : ""}`}
              key={group.id}
              onDragOver={(e) => {
                e.preventDefault();
                e.dataTransfer.dropEffect = "move";
                setDropGroup(group.id);
              }}
              onDragLeave={(e) => {
                if (!e.currentTarget.contains(e.relatedTarget as Node))
                  setDropGroup("");
              }}
              onDrop={(e) => drop(e, group.id)}
              title={`Drop a vessel into ${group.code} ${group.name}`}
            >
              <header style={{ borderLeftColor: group.color }}>
                <span>
                  <b>
                    {group.code} · {group.name}
                  </b>
                  <small>
                    {members.length}/{group.member_ids.length} selected
                  </small>
                </span>
                <button
                  title={`Inspect group ${group.code}`}
                  onClick={() => onInspectGroup(group.id)}
                >
                  <Eye />
                </button>
              </header>
              {members.map((v) => (
                <div
                  className="selected-vessel-row"
                  draggable
                  onDragStart={(e) => {
                    e.dataTransfer.effectAllowed = "move";
                    e.dataTransfer.setData(
                      "application/x-keelmesh-vessel",
                      v.id,
                    );
                  }}
                  onContextMenu={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    setMenu({ vessel: v, x: e.clientX, y: e.clientY });
                  }}
                  key={v.id}
                >
                  <GripVertical />
                  <img src={`/assets/vessels/${v.class.id}.png`} />
                  <span>
                    <b>{v.callsign}</b>
                    <small>
                      {v.designation} · {v.class.name}
                    </small>
                  </span>
                  <em>{Math.round(v.telemetry.reserve * 100)}%</em>
                  <button
                    title={`Inspect ${v.callsign}`}
                    onClick={() => onInspectVessel(v.id)}
                  >
                    <Eye />
                  </button>
                  <button
                    title={`Remove ${v.callsign} from selection`}
                    onClick={() => onRemove(v.id)}
                  >
                    <X />
                  </button>
                </div>
              ))}
            </section>
          );
        })}
      </div>
      {menu && (
        <VesselGroupContextMenu
          pirate={pirate}
          state={menu}
          groups={groups}
          onMove={onMove}
          onCreate={onCreateGroup}
          onClose={() => setMenu(null)}
        />
      )}
    </div>
  );
}
function SelectionInspector({
  vessels,
  groups,
  onInspectVessel,
  onInspectGroup,
}: {
  vessels: VesselProfileV2[];
  groups: FleetSnapshotV2["groups"];
  onInspectVessel: (id: string) => void;
  onInspectGroup: (id: string) => void;
}) {
  const groupIDs = [...new Set(vessels.map((v) => v.group_id))],
    average =
      vessels.reduce((sum, v) => sum + v.telemetry.reserve, 0) / vessels.length,
    minimum = Math.min(...vessels.map((v) => v.telemetry.reserve)),
    missions = new Set(
      vessels.map((v) => v.telemetry.mission_id).filter(Boolean),
    ).size;
  return (
    <div className="selection-inspector">
      <div className="selection-overview">
        <Metric
          k="ASSETS"
          v={String(vessels.length)}
          sub={`${groupIDs.length} operational groups`}
        />
        <Metric
          k="AVG RESERVE"
          v={`${Math.round(average * 100)}%`}
          sub={`minimum ${Math.round(minimum * 100)}%`}
        />
        <Metric k="MISSIONS" v={String(missions)} sub="current assignments" />
      </div>
      {groupIDs.map((groupID) => {
        const group = groups.find((g) => g.id === groupID),
          members = vessels.filter((v) => v.group_id === groupID);
        return (
          <section key={groupID}>
            <header style={{ borderLeftColor: group?.color }}>
              <span>
                <b>
                  {group?.code} · {group?.name}
                </b>
                <small>{members.length} selected</small>
              </span>
              <button onClick={() => onInspectGroup(groupID)}>
                <Eye />
                Inspect group
              </button>
            </header>
            {members.map((v) => (
              <button
                className="inspection-row"
                key={v.id}
                onClick={() => onInspectVessel(v.id)}
              >
                <img src={`/assets/vessels/${v.class.id}.png`} />
                <span>
                  <b>{v.callsign}</b>
                  <small>
                    {v.designation} · {v.telemetry.mode}
                  </small>
                </span>
                <em>{Math.round(v.telemetry.reserve * 100)}%</em>
                <Eye />
              </button>
            ))}
          </section>
        );
      })}
    </div>
  );
}
function GroupManager({
  group,
  vessels,
  onSave,
}: {
  group: FleetSnapshotV2["groups"][number];
  vessels: Map<string, VesselProfileV2>;
  onSave: (v: { name: string; color: string; pattern: string }) => void;
}) {
  const [name, setName] = useState(group.name),
    [color, setColor] = useState(group.color),
    [pattern, setPattern] = useState(group.pattern),
    members = group.member_ids
      .map((id) => vessels.get(id))
      .filter((v): v is VesselProfileV2 => !!v),
    averageReserve =
      members.reduce((sum, v) => sum + v.telemetry.reserve, 0) /
      Math.max(1, members.length),
    minimumReserve = Math.min(...members.map((v) => v.telemetry.reserve), 1),
    moving = members.filter((v) => v.telemetry.speed_mps > 0.1).length,
    attention = members.filter(
      (v) =>
        v.telemetry.health !== "nominal" ||
        v.telemetry.pnt_integrity !== "trusted",
    ).length;
  return (
    <div className="group-manager">
      <div className="group-identity">
        <i style={{ background: color }} />
        <div>
          <small>PRIMARY OPERATIONAL GROUP</small>
          <h2>
            {group.code} · {name}
          </h2>
          <span>
            {group.member_ids.length} exclusive members · revision{" "}
            {group.revision}
          </span>
        </div>
      </div>
      <div className="metric-grid group-status">
        <Metric
          k="AVG RESERVE"
          v={`${Math.round(averageReserve * 100)}%`}
          sub={`minimum ${Math.round(minimumReserve * 100)}%`}
        />
        <Metric
          k="UNDERWAY"
          v={`${moving}/${members.length}`}
          sub="moving vessels"
        />
        <Metric
          k="ATTENTION"
          v={String(attention)}
          sub={attention ? "health or PNT" : "all nominal"}
        />
      </div>
      <label>
        <span className="field-label">
          <Pencil /> GROUP NAME
        </span>
        <input value={name} onChange={(e) => setName(e.target.value)} />
      </label>
      <label>
        IDENTITY COLOR
        <input
          type="color"
          value={color}
          onChange={(e) => setColor(e.target.value)}
        />
      </label>
      <label>
        MAP PATTERN
        <select value={pattern} onChange={(e) => setPattern(e.target.value)}>
          <option>solid</option>
          <option>diagonal</option>
          <option>dots</option>
          <option>crosshatch</option>
          <option>rings</option>
          <option>chevron</option>
        </select>
      </label>
      <div className="group-members">
        {members.map((v) => (
          <span key={v.id}>{v.display_name}</span>
        ))}
      </div>
      <p>
        Membership is exclusive. Drag vessels between group sections in Fleet /
        Groups; active mission membership remains frozen.
      </p>
      <button
        className="wide amber"
        onClick={() => onSave({ name: name.trim(), color, pattern })}
        disabled={!name.trim()}
      >
        Save group identity
      </button>
    </div>
  );
}
function VesselInspector({
  vessel,
  reachability,
  lookup,
  onRename,
}: {
  vessel: VesselProfileV2;
  reachability: ReachabilityV2 | null;
  lookup: Map<string, VesselProfileV2>;
  onRename: (name: string) => void;
}) {
  const t = vessel.telemetry;
  return (
    <div className="vessel-inspector">
      <div className="vessel-hero">
        <img src={`/assets/vessels/${vessel.class.id}.png`} />
        <div>
          <span style={{ color: vessel.group_color }}>
            {vessel.group_code} · {vessel.class.name.toUpperCase()}
          </span>
          <EditableTitle
            value={vessel.callsign}
            label="vessel callsign"
            onSave={onRename}
            maxLength={32}
          />
          <p>
            {vessel.designation} · {vessel.class.role}
          </p>
        </div>
      </div>
      <div className="metric-grid">
        <Metric
          k="RESERVE"
          v={`${Math.round(t.reserve * 100)}%`}
          sub={`projected ${Math.round(t.projected_reserve * 100)}%`}
        />
        <Metric
          k="SPEED"
          v={`${t.speed_mps.toFixed(1)} m/s`}
          sub={`max ${vessel.class.max_speed_mps}`}
        />
        <Metric k="HEADING" v={`${Math.round(t.heading_deg)}°`} sub={t.mode} />
        <Metric
          k="PNT"
          v={t.pnt_integrity}
          sub={`±${t.uncertainty_m.toFixed(0)} m`}
        />
        <Metric
          k="MISSION TAPE"
          v={`${t.tape_depth_seconds}s`}
          sub="validated work"
        />
        <Metric k="HEALTH" v={t.health} sub="all systems" />
      </div>
      <h3>LOCAL CONDITIONS</h3>
      <div className="condition-strip">
        <Metric
          k="WIND"
          v={`${t.environment.wind_speed_mps.toFixed(1)} m/s`}
          sub={`${Math.round(t.environment.wind_direction_deg)}°`}
        />
        <Metric
          k="CURRENT"
          v={`${t.environment.current_speed_mps.toFixed(2)} m/s`}
          sub={`${Math.round(t.environment.current_direction_deg)}°`}
        />
        <Metric
          k="WAVES"
          v={`${t.environment.wave_height_m.toFixed(1)} m`}
          sub="significant"
        />
        <Metric
          k="WATER"
          v={`${t.environment.water_temperature_c.toFixed(1)}°C`}
          sub="fixture"
        />
      </div>
      <small className="fixture">
        {t.environment.label} · {t.environment.source_ids.join(" · ")}
      </small>
      <h3>
        REACHABLE SWARM{" "}
        <span>
          {(reachability?.direct_peers.length ?? 0) +
            (reachability?.relayed_peers.length ?? 0)}
        </span>
      </h3>
      <div className="peer-list">
        {reachability?.direct_peers.map((p) => (
          <Peer key={p.vessel_id} p={p} lookup={lookup} />
        ))}
        {reachability?.relayed_peers.map((p) => (
          <Peer key={p.vessel_id} p={p} lookup={lookup} />
        ))}
      </div>
      <div className="authority-note">
        <b>Reachability ≠ authority</b>
        <span>{reachability?.authority ?? "Loading scoped authority…"}</span>
      </div>
    </div>
  );
}
function Peer({
  p,
  lookup,
}: {
  p: ReachabilityV2["direct_peers"][number];
  lookup: Map<string, VesselProfileV2>;
}) {
  const v = lookup.get(p.vessel_id);
  return (
    <div>
      <i className={p.hops.length > 2 ? "relay" : "direct"} />
      <span>
        <b>{v?.callsign ?? p.vessel_id}</b>
        <small>
          {p.hops.length - 1} hop · {p.underlay.join(" → ")}
        </small>
      </span>
      <em>{p.latency_ms.toFixed(0)} ms</em>
    </div>
  );
}
function Metric({ k, v, sub }: { k: string; v: string; sub: string }) {
  return (
    <div>
      <small>{k}</small>
      <strong>{v}</strong>
      <span>{sub}</span>
    </div>
  );
}
function EditableTitle({
  value,
  label,
  onSave,
  maxLength = 64,
}: {
  value: string;
  label: string;
  onSave: (value: string) => void;
  maxLength?: number;
}) {
  const [editing, setEditing] = useState(false),
    [draft, setDraft] = useState(value);
  useEffect(() => setDraft(value), [value]);
  if (!editing)
    return (
      <div className="editable-title">
        <h2>{value}</h2>
        <button
          aria-label={`Rename ${label}`}
          title={`Rename ${label}`}
          onClick={() => setEditing(true)}
        >
          <Pencil />
        </button>
      </div>
    );
  return (
    <form
      className="editable-title editing"
      onSubmit={(event) => {
        event.preventDefault();
        const next = draft.trim();
        if (next && next !== value) onSave(next);
        setEditing(false);
      }}
    >
      <input
        autoFocus
        aria-label={`New ${label}`}
        value={draft}
        maxLength={maxLength}
        onChange={(event) => setDraft(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Escape") {
            event.preventDefault();
            setDraft(value);
            setEditing(false);
          }
        }}
      />
      <button
        type="submit"
        aria-label={`Save ${label}`}
        disabled={!draft.trim()}
      >
        <Save />
      </button>
      <button
        type="button"
        aria-label={`Cancel ${label} rename`}
        onClick={() => {
          setDraft(value);
          setEditing(false);
        }}
      >
        <X />
      </button>
    </form>
  );
}
function Planner({
  pirate,
  mission,
  draft,
  command,
  setCommand,
  plans,
  activePlan,
  preview,
  lease,
  busy,
  voices,
  voice,
  speechState,
  onVoice,
  onSpeak,
  onFormation,
  onArea,
  onTool,
  onCreate,
  onPlan,
  onPreview,
  onAuthorize,
  onStart,
  onStatus,
  onRename,
  onDelete,
}: {
  pirate: boolean;
  mission: MissionWorkspaceV2 | null;
  draft: CommandDraftV2 | null;
  command: string;
  setCommand: (v: string) => void;
  plans: FleetPlanV2[];
  activePlan: FleetPlanV2 | null;
  preview: FleetPreviewV2 | null;
  lease: FleetLeaseV2 | null;
  busy: boolean;
  voices: VoiceV2[];
  voice: string;
  speechState: string;
  onVoice: (v: string) => void;
  onSpeak: () => void;
  onFormation: (v: string) => void;
  onArea: (k: "include" | "exclude") => void;
  onTool: (t: Tool) => void;
  onCreate: () => void;
  onPlan: (id: string) => void;
  onPreview: () => void;
  onAuthorize: () => void;
  onStart: () => void;
  onStatus: (status: "paused" | "executing") => void;
  onRename: (name: string) => void;
  onDelete: () => void;
}) {
  if (!mission)
    return (
      <div className="window-empty">
        {pirate
          ? "Muster ships and chart a voyage."
          : "Select assets and create a mission."}
      </div>
    );
  return (
    <div className="planner">
      <div className="mission-summary">
        <div className="mission-summary-actions">
          <span>{mission.status}</span>
          <button
            aria-label={`${mission.status === "paused" ? "Resume" : "Pause"} ${mission.name}`}
            title={
              mission.status === "paused" ? "Resume mission" : "Pause mission"
            }
            disabled={
              mission.status !== "executing" && mission.status !== "paused"
            }
            onClick={() =>
              onStatus(mission.status === "paused" ? "executing" : "paused")
            }
          >
            {mission.status === "paused" ? <Play /> : <Pause />}
          </button>
          <button
            className="delete"
            aria-label={`Delete ${mission.name}`}
            title="Delete mission"
            onClick={onDelete}
          >
            <Trash2 />
          </button>
        </div>
        <EditableTitle value={mission.name} label="mission" onSave={onRename} />
        <p>
          {mission.target_ids.length} {pirate ? "sworn hands" : "frozen assets"}{" "}
          · geometry r{mission.geometry.revision} ·{" "}
          {pirate ? "voyage" : "mission"} v{mission.version}
        </p>
      </div>
      <label>
        {pirate ? "CAPTAIN'S ORDERS" : "PLAIN-ENGLISH INTENT"}
        <textarea
          value={command}
          onChange={(e) => setCommand(e.target.value)}
        />
      </label>
      <div className="voice-row">
        <select value={voice} onChange={(e) => onVoice(e.target.value)}>
          {voices.map((v) => (
            <option value={v.id} key={v.id}>
              {v.name}
              {v.default ? " · default" : ""}
            </option>
          ))}
        </select>
        <button onClick={onSpeak}>
          <Volume2 />
          {pirate ? "Sound the orders" : "Read aloud"}
        </button>
        <small>{speechState}</small>
      </div>
      {mission.target_ids.length > 1 ? (
        <label>
          {pirate ? "SAILING FORMATION" : "FORMATION PREFERENCE"}
          <select
            value={mission.formation}
            onChange={(e) => onFormation(e.target.value)}
          >
            {formations.map((f) => (
              <option value={f} key={f}>
                {f.replaceAll("_", " ")}
              </option>
            ))}
          </select>
        </label>
      ) : (
        <div className="solo-mode">
          <Ship />
          <span>
            <b>INDEPENDENT VESSEL</b>
            <small>
              Strategy options replace fleet formations for a single target.
            </small>
          </span>
        </div>
      )}
      <div className="geometry-actions">
        <button onClick={() => onArea("include")}>
          <Plus />
          {pirate ? "Sailing waters" : "Operating area"}
        </button>
        <button onClick={() => onArea("exclude")}>
          <Ban />
          {pirate ? "Forbidden waters" : "Exclusion"}
        </button>
        <button onClick={() => onTool("waypoint")}>
          <MapPinned />
          {pirate ? "Bearing mark" : "Waypoint"}
        </button>
      </div>
      <div className="geometry-summary">
        <span>{mission.geometry.included_areas.length} operating</span>
        <span>{mission.geometry.exclusion_areas.length} excluded</span>
        <span>{mission.geometry.waypoints.length} waypoints</span>
      </div>
      {draft?.geometry_source && (
        <div className="intent-resolution">
          <b>INTENT-DERIVED GEOMETRY</b>
          <code>{draft.geometry_source}</code>
          {draft.resolution_notes?.map((note) => (
            <span key={note}>{note}</span>
          ))}
        </div>
      )}
      {draft?.advisor && (
        <div className={`mission-advisor ${draft.advisor.state}`}>
          <header>
            <Sparkles />
            <b>
              {draft.advisor.provider === "openai"
                ? "OPENAI MISSION ADVISOR"
                : draft.advisor.provider === "openrouter"
                  ? "OPENROUTER MISSION ADVISOR"
                  : draft.advisor.provider === "local"
                    ? "LOCAL MISSION ADVISOR"
                    : "DETERMINISTIC ADVISOR FALLBACK"}
            </b>
            <em>{draft.advisor.model}</em>
          </header>
          <p>{draft.advisor.summary}</p>
          {draft.advisor.geometry_option_id && (
            <code className="advisor-geometry">
              AI GEOMETRY · {draft.advisor.geometry_option_id}
            </code>
          )}
          <div>
            {draft.advisor.attempts.map((attempt, index) => (
              <span
                className={attempt.state}
                key={`${attempt.provider}-${attempt.model}-${index}`}
              >
                {attempt.provider} · {attempt.model} · {attempt.state} ·{" "}
                {attempt.latency_ms} ms
              </span>
            ))}
          </div>
        </div>
      )}
      {plans.length === 0 ? (
        <button className="wide amber" disabled={busy} onClick={onCreate}>
          <Sparkles />
          {pirate
            ? "Ask the ship's AI for courses"
            : "Ask AI for strategy options"}
        </button>
      ) : (
        <div className="candidate-list">
          {plans.map((p) => (
            <button
              key={p.id}
              className={activePlan?.id === p.id ? "selected" : ""}
              onClick={() => onPlan(p.id)}
            >
              <header>
                <b>{p.name}</b>
                <span>
                  {p.advisor_source}
                  {p.advisor_model ? ` · ${p.advisor_model}` : ""}
                </span>
                {p.recommended && (
                  <em>{pirate ? "CAPTAIN'S PICK" : "RECOMMENDED"}</em>
                )}
              </header>
              <p>{p.description}</p>
              <small className="maneuver-sequence">
                {p.maneuvers.join(" → ")}
              </small>
              <dl>
                <span>
                  <small>COVERAGE</small>
                  {p.coverage_percent.toFixed(0)}%
                </span>
                <span>
                  <small>RESERVE</small>
                  {Math.round(p.minimum_reserve * 100)}%
                </span>
                <span>
                  <small>DURATION</small>
                  {p.duration_minutes.toFixed(1)}m
                </span>
                <span>
                  <small>SEPARATION</small>
                  {p.minimum_separation_m}m
                </span>
              </dl>
              <code>{p.content_hash.slice(0, 24)}…</code>
            </button>
          ))}
        </div>
      )}
      {activePlan && !preview && (
        <button className="wide amber" onClick={onPreview}>
          <Eye />
          {pirate ? "Spy the exact course" : "Preview exact routes"}
        </button>
      )}
      {preview && !lease && (
        <>
          <div className="nothing-sent">
            <b>
              {pirate
                ? "No order has left the flagship."
                : "Nothing has been sent yet."}
            </b>
            <span>Preview hash {preview.plan_hash.slice(7, 19)}</span>
            {activePlan?.policy_status === "prohibited" && (
              <span>{activePlan.reason_codes.join(" · ")}</span>
            )}
          </div>
          <button
            className="wide amber"
            disabled={activePlan?.policy_status === "prohibited"}
            onClick={onAuthorize}
          >
            <ShieldCheck />
            {pirate ? "Authorize exact orders" : "Authorize exact plan"}
          </button>
        </>
      )}
      {lease && mission.status !== "executing" && (
        <>
          <div className="nothing-sent ready">
            <b>
              {pirate
                ? "Signed sailing authority ready"
                : "Movement lease ready"}
            </b>
            <span>
              Expires {new Date(lease.expires_at).toLocaleTimeString()}
            </span>
          </div>
          <button className="wide amber" onClick={onStart}>
            <Ship />
            {pirate ? "Make sail under authority" : "Start authorized mission"}
          </button>
        </>
      )}
    </div>
  );
}
function Constraints({
  mission,
  onSave,
}: {
  mission: MissionWorkspaceV2;
  onSave: (v: MissionWorkspaceV2["constraints"]) => void;
}) {
  const [c, setC] = useState(mission.constraints);
  const fields: [keyof typeof c, string, string][] = [
    ["minimum_reserve", "Minimum reserve", "%"],
    ["maximum_speed_mps", "Maximum speed", "m/s"],
    ["minimum_vessel_separation_m", "Vessel separation", "m"],
    ["minimum_object_separation_m", "Object / shore separation", "m"],
    ["maximum_shore_distance_m", "Maximum coastal offset", "m (0 = off)"],
    ["maximum_wave_height_m", "Maximum waves", "m"],
    ["maximum_wind_mps", "Maximum wind", "m/s"],
    ["maximum_pnt_uncertainty_m", "Maximum PNT uncertainty", "m"],
    ["maximum_duration_minutes", "Maximum duration", "min"],
    ["maximum_route_distance_km", "Maximum route distance", "km"],
    ["minimum_tape_watermark_seconds", "Minimum mission tape", "s"],
    ["formation_spacing_m", "Formation spacing", "m"],
    ["regroup_threshold_m", "Regroup threshold", "m"],
  ];
  return (
    <div className="constraints">
      <div className="inheritance">
        <span>HARDWARE</span>
        <i>→</i>
        <span>FLEET</span>
        <i>→</i>
        <span>GROUP</span>
        <i>→</i>
        <span>MISSION</span>
        <i>→</i>
        <span>VESSEL</span>
      </div>
      <p>
        Effective limits merge conservatively. Looser active limits require a
        new exact-diff authorization.
      </p>
      {fields.map(([k, label, unit]) => (
        <label key={k}>
          <span>
            {label}
            <small>{unit}</small>
          </span>
          <input
            type="number"
            step="0.1"
            value={
              k === "minimum_reserve"
                ? Number(c[k] ?? 0) * 100
                : Number(c[k] ?? 0)
            }
            onChange={(e) =>
              setC({
                ...c,
                [k]:
                  k === "minimum_reserve"
                    ? Number(e.target.value) / 100
                    : Number(e.target.value),
              })
            }
          />
        </label>
      ))}
      <button className="wide amber" onClick={() => onSave(c)}>
        Apply safer effective limits
      </button>
    </div>
  );
}
