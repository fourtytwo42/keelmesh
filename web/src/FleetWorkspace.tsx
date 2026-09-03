import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
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
  SurfaceContactV2,
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
  ChevronDown,
  ChevronUp,
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
  Send,
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
  RotateCcw,
  CircleDot,
  Undo2,
} from "lucide-react";

type Tool = "select" | "box" | "waypoint" | "include" | "exclude" | "hold" | "orbit";
type GeometryFocus = { kind: "waypoint" | "include" | "exclude" | "poi"; index: number };
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
const groupPalette = [
  { name: "amber", hex: "#e9a93f" },
  { name: "teal", hex: "#62c5a8" },
  { name: "coral", hex: "#d86f5f" },
  { name: "violet", hex: "#b895d8" },
  { name: "blue", hex: "#7eb4df" },
  { name: "yellow", hex: "#d2c05d" },
  { name: "pink", hex: "#df8fb0" },
  { name: "lime", hex: "#8fca72" },
] as const;
const vesselAsset = (classID: string, pirate: boolean) =>
  `/assets/vessels/${pirate ? "pirate-" : ""}${classID}.png`;

export function FleetWorkspace() {
  const [fleet, setFleet] = useState<FleetSnapshotV2 | null>(null),
    [legacy, setLegacy] = useState<Bootstrap | null>(null),
    [platform, setPlatform] = useState<PlatformSnapshot | null>(null),
    [agent, setAgent] = useState<AgentSnapshot | null>(null),
    [arena, setArena] = useState<ArenaSnapshotV1 | null>(null),
    [voices, setVoices] = useState<VoiceV2[]>([]),
    [voice, setVoice] = useState("jarvis"),
    [speechState, setSpeechState] = useState("ready"),
    [selected, setSelected] = useState<Set<string>>(new Set()),
    [activeMissionID, setActiveMissionID] = useState<string>(""),
    [activeGroupID, setActiveGroupID] = useState<string>(""),
    [tool, setTool] = useState<Tool>("select"),
    [plannerVisible, setPlannerVisible] = useState(false),
    [plannerContactSeed, setPlannerContactSeed] = useState<SurfaceContactV2 | null>(null),
    [geometryFocus, setGeometryFocus] = useState<GeometryFocus | null>(null),
    [search, setSearch] = useState(""),
    [command, setCommand] = useState(""),
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
    [autoRead, setAutoRead] = useState(
      () => localStorage.getItem("keelmesh.auto-read.v2") !== "false",
    ),
    [pendingDeleteID, setPendingDeleteID] = useState("");
  const audio = useRef<HTMLAudioElement | null>(null),
    speechAbort = useRef<AbortController | null>(null),
    recorder = useRef<MediaRecorder | null>(null),
    recordingStream = useRef<MediaStream | null>(null),
    recordingChunks = useRef<BlobPart[]>([]),
    stopRequested = useRef(false),
    geometryHistory = useRef<Record<string, MissionWorkspaceV2["geometry"][]>>({});
  const [inspectVesselID, setInspectVesselID] = useState(""),
    [inspectContactID, setInspectContactID] = useState(""),
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
  useEffect(() => {
    const preferredVoice = pirate ? "barbossa" : "jarvis";
    if (voices.some((candidate) => candidate.id === preferredVoice && candidate.available)) {
      setVoice(preferredVoice);
    }
  }, [pirate, voices]);
  useEffect(() => {
    localStorage.setItem("keelmesh.auto-read.v2", String(autoRead));
  }, [autoRead]);
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
      api<{ voices: VoiceV2[] }>("/api/v2/voices").then((v) => {
        setVoices(v.voices);
        setVoice(v.voices.find((candidate) => candidate.default)?.id ?? "jarvis");
      }),
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
    if (id === "planner") setPlannerVisible(true);
    setWindows((current) => new Set(current).add(id));
    setWindowActivations((current) => ({
      ...current,
      [id]: (current[id] ?? 0) + 1,
    }));
  }, []);
  useEffect(() => {
    setTool("select");
    setGeometryFocus(null);
  }, [activeMissionID]);
  useEffect(() => {
    if (!plannerVisible) setTool("select");
  }, [plannerVisible]);
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
      if (ids.length === 1) {
        window.requestAnimationFrame(() =>
          window.requestAnimationFrame(() =>
            document
              .querySelector(`[data-fleet-vessel="${CSS.escape(ids[0])}"]`)
              ?.scrollIntoView({ block: "nearest", behavior: "smooth" }),
          ),
        );
      }
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
  const inspectedVessel = inspectVesselID
    ? vesselsByID.get(inspectVesselID)
    : undefined;
  const inspectedContact = inspectContactID
    ? fleet?.surface_contacts.find((contact) => contact.id === inspectContactID)
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
    // Selection is convenient but no longer a prerequisite: + opens a blank
    // draft when nothing is selected, and carries an existing selection when present.
    await createMissionFor([...selected]);
  }
  async function createGroupFor(memberIDs: string[], name?: string) {
    if (memberIDs.length === 0) return null;
    const current = await api<FleetSnapshotV2>("/api/v2/fleet");
    const usedColors = new Set(current.groups.map((group) => group.color.toLowerCase()));
    const nextColor =
      groupPalette.find((candidate) => !usedColors.has(candidate.hex))?.hex ??
      groupPalette[current.groups.length % groupPalette.length].hex;
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
          color: nextColor,
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
    values: {
      name?: string;
      color?: string;
      pattern?: string;
      formation?: string;
      formation_spacing_m?: number;
      formation_heading_deg?: number;
      assembly_point?: Point;
      clear_assembly_point?: boolean;
      use_first_member_assembly?: boolean;
    },
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
  async function deleteGroup(id: string) {
    const current = await api<FleetSnapshotV2>("/api/v2/fleet"),
      group = current.groups.find((item) => item.id === id);
    if (!group) return;
    if (
      !window.confirm(
        `Delete ${group.code} ${group.name}? Its ${group.member_ids.length} vessels will remain available under Unassigned.`,
      )
    )
      return;
    await mutate(() =>
      api(`/api/v2/groups/${id}`, {
        method: "DELETE",
        body: JSON.stringify({
          request_id: requestID("group-delete"),
          idempotency_key: requestID("group-delete-key"),
          expected_version: group.revision,
        }),
      }),
    );
    if (activeGroupID === id) {
      setActiveGroupID("");
      setWindows((value) => {
        const next = new Set(value);
        next.delete("group-manager");
        return next;
      });
    }
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
    if ((!group && groupID !== "unassigned") || !vessel || vessel.group_id === groupID) return;
    await mutate(() =>
      api(`/api/v2/groups/${groupID}/members:move`, {
        method: "POST",
        body: JSON.stringify({
          request_id: requestID("group-move"),
          idempotency_key: requestID("group-move-key"),
          expected_version:
            groupID === "unassigned" ? current.fleet_version : group!.revision,
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
      if (updated.missions.length === 0) setPlannerVisible(false);
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
  function selectedOperationalGroup(snapshot = fleet) {
    if (!snapshot || selected.size === 0) return undefined;
    return snapshot.groups.find((group) =>
      group.member_ids.length === selected.size &&
      [...selected].every((id) => group.member_ids.includes(id)),
    );
  }
  function rememberGeometry(value: MissionWorkspaceV2) {
    const stack = geometryHistory.current[value.id] ?? [];
    stack.push(structuredClone(value.geometry));
    geometryHistory.current[value.id] = stack.slice(-20);
  }
  async function saveGeometry(
    target: MissionWorkspaceV2,
    geometry: MissionWorkspaceV2["geometry"],
    remember = true,
  ) {
    const current = (await api<FleetSnapshotV2>("/api/v2/fleet")).missions.find(
      (candidate) => candidate.id === target.id,
    );
    if (!current) return null;
    if (remember) rememberGeometry(current);
    const updated = await mutate(() =>
      api<MissionWorkspaceV2>(`/api/v2/missions/${current.id}/geometry`, {
        method: "POST",
        body: JSON.stringify({
          request_id: requestID("geometry"),
          idempotency_key: requestID("geometry-key"),
          expected_version: current.version,
          included_areas: geometry.included_areas,
          exclusion_areas: geometry.exclusion_areas,
          waypoints: geometry.waypoints,
          waypoint_details: geometry.waypoint_details,
          pois: geometry.pois,
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
    return saveGeometry(current, {
      ...current.geometry,
      waypoints: normalized.map((entry) => entry.position),
      waypoint_details: normalized,
    });
  }
  async function addWaypoint(p: Point, color: WaypointColor) {
    if (!mission || !plannerVisible) return;
    const target = mission;
    const owner = selectedOperationalGroup();
    const entries = waypointEntries(target);
    entries.push({
      id: requestID("waypoint"),
      position: p,
      color: (owner?.color_name as WaypointColor | undefined) ?? color,
      sequence: entries.length + 1,
      owner_group_id: owner?.id,
    });
    await saveWaypoints(target, entries);
  }
  async function moveWaypoint(index: number, position: Point) {
    if (!mission) return;
    const entries = waypointEntries(mission);
    if (!entries[index]) return;
    entries[index] = { ...entries[index], position };
    const updated = await saveWaypoints(mission, entries);
    const ownerID = entries[index].owner_group_id;
    if (!updated || !ownerID) return;
    const group = (await api<FleetSnapshotV2>("/api/v2/fleet")).groups.find(
      (candidate) => candidate.id === ownerID,
    );
    if (group && ["once", "loop", "pause_pending"].includes(group.route_mode))
      await commandGroupRoute(
        ownerID,
        group.route_mode === "loop" ? "enable_loop" : "start_once",
        entries.filter((entry) => entry.owner_group_id === ownerID),
      );
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
  async function addPOI(kind: "hold" | "orbit", position: Point) {
    if (!mission || !plannerVisible) return;
    const next = {
      ...mission.geometry,
      pois: [
        ...mission.geometry.pois,
        { id: requestID(kind), name: kind === "hold" ? "Hold point" : "Orbit point", kind, position, radius_m: kind === "orbit" ? 180 : 50 },
      ],
    };
    await saveGeometry(mission, next);
  }
  async function movePOI(index: number, position: Point) {
    if (!mission || !mission.geometry.pois[index]) return;
    const next = structuredClone(mission.geometry);
    next.pois[index] = { ...next.pois[index], position };
    await saveGeometry(mission, next);
  }
  async function deleteGeometry(focus: GeometryFocus) {
    if (!mission) return;
    if (focus.kind === "waypoint") return deleteWaypoint(focus.index);
    const next = structuredClone(mission.geometry);
    if (focus.kind === "include") next.included_areas.splice(focus.index, 1);
    if (focus.kind === "exclude") next.exclusion_areas.splice(focus.index, 1);
    if (focus.kind === "poi") next.pois.splice(focus.index, 1);
    setGeometryFocus(null);
    await saveGeometry(mission, next);
  }
  async function clearGeometry(kind: "include" | "exclude" | "waypoint" | "poi") {
    if (!mission) return;
    if (kind === "waypoint") return clearWaypoints();
    const next = structuredClone(mission.geometry);
    if (kind === "include") next.included_areas = [];
    if (kind === "exclude") next.exclusion_areas = [];
    if (kind === "poi") next.pois = [];
    setGeometryFocus(null);
    await saveGeometry(mission, next);
  }
  async function reorderWaypoint(index: number, direction: -1 | 1) {
    if (!mission) return;
    const entries = waypointEntries(mission), destination = index + direction;
    if (!entries[index] || destination < 0 || destination >= entries.length) return;
    [entries[index], entries[destination]] = [entries[destination], entries[index]];
    setGeometryFocus({ kind: "waypoint", index: destination });
    await saveWaypoints(mission, entries);
  }
  async function undoGeometry() {
    if (!mission) return;
    const stack = geometryHistory.current[mission.id] ?? [], previous = stack.pop();
    if (!previous) return;
    geometryHistory.current[mission.id] = stack;
    setGeometryFocus(null);
    await saveGeometry(mission, previous, false);
  }
  async function commandGroupRoute(
    groupID: string,
    action: "start_once" | "enable_loop" | "pause_after_leg" | "clear" | "hold_at_vessel",
    explicitWaypoints?: MissionWaypointV2[],
    anchorVesselID?: string,
  ) {
    const snapshot = await api<FleetSnapshotV2>("/api/v2/fleet"),
      group = snapshot.groups.find((candidate) => candidate.id === groupID);
    if (!group) return;
    const sourceMission = snapshot.missions.find((candidate) => candidate.id === activeMissionID) ?? mission;
    const entries = explicitWaypoints ?? (sourceMission ? waypointEntries(sourceMission) : []);
    const route = entries.filter(
      (entry) =>
        entry.owner_group_id === groupID ||
        (!entry.owner_group_id && entry.color === group.color_name),
    );
    await mutate(() =>
      api(`/api/v2/groups/${groupID}/route:command`, {
        method: "POST",
        body: JSON.stringify({
          request_id: requestID("group-route"),
          idempotency_key: requestID("group-route-key"),
          expected_version: group.revision,
          action,
          waypoints: route,
          anchor_vessel_id: anchorVesselID,
        }),
      }),
    );
    await refresh();
  }
  async function cycleGroupRoute(groupID: string) {
    const group = fleet?.groups.find((candidate) => candidate.id === groupID);
    if (!group) return;
    if (group.route_mode === "once")
      return commandGroupRoute(groupID, "enable_loop");
    if (group.route_mode === "loop")
      return commandGroupRoute(groupID, "pause_after_leg");
    if (group.route_mode === "pause_pending") return;
    return commandGroupRoute(groupID, "start_once");
  }
  async function holdGroupAtVessel(groupID: string, vesselID: string) {
    await commandGroupRoute(groupID, "hold_at_vessel", [], vesselID);
  }
  function planSurfaceContact(contactID: string) {
    const contact = fleet?.surface_contacts.find((item) => item.id === contactID);
    if (!contact) return;
    setPlannerContactSeed(contact);
    setTool("select");
    open("planner");
  }
  async function applyContactSeed(createNew: boolean) {
    if (!plannerContactSeed) return;
    let target = createNew ? await createMissionFor([...selected]) : mission;
    if (!target) {
      setError("Select one or more vessels before creating a contact-follow mission.");
      return;
    }
    const intent = `Follow ${plannerContactSeed.name} (${plannerContactSeed.boat_id}) at a safe stand-off distance.`;
    setActiveMissionID(target.id);
    setCommand(intent);
    setPlans([]);
    setDraft(null);
    setPreview(null);
    setLease(null);
    open("planner");
  }
  async function setSimulationRate(rate: FleetSnapshotV2["simulation_rate"]) {
    const current = await api<FleetSnapshotV2>("/api/v2/fleet");
    const updated = await mutate(() =>
      api<FleetSnapshotV2>("/api/v2/simulation/rate", {
        method: "POST",
        body: JSON.stringify({
          request_id: requestID("simulation-rate"),
          idempotency_key: requestID("simulation-rate-key"),
          expected_version: current.fleet_version,
          rate,
        }),
      }),
    ).catch(() => null);
    if (updated) setFleet(updated);
  }
  async function addPolygon(kind: "include" | "exclude", poly: Point[]) {
    if (!mission || !plannerVisible) return;
    await saveGeometry(mission, {
      ...mission.geometry,
      included_areas: kind === "include" ? [...mission.geometry.included_areas, poly] : mission.geometry.included_areas,
      exclusion_areas: kind === "exclude" ? [...mission.geometry.exclusion_areas, poly] : mission.geometry.exclusion_areas,
    });
  }
  async function createPlans(
    targetMission: MissionWorkspaceV2 | null,
    intent = command,
    planningMode: "manual" | "ai_assisted" = "ai_assisted",
    guidanceKind = "",
    followContactID = "",
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
            planning_mode: planningMode,
            guidance_kind: guidanceKind,
            follow_contact_id: followContactID,
          }),
        },
      ),
    ).catch(() => null);
    if (!compiled) return;
    setDraft(compiled);
    setCommand("");
    if (planningMode === "ai_assisted" && autoRead && compiled.advisor?.summary)
      void speak(compiled.advisor.summary);
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
        body: JSON.stringify({
          request_id: requestID("speech"),
          voice,
          text: speechText(text),
        }),
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
    const targetMission = mission;
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
          if (targetMission) await createPlans(targetMission, result.text);
        } catch (e) {
          setSpeechState("typed fallback");
          setError((e as Error).message);
        }
      };
      active.start(200);
      setSpeechState("listening · release to send");
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
      preferredDock: "left",
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
          onDeleteGroup={deleteGroup}
          mission={mission}
          onCycleRoute={cycleGroupRoute}
          onHoldGroupAtVessel={holdGroupAtVessel}
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
          onDelete={() => deleteGroup(activeGroup.id)}
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
          pirate={pirate}
          vessel={inspectedVessel}
          reachability={reachability}
          lookup={vesselsByID}
          onRename={(name) => renameVessel(inspectedVessel.id, name)}
        />
      ),
    });
  if (windows.has("contact-inspector") && inspectedContact)
    defs.push({
      id: "contact-inspector",
      kind: "context",
      activation: windowActivations["contact-inspector"],
      title: inspectedContact.name,
      icon: <Ship />,
      initial: { x: 370, y: 105, width: 360, height: 540 },
      content: <SurfaceContactInspector contact={inspectedContact} />,
    });
  if (windows.has("planner"))
    defs.push({
      id: "planner",
      kind: "primary",
      maximizable: true,
      preferredDock: "right",
      onVisibilityChange: (visible) => setPlannerVisible(visible),
      activation: windowActivations.planner,
      title: pirate ? "Voyage Plotter" : "Mission Planner",
      icon: <Route />,
      initial: { x: window.innerWidth - 440, y: 92, width: 420, height: 650 },
      content: (
        <Planner
          pirate={pirate}
          mission={mission}
          groups={fleet.groups}
          selectedIDs={[...selected]}
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
          autoRead={autoRead}
          recording={
            speechState === "requesting microphone" ||
            speechState.startsWith("listening")
          }
          tool={tool}
          contactSeed={plannerContactSeed}
          geometryFocus={geometryFocus}
          onVoice={setVoice}
          onSpeak={(text) => void speak(text)}
          onAutoRead={setAutoRead}
          onTranscriptionStart={() => void beginTranscription()}
          onTranscriptionStop={endTranscription}
          onFormation={(f) => updateMission({ formation: f })}
          onObjective={(objective) => updateMission({ objective })}
          onAssignGroup={(groupID) => {
            const group = fleet.groups.find((candidate) => candidate.id === groupID);
            if (group) {
              setPlans([]);
              setDraft(null);
              setPreview(null);
              setLease(null);
              void updateMission({ target_ids: group.member_ids });
            }
          }}
          onAssignAssets={(targetIDs) => {
            setPlans([]);
            setDraft(null);
            setPreview(null);
            setLease(null);
            void updateMission({ target_ids: targetIDs });
          }}
          onOpenFleet={() => open("fleet")}
          onArea={(kind) => setTool(kind)}
          onTool={setTool}
          onCreate={(intent) => createPlans(mission, intent, "ai_assisted")}
          onGenerateManual={(intent, guidance, followContactID) =>
            createPlans(mission, intent, "manual", guidance, followContactID)
          }
          onApplyContactSeed={applyContactSeed}
          onClearContactSeed={() => setPlannerContactSeed(null)}
          onUndoGeometry={() => void undoGeometry()}
          onClearGeometry={(kind) => void clearGeometry(kind)}
          onDeleteGeometry={(focus) => void deleteGeometry(focus)}
          onFocusGeometry={setGeometryFocus}
          onReorderWaypoint={(index, direction) => void reorderWaypoint(index, direction)}
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
          <span>{fleet.surface_contacts.length} CONTACTS</span>
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
                setPlannerContactSeed(null);
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
        editingEnabled={plannerVisible && !!mission}
        focusedGeometry={geometryFocus}
        onSelect={(ids, mode) => {
          select(ids, mode);
          if (plannerVisible && tool === "box" && mission && mode === "replace" && ids.length > 0)
            void updateMission({ target_ids: ids });
        }}
        onGroup={selectGroup}
        onVessel={(id) => {
          setInspectVesselID(id);
          open("inspector");
        }}
        onOpenFleet={revealFleet}
        onContact={(id) => {
          setInspectContactID(id);
          open("contact-inspector");
        }}
        onPlanContact={planSurfaceContact}
        onGeometryFocus={(kind, index) => {
          setGeometryFocus({ kind, index });
          open("planner");
        }}
        onWaypoint={addWaypoint}
        onPOI={addPOI}
        onMoveWaypoint={moveWaypoint}
        onMovePOI={movePOI}
        onMoveGroupAssembly={(groupID, point) =>
          void patchGroup(groupID, { assembly_point: point })
        }
        onHoldGroupAtVessel={(groupID, vesselID) =>
          void holdGroupAtVessel(groupID, vesselID)
        }
        onArea={addPolygon}
        onToolDone={() => setTool("select")}
      />
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
        <div className="sim-speed" role="group" aria-label="Simulation speed">
          <small>SIM</small>
          {([0, 1, 5, 20, 100, 500] as const).map((rate) => (
            <button
              key={rate}
              type="button"
              className={fleet.simulation_rate === rate ? "active" : ""}
              aria-label={rate === 0 ? "Pause simulation" : `Run simulation at ${rate} times speed`}
              aria-pressed={fleet.simulation_rate === rate}
              title={rate === 0 ? "Pause simulation" : `${rate}× simulation speed`}
              onClick={() => void setSimulationRate(rate)}
            >
              {rate === 0 ? <Pause /> : `${rate}×`}
            </button>
          ))}
        </div>
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
  onHold,
  onClose,
}: {
  pirate: boolean;
  state: VesselGroupMenuState;
  groups: FleetSnapshotV2["groups"];
  onMove: (vesselID: string, groupID: string) => void;
  onCreate: (vesselID: string, name: string) => void;
  onHold?: (groupID: string, vesselID: string) => void;
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
          {state.vessel.group_id && onHold && (
            <>
              <button
                role="menuitem"
                onClick={() => {
                  onHold(state.vessel.group_id, state.vessel.id);
                  onClose();
                }}
              >
                <MapPinned />
                <span>{pirate ? "Hold crew on this ship" : "Hold group at this vessel"}</span>
              </button>
            </>
          )}
          <button
            role="menuitem"
            disabled={!state.vessel.group_id}
            onClick={() => {
              onMove(state.vessel.id, "unassigned");
              onClose();
            }}
          >
            <i style={{ background: "#737973" }} />
            <span>
              <b>—</b>
              {pirate ? "Without crew" : "Unassigned"}
            </span>
            <small>{!state.vessel.group_id ? "current" : ""}</small>
          </button>
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
                {group.name} · {group.color_name}
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
  onDeleteGroup,
  mission,
  onCycleRoute,
  onHoldGroupAtVessel,
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
  onDeleteGroup: (groupID: string) => void;
  mission: MissionWorkspaceV2 | null;
  onCycleRoute: (groupID: string) => void;
  onHoldGroupAtVessel: (groupID: string, vesselID: string) => void;
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
        {fleet.groups.map((g) => {
          const routeWaypointCount = (mission?.geometry.waypoint_details ?? []).filter(
            (waypoint) =>
              waypoint.owner_group_id === g.id ||
              (!waypoint.owner_group_id && waypoint.color === g.color_name),
          ).length;
          const routeTitle =
            g.route_mode === "once"
              ? "Enable waypoint loop"
              : g.route_mode === "loop"
                ? "Pause after current waypoint"
                : g.route_mode === "pause_pending"
                  ? "Pause pending at current waypoint"
                  : "Run waypoints once";
          return (
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
                <em>{g.color_name}</em>
                <small>
                  {g.member_ids.filter((id) => selected.has(id)).length}/
                  {g.member_ids.length}
                </small>
              </button>
              <button
                className={`group-route ${g.route_mode}`}
                aria-label={`${routeTitle} for ${g.code} ${g.name}`}
                title={routeTitle}
                disabled={routeWaypointCount < 2 || g.route_mode === "pause_pending"}
                onClick={() => onCycleRoute(g.id)}
              >
                {g.route_mode === "pause_pending" ? <Pause /> : <Play />}
                {g.route_mode === "loop" && <span>∞</span>}
              </button>
              <button
                className="group-view"
                aria-label={`View status of ${g.code} ${g.name}`}
                title={`View ${g.code} group status`}
                onClick={() => onManage(g.id)}
              >
                <Eye />
              </button>
              <button
                className="group-delete"
                aria-label={`Delete ${g.code} ${g.name}`}
                title={`Delete group and leave its vessels unassigned`}
                onClick={() => onDeleteGroup(g.id)}
              >
                <Trash2 />
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
                  data-fleet-vessel={v.id}
                  key={v.id}
                >
                  <input
                    aria-label={`Select ${v.callsign}`}
                    type="checkbox"
                    checked={selected.has(v.id)}
                    onChange={() => onSelect([v.id], "toggle")}
                  />
                  <i style={{ borderColor: v.group_color }}>
                    <img src={vesselAsset(v.class.id, pirate)} />
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
          );
        })}
        {filtered.some((v) => !v.group_id) && (
          <section
            className={`unassigned-group ${dropGroup === "unassigned" ? "drop-active" : ""}`}
            onDragOver={(e) => {
              e.preventDefault();
              e.dataTransfer.dropEffect = "move";
              setDropGroup("unassigned");
            }}
            onDragLeave={(e) => {
              if (!e.currentTarget.contains(e.relatedTarget as Node))
                setDropGroup("");
            }}
            onDrop={(e) => drop(e, "unassigned")}
          >
            <div className="group-row-wrap unassigned">
              <div className="group-row">
                <i />
                <b>—</b>
                <span>{pirate ? "Without crew" : "Unassigned"}</span>
                <small>
                  {filtered.filter((v) => !v.group_id && selected.has(v.id)).length}/
                  {fleet.vessels.filter((v) => !v.group_id).length}
                </small>
              </div>
            </div>
            {filtered
              .filter((v) => !v.group_id)
              .map((v) => (
                <div
                  draggable
                  onDragStart={(e) => {
                    e.dataTransfer.effectAllowed = "move";
                    e.dataTransfer.setData("application/x-keelmesh-vessel", v.id);
                  }}
                  onContextMenu={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    setMenu({ vessel: v, x: e.clientX, y: e.clientY });
                  }}
                  className={`fleet-vessel-row ${selected.has(v.id) ? "selected" : ""}`}
                  data-fleet-vessel={v.id}
                  key={v.id}
                >
                  <input
                    aria-label={`Select ${v.callsign}`}
                    type="checkbox"
                    checked={selected.has(v.id)}
                    onChange={() => onSelect([v.id], "toggle")}
                  />
                  <i style={{ borderColor: "#737973" }}>
                    <img src={vesselAsset(v.class.id, pirate)} />
                  </i>
                  <span>
                    <b>{v.callsign}</b>
                    <small>{v.designation} · {v.class.name}</small>
                  </span>
                  <em>{Math.round(v.telemetry.reserve * 100)}%</em>
                  <button
                    className="vessel-view"
                    aria-label={`View status of ${v.callsign}`}
                    onClick={() => onInspect(v.id)}
                  >
                    <Eye />
                  </button>
                </div>
              ))}
          </section>
        )}
      </div>
      {menu && (
        <VesselGroupContextMenu
          pirate={pirate}
          state={menu}
          groups={fleet.groups}
          onMove={onMove}
          onCreate={onCreateGroupFromVessel}
          onHold={onHoldGroupAtVessel}
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
                    {group.color_name} team · {members.length}/{group.member_ids.length} selected
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
                  <img src={vesselAsset(v.class.id, pirate)} />
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
  pirate,
  vessels,
  groups,
  onInspectVessel,
  onInspectGroup,
}: {
  pirate: boolean;
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
                <img src={vesselAsset(v.class.id, pirate)} />
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
  onDelete,
}: {
  group: FleetSnapshotV2["groups"][number];
  vessels: Map<string, VesselProfileV2>;
  onSave: (v: {
    name?: string;
    color?: string;
    pattern?: string;
    formation?: string;
    formation_spacing_m?: number;
    formation_heading_deg?: number;
    clear_assembly_point?: boolean;
    use_first_member_assembly?: boolean;
  }) => void;
  onDelete: () => void;
}) {
  const [name, setName] = useState(group.name),
    [color, setColor] = useState(group.color),
    [pattern, setPattern] = useState(group.pattern),
    [formation, setFormation] = useState(group.formation || "column"),
    [spacing, setSpacing] = useState(group.formation_spacing_m || 60),
    [heading, setHeading] = useState(group.formation_heading_deg || 0),
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
    ).length,
    decisionNode = members.find((v) => v.id === group.decision_node_id);
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
            {group.color_name} team · {group.member_ids.length} exclusive members · revision{" "}
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
        <Metric
          k="DECISION NODE"
          v={decisionNode?.callsign || "None reachable"}
          sub={`epoch ${group.decision_epoch} · ${group.decision_policy.replaceAll("_", " ")}`}
        />
      </div>
      <p className="group-decision-note">
        This node coordinates bounded group adaptations. Decisions outside the
        signed mission guardrails stop safely and request operator instruction.
      </p>
      <label>
        <span className="field-label">
          <Pencil /> GROUP NAME
        </span>
        <input value={name} onChange={(e) => setName(e.target.value)} />
      </label>
      <label>
        IDENTITY COLOR
        <select value={color} onChange={(e) => setColor(e.target.value)}>
          {!groupPalette.some((candidate) => candidate.hex === color) && (
            <option value={color}>{group.color_name}</option>
          )}
          {groupPalette.map((candidate) => (
            <option value={candidate.hex} key={candidate.name}>
              {candidate.name[0].toUpperCase() + candidate.name.slice(1)}
            </option>
          ))}
        </select>
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
      <label>
        IDLE FORMATION
        <select value={formation} onChange={(e) => setFormation(e.target.value)}>
          {formations.map((value) => (
            <option key={value} value={value}>
              {value.replaceAll("_", " ")}
            </option>
          ))}
        </select>
      </label>
      <label>
        FORMATION SPACING
        <span className="number-field">
          <input
            type="number"
            min={15}
            max={1000}
            step={5}
            value={spacing}
            onChange={(e) => setSpacing(Number(e.target.value))}
          />
          <small>metres</small>
        </span>
      </label>
      <label>
        FORMATION HEADING
        <span className="number-field">
          <input
            type="number"
            min={0}
            max={359}
            step={5}
            value={heading}
            onChange={(e) => setHeading(Number(e.target.value))}
          />
          <small>° true</small>
        </span>
      </label>
      <div className="assembly-control">
        <header>
          <MapPinned />
          <span>
            <b>ASSEMBLY POINT</b>
            <small>
              {group.assembly_point
                ? `${group.assembly_point[1].toFixed(4)}°N · ${Math.abs(group.assembly_point[0]).toFixed(4)}°W`
                : "Not assigned"}
            </small>
          </span>
        </header>
        <p>
          The group station-keeps around this point using the selected formation
          and spacing when it has no active movement mission.
        </p>
        <div>
          <button
            onClick={() => onSave({ use_first_member_assembly: true })}
            disabled={members.length === 0}
          >
            Use first member
          </button>
          <button
            className="delete"
            onClick={() => onSave({ clear_assembly_point: true })}
            disabled={!group.assembly_point}
          >
            <Trash2 /> Clear point
          </button>
        </div>
      </div>
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
        onClick={() =>
          onSave({
            name: name.trim(),
            color,
            pattern,
            formation,
            formation_spacing_m: spacing,
            formation_heading_deg: heading,
          })
        }
        disabled={!name.trim() || spacing < 15 || spacing > 1000 || heading < 0 || heading >= 360}
      >
        Save group station policy
      </button>
      <button className="wide danger group-delete-wide" onClick={onDelete}>
        <Trash2 /> Delete group · keep vessels unassigned
      </button>
    </div>
  );
}
function VesselInspector({
  pirate,
  vessel,
  reachability,
  lookup,
  onRename,
}: {
  pirate: boolean;
  vessel: VesselProfileV2;
  reachability: ReachabilityV2 | null;
  lookup: Map<string, VesselProfileV2>;
  onRename: (name: string) => void;
}) {
  const t = vessel.telemetry;
  return (
    <div className="vessel-inspector">
      <div className="vessel-hero">
        <img src={vesselAsset(vessel.class.id, pirate)} />
        <div>
          <span style={{ color: vessel.group_color }}>
            {vessel.group_code} · {vessel.group_color_name.toUpperCase()} TEAM · {vessel.class.name.toUpperCase()}
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

function SurfaceContactInspector({ contact }: { contact: SurfaceContactV2 }) {
  return (
    <div className="surface-contact-inspector">
      <div className="surface-contact-hero">
        <img src={`/assets/traffic/${contact.class}.png`} alt="" />
        <div>
          <span style={{ color: contact.color }}>
            {contact.color_name.toUpperCase()} CONTACT · {contact.class.toUpperCase()}
          </span>
          <h2>{contact.name}</h2>
          <p>{contact.boat_id} · {contact.callsign}</p>
        </div>
      </div>
      <div className="contact-simulation-note">
        FICTIONAL SURFACE TRAFFIC · SIMULATED AIS-LIKE TRACK
      </div>
      <div className="metric-grid">
        <Metric
          k="POSITION"
          v={`${contact.position[1].toFixed(4)}° N`}
          sub={`${Math.abs(contact.position[0]).toFixed(4)}° W`}
        />
        <Metric
          k="COURSE"
          v={`${Math.round(contact.heading_deg)}°`}
          sub={contact.navigation_state}
        />
        <Metric
          k="SPEED"
          v={`${contact.speed_knots.toFixed(1)} kn`}
          sub={`${contact.speed_mps.toFixed(1)} m/s`}
        />
        <Metric
          k="DIMENSIONS"
          v={`${contact.length_m.toFixed(0)} m`}
          sub={`${contact.draft_m.toFixed(1)} m draft`}
        />
      </div>
      <h3>CURRENT ACTIVITY</h3>
      <p className="contact-activity">{contact.activity}</p>
      <h3>PROGRAMMED TRACK</h3>
      <div className="contact-route">
        <i style={{ background: contact.color }} />
        <span>
          <b>{contact.route_name}</b>
          <small>
            {contact.route.length} route points · {contact.looping ? "continuous loop" : "one way"}
          </small>
        </span>
      </div>
      <div className="authority-note">
        <b>Observable, not commandable</b>
        <span>
          Tell KeelMesh AI “follow {contact.name}” or “follow {contact.boat_id}”
          to create policy-checked intercept and trail options.
        </span>
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

function speechText(markdown: string) {
  return markdown
    .replace(/!\[[^\]]*\]\([^)]*\)/g, "")
    .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
    .replace(/[`*_>#~-]/g, " ")
    .replace(/\s+/g, " ")
    .trim();
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
  groups,
  selectedIDs,
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
  autoRead,
  recording,
  tool,
  contactSeed,
  geometryFocus,
  onVoice,
  onSpeak,
  onAutoRead,
  onTranscriptionStart,
  onTranscriptionStop,
  onFormation,
  onObjective,
  onAssignGroup,
  onAssignAssets,
  onOpenFleet,
  onArea,
  onTool,
  onCreate,
  onGenerateManual,
  onApplyContactSeed,
  onClearContactSeed,
  onUndoGeometry,
  onClearGeometry,
  onDeleteGeometry,
  onFocusGeometry,
  onReorderWaypoint,
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
  groups: FleetSnapshotV2["groups"];
  selectedIDs: string[];
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
  autoRead: boolean;
  recording: boolean;
  tool: Tool;
  contactSeed: SurfaceContactV2 | null;
  geometryFocus: GeometryFocus | null;
  onVoice: (v: string) => void;
  onSpeak: (text: string) => void;
  onAutoRead: (enabled: boolean) => void;
  onTranscriptionStart: () => void;
  onTranscriptionStop: () => void;
  onFormation: (v: string) => void;
  onObjective: (v: string) => void;
  onAssignGroup: (groupID: string) => void;
  onAssignAssets: (targetIDs: string[]) => void;
  onOpenFleet: () => void;
  onArea: (k: "include" | "exclude") => void;
  onTool: (t: Tool) => void;
  onCreate: (intent: string) => void;
  onGenerateManual: (intent: string, guidance: string, followContactID: string) => void;
  onApplyContactSeed: (createNew: boolean) => void;
  onClearContactSeed: () => void;
  onUndoGeometry: () => void;
  onClearGeometry: (kind: "include" | "exclude" | "waypoint" | "poi") => void;
  onDeleteGeometry: (focus: GeometryFocus) => void;
  onFocusGeometry: (focus: GeometryFocus) => void;
  onReorderWaypoint: (index: number, direction: -1 | 1) => void;
  onPlan: (id: string) => void;
  onPreview: () => void;
  onAuthorize: () => void;
  onStart: () => void;
  onStatus: (status: "paused" | "executing") => void;
  onRename: (name: string) => void;
  onDelete: () => void;
}) {
  const chatEnd = useRef<HTMLDivElement | null>(null);
  const [expandedPlans, setExpandedPlans] = useState<Set<string>>(new Set());
  const [missionType, setMissionType] = useState("patrol");
  const [manualObjective, setManualObjective] = useState(mission?.objective ?? "");
  useEffect(() => {
    if (contactSeed) setMissionType("follow_contact");
  }, [contactSeed]);
  useEffect(() => setManualObjective(mission?.objective ?? ""), [mission?.id, mission?.objective]);
  useEffect(() => {
    chatEnd.current?.scrollIntoView({ block: "end", behavior: "smooth" });
  }, [mission?.conversation?.length, busy]);
  if (!mission)
    return (
      <div className="window-empty planner-seed-empty">
        {contactSeed ? <>
          <Ship />
          <b>{contactSeed.name}</b>
          <span>{contactSeed.boat_id} · {contactSeed.class} · uncommitted planning context</span>
          <button className="wide amber" onClick={() => onApplyContactSeed(true)} disabled={busy}>
            <Plus /> {pirate ? "Create shadowing voyage" : "Create follow mission"}
          </button>
          <button className="wide" onClick={onClearContactSeed}>Cancel</button>
        </> : pirate
          ? "Muster ships and chart a voyage."
          : "Select assets and create a mission."}
      </div>
    );
  const manualIntent = manualObjective.trim() || `${missionType.replaceAll("_", " ")} mission`,
    followContactID = missionType === "follow_contact" ? contactSeed?.id ?? draft?.follow_contact_id ?? "" : "",
    hasRouteGeometry = mission.geometry.waypoints.length > 0 || mission.geometry.included_areas.length > 0,
    hasPointForBehavior = mission.geometry.pois.some((poi) => poi.kind === missionType),
    manualReady = mission.target_ids.length > 0 && (missionType === "follow_contact" ? !!followContactID : missionType === "hold" || missionType === "orbit" ? hasPointForBehavior : hasRouteGeometry);
  return (
    <div className="planner">
      <div className="planner-layout">
      <section className="planner-chat-pane">
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
      <div className="mission-chat" aria-live="polite">
        {(mission.conversation ?? []).length === 0 && (
          <div className="chat-empty">
            <Bot />
            <b>{pirate ? "Ask the ship's intelligence" : "Plan with KeelMesh AI"}</b>
            <span>
              Describe the outcome, ask about constraints, or request alternatives.
              The agent may annotate the map and propose plans, but exact authority
              remains yours.
            </span>
          </div>
        )}
        {(mission.conversation ?? []).map((message) => (
          <article className={`chat-message ${message.role}`} key={message.id}>
            <header>
              {message.role === "operator" ? <strong>YOU</strong> : <><Sparkles /><strong>KEELMESH AI</strong><button className="chat-replay" aria-label="Read this AI reply aloud" title="Read aloud" onClick={() => onSpeak(message.markdown)}><Volume2 /></button></>}
              <time>{new Date(message.created_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}</time>
            </header>
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{message.markdown}</ReactMarkdown>
          </article>
        ))}
        {busy && (
          <div className="agent-work-chips">
            <span><Sparkles /> Interpreting intent</span>
            <span>Inspecting map context</span>
            <span>Validating bounded options</span>
          </div>
        )}
        {activePlan && <PlanMiniMap plan={activePlan} compact />}
        <div ref={chatEnd} />
      </div>
      <div className="chat-composer">
        <textarea
          aria-label={pirate ? "Captain's orders" : "Message mission AI"}
          placeholder={pirate ? "Ask about this voyage…" : "Ask about this mission or describe what should happen…"}
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              if (command.trim() && !busy) onCreate(command.trim());
            }
          }}
        />
        <div className="chat-composer-actions">
          <button
            className={`chat-mic ${recording ? "recording" : ""}`}
            aria-label={recording ? "Release to send voice message" : "Hold to talk"}
            aria-pressed={recording}
            title={recording ? "Release to transcribe and send" : "Hold to talk"}
            disabled={busy}
            onPointerDown={(event) => {
              if (event.button !== 0) return;
              event.preventDefault();
              event.currentTarget.setPointerCapture(event.pointerId);
              onTranscriptionStart();
            }}
            onPointerUp={(event) => {
              event.preventDefault();
              if (event.currentTarget.hasPointerCapture(event.pointerId))
                event.currentTarget.releasePointerCapture(event.pointerId);
              onTranscriptionStop();
            }}
            onPointerCancel={onTranscriptionStop}
            onKeyDown={(event) => {
              if ((event.key === " " || event.key === "Enter") && !event.repeat) {
                event.preventDefault();
                onTranscriptionStart();
              }
            }}
            onKeyUp={(event) => {
              if (event.key === " " || event.key === "Enter") {
                event.preventDefault();
                onTranscriptionStop();
              }
            }}
          >
            <Mic />
          </button>
          <button
            className="chat-send"
            aria-label="Send to mission AI"
            disabled={!command.trim() || busy}
            onClick={() => onCreate(command.trim())}
          >
            <Send />
          </button>
        </div>
      </div>
      <div className="voice-row">
        <select aria-label="AI voice" value={voice} onChange={(e) => onVoice(e.target.value)}>
          {voices.map((v) => (
            <option value={v.id} key={v.id}>
              {v.name}
              {v.default ? " · default" : ""}
            </option>
          ))}
        </select>
        <label className="auto-read-toggle">
          <input
            type="checkbox"
            checked={autoRead}
            onChange={(event) => onAutoRead(event.target.checked)}
          />
          <Volume2 />
          {pirate ? "Read replies aloud" : "Read AI replies aloud"}
        </label>
        <small>{speechState}</small>
      </div>
      </section>
      <section className="planner-options-pane">
      <section className="planner-assets">
      <label>
        {pirate ? "ASSIGNED CREW" : "ASSIGNED ASSETS"}
        <select
          value={
            groups.find(
              (group) =>
                group.member_ids.length === mission.target_ids.length &&
                group.member_ids.every((id) => mission.target_ids.includes(id)),
            )?.id ?? ""
          }
          onChange={(event) => onAssignGroup(event.target.value)}
        >
          <option value="" disabled>
            {mission.target_ids.length === 0
              ? pirate ? "No ships assigned yet" : "No assets assigned yet"
              : `${mission.target_ids.length} mixed or individual assets`}
          </option>
          {groups.map((group) => (
            <option value={group.id} key={group.id}>
              {group.code} · {group.name} · {group.color_name}
            </option>
          ))}
        </select>
      </label>
      <div className="planner-asset-actions">
        <span>
          <Users />
          {mission.target_ids.length} {pirate ? "aboard" : "assigned"}
        </span>
        <button type="button" onClick={onOpenFleet}>
          <Ship /> {pirate ? "Choose ships" : "Open Fleet"}
        </button>
        <button
          type="button"
          disabled={selectedIDs.length === 0 || busy}
          onClick={() => onAssignAssets(selectedIDs)}
        >
          <CheckCircle2 />
          {pirate
            ? `Muster selected (${selectedIDs.length})`
            : `Use fleet selection (${selectedIDs.length})`}
        </button>
        {mission.target_ids.length > 0 && (
          <button type="button" disabled={busy} onClick={() => onAssignAssets([])}>
            <X /> {pirate ? "Dismiss crew" : "Clear assignment"}
          </button>
        )}
      </div>
      {mission.target_ids.length === 0 && (
        <p className="planner-assets-hint">
          {pirate
            ? "Choose a crew, use the current fleet selection, or tell the ship's intelligence which crew you intend to command."
            : "Choose an operational group or select any mix in Fleet / Groups, then apply that selection here. The planner and AI chat stay available while you build the mission."}
        </p>
      )}
      </section>
      {mission.trajectory && (
        <div className="trajectory-program-summary">
          <header>
            <Route />
            <b>TRAJECTORY PROGRAM · REVISION {mission.trajectory.active_revision}</b>
            {mission.trajectory.pending_revision && (
              <em>R{mission.trajectory.pending_revision} ARMED · T+{mission.trajectory.activation_tick}</em>
            )}
          </header>
          <dl>
            <span><small>FULL PROGRAM</small>{Math.ceil(mission.trajectory.duration_seconds / 60)} min</span>
            <span><small>SEGMENTS</small>{mission.trajectory.total_segments}</span>
            <span><small>HOT TAPE</small>{mission.trajectory.hot_tape_horizon_seconds}s rolling</span>
            <span><small>CURSOR</small>T+{mission.trajectory.mission_tick}s</span>
          </dl>
          <p>
            The complete signed mission is retained; each node materializes the
            next bounded execution window and may adjust only inside its active envelope.
          </p>
        </div>
      )}
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
      ) : mission.target_ids.length === 1 ? (
        <div className="solo-mode">
          <Ship />
          <span>
            <b>INDEPENDENT VESSEL</b>
            <small>
              Strategy options replace fleet formations for a single target.
            </small>
          </span>
        </div>
      ) : (
        <div className="solo-mode pending">
          <Users />
          <span>
            <b>{pirate ? "CREW NOT YET MUSTERED" : "ASSET ASSIGNMENT PENDING"}</b>
            <small>
              {pirate
                ? "Choose a crew or muster selected ships before plotting routes."
                : "Choose a group or apply the current Fleet selection before generating routes."}
            </small>
          </span>
        </div>
      )}
      {contactSeed && (
        <div className="planner-contact-seed">
          <i style={{ background: contactSeed.color }} />
          <span><b>{contactSeed.name}</b><small>{contactSeed.boat_id} · uncommitted objective</small></span>
          <button onClick={() => onApplyContactSeed(false)}>Use in this mission</button>
          <button onClick={() => onApplyContactSeed(true)}>Create new mission</button>
          <button aria-label="Dismiss contact planning context" onClick={onClearContactSeed}><X /></button>
        </div>
      )}
      <details className="planner-section" open>
        <summary>OBJECTIVE &amp; MISSION TYPE</summary>
        <label>
          MISSION TYPE
          <select value={missionType} onChange={(event) => setMissionType(event.target.value)}>
            <option value="transit">Transit</option>
            <option value="patrol">Patrol</option>
            <option value="search">Search</option>
            <option value="follow_contact">Follow contact</option>
            <option value="hold">Hold</option>
            <option value="orbit">Orbit</option>
            <option value="custom_route">Custom route</option>
          </select>
        </label>
        <label>
          OBJECTIVE
          <textarea value={manualObjective} onChange={(event) => setManualObjective(event.target.value)} onBlur={() => { if (manualObjective.trim() && manualObjective.trim() !== mission.objective) onObjective(manualObjective.trim()); }} />
        </label>
      </details>
      <details className="planner-section map-authoring" open>
        <summary>MAP AUTHORING</summary>
        <div className="authoring-status">
          <b>{mission.name.toUpperCase()}</b>
          <span>{tool === "select" ? "READY · SELECT OR EDIT" : `${tool.replaceAll("_", " ").toUpperCase()} TOOL ACTIVE · ESC TO CANCEL`}</span>
        </div>
        <div className="geometry-actions">
          <button className={tool === "select" ? "active" : ""} onClick={() => onTool("select")} title="Select or drag existing geometry"><MousePointer2 />Edit</button>
          <button className={tool === "box" ? "active" : ""} onClick={() => onTool("box")} title="Assign vessels inside a rectangle"><BoxSelect />Assign box</button>
          <button className={tool === "include" ? "active" : ""} onClick={() => onArea("include")}><Plus /><BoxSelect />Operating</button>
          <button className={tool === "exclude" ? "active" : ""} onClick={() => onArea("exclude")}><Ban />Exclusion</button>
          <button className={tool === "waypoint" ? "active" : ""} onClick={() => onTool("waypoint")}><MapPinned />Waypoint</button>
          <button className={tool === "hold" ? "active" : ""} onClick={() => onTool("hold")}><CircleDot />Hold point</button>
          <button className={tool === "orbit" ? "active" : ""} onClick={() => onTool("orbit")}><RotateCcw />Orbit point</button>
          <button onClick={onUndoGeometry} title="Undo the most recent geometry mutation"><Undo2 />Undo</button>
        </div>
        <div className="geometry-summary">
          <span>{mission.geometry.included_areas.length} operating</span>
          <span>{mission.geometry.exclusion_areas.length} excluded</span>
          <span>{mission.geometry.waypoints.length} waypoints</span>
          <span>{mission.geometry.pois.length} hold/orbit</span>
        </div>
        <div className="geometry-inventory">
          {mission.geometry.included_areas.map((_, index) => <button className={geometryFocus?.kind === "include" && geometryFocus.index === index ? "selected" : ""} key={`include-${index}`} onClick={() => onFocusGeometry({ kind: "include", index })}><span>Operating area {index + 1}</span><Eye /></button>)}
          {mission.geometry.exclusion_areas.map((_, index) => <button className={geometryFocus?.kind === "exclude" && geometryFocus.index === index ? "selected" : ""} key={`exclude-${index}`} onClick={() => onFocusGeometry({ kind: "exclude", index })}><span>Exclusion area {index + 1}</span><Eye /></button>)}
          {mission.geometry.waypoints.map((_, index) => <div className={geometryFocus?.kind === "waypoint" && geometryFocus.index === index ? "selected" : ""} key={`waypoint-${index}`}><button onClick={() => onFocusGeometry({ kind: "waypoint", index })}><span>Waypoint {index + 1}</span><Eye /></button><button disabled={index === 0} onClick={() => onReorderWaypoint(index, -1)}><ChevronUp /></button><button disabled={index === mission.geometry.waypoints.length - 1} onClick={() => onReorderWaypoint(index, 1)}><ChevronDown /></button></div>)}
          {mission.geometry.pois.map((poi, index) => <button className={geometryFocus?.kind === "poi" && geometryFocus.index === index ? "selected" : ""} key={poi.id} onClick={() => onFocusGeometry({ kind: "poi", index })}><span>{poi.kind === "orbit" ? "Orbit" : "Hold"} point {index + 1}</span><Eye /></button>)}
          {geometryFocus && <button className="geometry-delete" onClick={() => onDeleteGeometry(geometryFocus)}><Trash2 />Delete selected</button>}
          <div className="geometry-clear-actions">
            <button onClick={() => onClearGeometry("include")}>Clear operating</button>
            <button onClick={() => onClearGeometry("exclude")}>Clear exclusions</button>
            <button onClick={() => onClearGeometry("waypoint")}>Clear waypoints</button>
            <button onClick={() => onClearGeometry("poi")}>Clear hold/orbit</button>
          </div>
        </div>
      </details>
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
              {draft.planning_mode === "manual"
                ? "MANUAL DETERMINISTIC PLANNER"
                : draft.advisor.provider === "openai"
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
        <div className="planning-actions">
          <button className="wide amber" disabled={busy || !manualReady} onClick={() => onGenerateManual(manualIntent, missionType, followContactID)}>
            <Route /> {pirate ? "Plot deterministic courses" : "Generate routes · no AI"}
          </button>
          <button className="wide" disabled={busy || !command.trim()} onClick={() => onCreate(command.trim())}>
            <Sparkles /> {pirate ? "Ask the ship's intelligence" : "Ask AI for strategy options"}
          </button>
          {!manualReady && <small className="manual-readiness">Assign assets and add the map geometry required by the selected mission type.</small>}
        </div>
      ) : (
        <div className="candidate-list">
          {plans.map((p) => {
            const expanded = expandedPlans.has(p.id);
            return (
            <article
              key={p.id}
              className={`${activePlan?.id === p.id ? "selected" : ""} ${expanded ? "expanded" : "collapsed"}`}
            >
              <header>
                <button className="candidate-select" onClick={() => onPlan(p.id)}>
                  <b>{p.name}</b>
                </button>
                <span>
                  {p.advisor_source}
                  {p.advisor_model ? ` · ${p.advisor_model}` : ""}
                </span>
                {p.recommended && (
                  <em>{pirate ? "CAPTAIN'S PICK" : "RECOMMENDED"}</em>
                )}
                <button
                  className="candidate-expand"
                  aria-label={`${expanded ? "Collapse" : "Expand"} ${p.name}`}
                  onClick={() =>
                    setExpandedPlans((current) => {
                      const next = new Set(current);
                      next.has(p.id) ? next.delete(p.id) : next.add(p.id);
                      return next;
                    })
                  }
                >
                  {expanded ? <ChevronUp /> : <ChevronDown />}
                </button>
              </header>
              <div className="candidate-detail">
                <p>{p.description}</p>
                <small className="maneuver-sequence">
                  {p.maneuvers.join(" → ")}
                </small>
                <PlanMiniMap plan={p} compact />
              </div>
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
              <div className="candidate-detail"><code>{p.content_hash.slice(0, 24)}…</code></div>
            </article>
          )})}
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
      </section>
      </div>
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

function PlanMiniMap({ plan, compact = false }: { plan: FleetPlanV2; compact?: boolean }) {
  const points = plan.assignments.flatMap((assignment) => assignment.route);
  if (points.length < 2) return null;
  const minX = Math.min(...points.map((point) => point[0])),
    maxX = Math.max(...points.map((point) => point[0])),
    minY = Math.min(...points.map((point) => point[1])),
    maxY = Math.max(...points.map((point) => point[1])),
    width = Math.max(maxX - minX, 0.000001),
    height = Math.max(maxY - minY, 0.000001),
    scale = Math.min(164 / width, 34 / height),
    drawnWidth = width * scale,
    drawnHeight = height * scale,
    offsetX = 18 + (164 - drawnWidth) / 2,
    offsetY = 7 + (34 - drawnHeight) / 2,
    colors = ["#e6a63b", "#62c58e", "#59bdd1", "#b895d8", "#e36e62", "#ece8dc"];
  const mapCoordinates = (point: Point): [number, number] => [
    offsetX + (point[0] - minX) * scale,
    offsetY + drawnHeight - (point[1] - minY) * scale,
  ];
  const mapPoint = (point: Point) => mapCoordinates(point).join(",");
  return (
    <figure className={`route-summary ${compact ? "compact" : ""}`}>
      <header>
        <span>ROUTE SCHEMATIC</span>
        <b>{plan.assignments.length} tracks · {plan.duration_minutes.toFixed(1)} min</b>
      </header>
      <svg viewBox="0 0 200 48" role="img" aria-label={`Route schematic for ${plan.name}`} preserveAspectRatio="xMidYMid meet">
        <rect className="route-background" x="1" y="1" width="198" height="46" rx="2" />
        <path className="route-grid" d="M50 2v44M100 2v44M150 2v44M2 24h196" />
        {plan.assignments.map((assignment, index) => {
          const color = colors[index % colors.length], start = assignment.route[0], end = assignment.route.at(-1), startXY = start ? mapCoordinates(start) : null, endXY = end ? mapCoordinates(end) : null;
          return <g key={assignment.vessel_id}>
            <polyline points={assignment.route.map(mapPoint).join(" ")} style={{ stroke: color }} />
            {startXY && <circle cx={startXY[0]} cy={startXY[1]} r="1.8" style={{ fill: color }} />}
            {endXY && <rect x={endXY[0]-1.8} y={endXY[1]-1.8} width="3.6" height="3.6" style={{ fill: color }} />}
          </g>;
        })}
      </svg>
      <figcaption><i className="start-marker" /> START <i className="end-marker" /> END · colored lines are vessel tracks</figcaption>
    </figure>
  );
}
