import { type CSSProperties, type ReactNode, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { api, clientUUID, KeelMeshError, requestID } from "./api";
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
  OperationalGroupV2,
  PlatformSnapshot,
  Point,
  ReachabilityV2,
  SurfaceContactV2,
  VesselProfileV2,
  WorkspaceAssistantActionV1,
  WorkspaceAssistantResponseV1,
  AssistantTurnV2,
  CommandSceneV1,
  ConversationTurnV1,
  MemorySnapshotV1,
} from "./types";
import { KeelMeshA2UISurface } from "./A2UISurface";
import { OperationsMap, type WaypointColor } from "./OperationsMap";
import { WindowManager, type WindowDefinition } from "./WindowManager";
import { HoverHelp } from "./HoverHelp";
import { useLongPressContext } from "./useLongPressContext";
import { EngineerView } from "./EngineerView";
import { PlatformCutaway } from "./PlatformCutaway";
import { ResilienceDrill } from "./ResilienceDrill";
import { QuietFleetDrill } from "./QuietFleetDrill";
import { ArenaView } from "./ArenaView";
import {
  Anchor,
  Activity,
  Ban,
  BatteryCharging,
  Bot,
  BoxSelect,
  CheckCircle2,
  ChevronDown,
  ChevronUp,
  Eye,
  GripVertical,
  ListFilter,
  MapPinned,
  MessageCircle,
  Mic,
  MousePointer2,
  Network,
  Pause,
  Pin,
  History,
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
  Compass,
  Gauge,
  Navigation,
  Satellite,
  Sun,
  Undo2,
} from "lucide-react";

type Tool = "select" | "box" | "waypoint" | "include" | "exclude" | "hold" | "orbit";
type GeometryFocus = { kind: "waypoint" | "include" | "exclude" | "poi"; index: number };
type GlobalVoicePhase =
  | "idle"
  | "listening"
  | "transcribing"
  | "thinking"
  | "received"
  | "synthesizing"
  | "speaking";
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
const reservePercent = (reserve: number) => {
  const percent = Math.max(0, Math.min(100, reserve * 100));
  if (percent === 100) return "100";
  return percent >= 99 ? percent.toFixed(2) : percent.toFixed(1);
};

function commandSceneSessionID() {
  const key = "keelmesh.command-scene-session.v1";
  const existing = sessionStorage.getItem(key);
  if (existing) return existing;
  const value = clientUUID();
  sessionStorage.setItem(key, value);
  return value;
}

export function FleetWorkspace() {
  const sceneSessionID = useRef(commandSceneSessionID()).current;
  const [fleet, setFleet] = useState<FleetSnapshotV2 | null>(null),
    [legacy, setLegacy] = useState<Bootstrap | null>(null),
    [platform, setPlatform] = useState<PlatformSnapshot | null>(null),
    [agent, setAgent] = useState<AgentSnapshot | null>(null),
    [arena, setArena] = useState<ArenaSnapshotV1 | null>(null),
    [memory, setMemory] = useState<MemorySnapshotV1 | null>(null),
    [speechState, setSpeechState] = useState("ready"),
    [globalVoicePhase, setGlobalVoicePhase] = useState<GlobalVoicePhase>("idle"),
    [globalVoiceProgress, setGlobalVoiceProgress] = useState(0),
    [selected, setSelected] = useState<Set<string>>(new Set()),
    [activeMissionID, setActiveMissionID] = useState<string>(""),
    [activeGroupID, setActiveGroupID] = useState<string>(""),
    [tool, setTool] = useState<Tool>("select"),
    [plannerVisible, setPlannerVisible] = useState(false),
    [plannerContactSeed, setPlannerContactSeed] = useState<SurfaceContactV2 | null>(null),
    [geometryFocus, setGeometryFocus] = useState<GeometryFocus | null>(null),
    [search, setSearch] = useState(""),
    [command, setCommand] = useState(""),
    [, setDraft] = useState<CommandDraftV2 | null>(null),
    [plans, setPlans] = useState<FleetPlanV2[]>([]),
    [activePlansByMission, setActivePlansByMission] = useState<Record<string, FleetPlanV2>>({}),
    [planID, setPlanID] = useState(""),
    [preview, setPreview] = useState<FleetPreviewV2 | null>(null),
    [lease, setLease] = useState<FleetLeaseV2 | null>(null),
    [windows, setWindows] = useState<Set<string>>(
      () =>
        new Set(
          window.location.search.includes("arena=1") ? ["arena"] : ["fleet"],
        ),
    ),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [connected, setConnected] = useState(true),
    [pendingDeleteID, setPendingDeleteID] = useState(""),
    [pendingPlanID, setPendingPlanID] = useState(""),
    [commandScenes, setCommandScenes] = useState<CommandSceneV1[]>([]),
    [activeSceneID, setActiveSceneID] = useState(""),
    [sceneCameraRequest, setSceneCameraRequest] = useState<{
      sceneID: string;
      token: number;
    } | null>(null),
    [assistantTurns, setAssistantTurns] = useState<ConversationTurnV1[]>([]),
    [assistantChatInput, setAssistantChatInput] = useState(""),
    [assistantChatBusy, setAssistantChatBusy] = useState(false);
  const audio = useRef<HTMLAudioElement | null>(null),
    speechAbort = useRef<AbortController | null>(null),
    recorder = useRef<MediaRecorder | null>(null),
    recordingStream = useRef<MediaStream | null>(null),
    recordingChunks = useRef<BlobPart[]>([]),
    stopRequested = useRef(false),
    geometryHistory = useRef<Record<string, MissionWorkspaceV2["geometry"][]>>({}),
    missionSelectionSync = useRef("");
  const [windowActivations, setWindowActivations] = useState<
      Record<string, number>
    >({}),
    [windowToggleActivations, setWindowToggleActivations] = useState<
      Record<string, number>
    >({});
  const [pirate, setPirate] = useState(
    () => localStorage.getItem("keelmesh.theme") === "pirate",
  );
  function focusCommandScene(sceneID: string, moveCamera = true) {
    setActiveSceneID(sceneID);
    if (moveCamera)
      setSceneCameraRequest((current) => ({
        sceneID,
        token: (current?.token ?? 0) + 1,
      }));
  }
  function releaseCommandScene(sceneID: string) {
    setActiveSceneID((current) => (current === sceneID ? "" : current));
    setSceneCameraRequest((current) =>
      current?.sceneID === sceneID ? null : current,
    );
  }
  const voice = pirate ? "barbossa" : "jarvis";
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
      api<MemorySnapshotV1>("/api/v5/memory").then(setMemory),
      api<{ scenes: CommandSceneV1[]; turns?: ConversationTurnV1[] }>(`/api/v4/assistant/history?actor_identity=demo-operator&session_id=${encodeURIComponent(sceneSessionID)}`).then((value) => {
        const visible = value.scenes.filter((scene) => scene.state === "active" || scene.pinned);
        setCommandScenes(value.scenes);
        setAssistantTurns(value.turns ?? []);
        if (visible[0]) focusCommandScene(visible[0].id);
        const hasMissionCanvas = visible.some((scene) => scene.type === "mission_canvas");
        setWindows((current) => new Set([...current, ...(hasMissionCanvas ? ["planner"] : []), ...visible.filter((scene) => scene.type !== "mission_canvas").map((scene) => `scene-${scene.id}`)]));
        if (hasMissionCanvas) setPlannerVisible(true);
      }),
    ]);
    const t = window.setInterval(() => {
      refresh()
        .then(() => setConnected(true))
        .catch(() => setConnected(false));
      api<ArenaSnapshotV1>("/api/v3/arena?faction=A")
        .then(setArena)
        .catch(() => {});
      api<MemorySnapshotV1>("/api/v5/memory").then(setMemory).catch(() => {});
      api<{ scenes: CommandSceneV1[]; turns?: ConversationTurnV1[] }>(`/api/v4/assistant/history?actor_identity=demo-operator&session_id=${encodeURIComponent(sceneSessionID)}`)
        .then((value) => {
          setCommandScenes(value.scenes);
          setAssistantTurns(value.turns ?? []);
          const active = value.scenes.find((scene) => scene.critical && scene.state === "active");
          if (active) {
            setActiveSceneID(active.id);
            setWindows((current) => current.has(`scene-${active.id}`) ? current : new Set([...current, `scene-${active.id}`]));
          }
        })
        .catch(() => {});
    }, 1000);
    return () => window.clearInterval(t);
  }, [refresh, sceneSessionID]);
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
  const selectedKey = useMemo(() => [...selected].sort().join(","), [selected]);
  const missionTargetKey = useMemo(
    () => [...(mission?.target_ids ?? [])].sort().join(","),
    [mission?.target_snapshot_hash, mission?.id, mission?.target_ids],
  );
  const pendingDeleteMission =
    fleet?.missions.find((item) => item.id === pendingDeleteID) ?? null;
  const pendingPlan = plans.find((item) => item.id === pendingPlanID) ?? null;
  useEffect(() => {
    if (!activeMissionID && fleet?.missions[0])
      setActiveMissionID(fleet.missions[0].id);
  }, [fleet, activeMissionID]);
  useEffect(() => {
    if (!plannerVisible || !mission) return;
    const marker = `${mission.id}:${missionTargetKey}`;
    missionSelectionSync.current = marker;
    setSelected(new Set(mission.target_ids));
  }, [plannerVisible, mission?.id, missionTargetKey]);
  useEffect(() => {
    if (!mission || (mission.plan_ids ?? []).length === 0 || plans.some((value) => value.mission_id === mission.id)) return;
    let active = true;
    void api<{ plans: FleetPlanV2[] }>(`/api/v2/missions/${mission.id}/plans`).then((value) => {
      if (!active) return;
      setPlans(value.plans);
      setPlanID((value.plans.find((plan) => plan.recommended) ?? value.plans[0])?.id ?? "");
    }).catch(() => undefined);
    return () => { active = false; };
  }, [mission?.id, (mission?.plan_ids ?? []).join("|")]);
  const activeMissionPlanKey = useMemo(
    () => (fleet?.missions ?? [])
      .filter((item) => ["authorized", "executing", "paused"].includes(item.status) && item.authorized_plan_id)
      .map((item) => `${item.id}:${item.authorized_plan_id}`)
      .sort()
      .join("|"),
    [fleet?.missions],
  );
  useEffect(() => {
    const activeMissions = (fleet?.missions ?? []).filter(
      (item) => ["authorized", "executing", "paused"].includes(item.status) && item.authorized_plan_id,
    );
    let active = true;
    void Promise.all(activeMissions.map(async (item) => {
      const response = await api<{ plans: FleetPlanV2[] }>(`/api/v2/missions/${item.id}/plans`);
      return [item.id, response.plans.find((plan) => plan.id === item.authorized_plan_id)] as const;
    })).then((entries) => {
      if (!active) return;
      setActivePlansByMission(Object.fromEntries(entries.filter((entry): entry is readonly [string, FleetPlanV2] => !!entry[1])));
    }).catch(() => undefined);
    return () => { active = false; };
  }, [activeMissionPlanKey]);
  useEffect(() => {
    if (!plannerVisible || !mission) return;
    const marker = `${mission.id}:${missionTargetKey}`;
    if (missionSelectionSync.current === marker) {
      if (selectedKey === missionTargetKey) missionSelectionSync.current = "";
      return;
    }
    if (selectedKey === missionTargetKey) return;
    const timer = window.setTimeout(() => {
      setPlans([]);
      setDraft(null);
      setPreview(null);
      setLease(null);
      void updateMissionTargets(mission.id, [...selected]);
    }, 140);
    return () => window.clearTimeout(timer);
  }, [plannerVisible, mission?.id, missionTargetKey, selectedKey]);
  const activePlan =
    plans.find((p) => p.mission_id === mission?.id && p.id === mission?.authorized_plan_id) ??
    plans.find((p) => p.mission_id === mission?.id && p.id === planID) ??
    plans.find((p) => p.mission_id === mission?.id && p.recommended) ??
    null;
  const activeMissionPlans = useMemo(
    () => (fleet?.missions ?? []).flatMap((item) => {
      if (!["authorized", "executing", "paused"].includes(item.status)) return [];
      const plan = item.id === mission?.id && activePlan?.id === item.authorized_plan_id
        ? activePlan
        : activePlansByMission[item.id];
      return plan ? [{ mission: item, plan }] : [];
    }),
    [fleet?.missions, mission?.id, activePlan, activePlansByMission],
  );
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
  const toggleWindow = useCallback((id: string) => {
    if (!windows.has(id)) {
      open(id);
      return;
    }
    setWindowToggleActivations((current) => ({
      ...current,
      [id]: (current[id] ?? 0) + 1,
    }));
  }, [open, windows]);
  const openGroupManager = useCallback((id: string) => {
    setActiveGroupID(id);
    open(`group-manager-${id}`);
  }, [open]);
  const openVesselInspector = useCallback((id: string) => {
    open(`inspector-${id}`);
  }, [open]);
  const openContactInspector = useCallback((id: string) => {
    open(`contact-inspector-${id}`);
  }, [open]);
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
    objectiveOverride = "",
	showPlanner = true,
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
    const objective = objectiveOverride.trim()
      ? objectiveOverride.trim()
      : namingMode === "ai"
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
	  if (showPlanner) open("planner");
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
    if (g) openGroupManager(g.id);
  }
  async function createGroupFromVessel(vesselID: string, name: string) {
    const g = await createGroupFor([vesselID], name);
    if (g) openGroupManager(g.id);
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
    if (activeGroupID === id) setActiveGroupID("");
    setWindows((value) => {
      const next = new Set(value);
      next.delete(`group-manager-${id}`);
      return next;
    });
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
  async function updateMissionTargets(missionID: string, targetIDs: string[]) {
    const snapshot = await api<FleetSnapshotV2>("/api/v2/fleet");
    const current = snapshot.missions.find((item) => item.id === missionID);
    if (!current) return;
    const next = [...new Set(targetIDs)].sort();
    const existing = [...current.target_ids].sort();
    if (next.join(",") === existing.join(",")) return;
    try {
      await mutate(() =>
        api<MissionWorkspaceV2>(`/api/v2/missions/${missionID}`, {
          method: "PATCH",
          body: JSON.stringify({
            request_id: requestID("mission-targets"),
            idempotency_key: requestID("mission-targets-key"),
            expected_version: current.version,
            target_ids: next,
          }),
        }),
      );
      await refresh();
    } catch {
      missionSelectionSync.current = `${missionID}:${existing.join(",")}`;
      setSelected(new Set(existing));
    }
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
    suppressSpeech = false,
	strategyCount: 1 | 3 = 3,
	showPlanner = true,
  ) {
    if (!targetMission) return null;
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
			strategy_count: strategyCount,
            guidance_kind: guidanceKind,
            follow_contact_id: followContactID,
          }),
        },
      ),
    ).catch(() => null);
    if (!compiled) return null;
    setDraft(compiled);
    setCommand("");
    const current = (await api<FleetSnapshotV2>("/api/v2/fleet")).missions.find(
      (m) => m.id === targetMission.id,
    );
    if (!current) return null;
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
	  if (showPlanner) open("planner");
      await refresh();
	  if (planningMode === "ai_assisted" && compiled.advisor?.summary && !suppressSpeech)
		await speak(compiled.advisor.summary);
    }
	return result ? compiled : null;
  }
  async function enactPlan(chosen: FleetPlanV2, showPlanner = true) {
    const targetMission = fleet?.missions.find((value) => value.id === chosen.mission_id);
    if (!targetMission || chosen.policy_status === "prohibited") return;
    setBusy(true);
    setError("");
    try {
      setActiveMissionID(targetMission.id);
      setPlanID(chosen.id);
	  if (showPlanner) open("planner");
      let current = (await api<FleetSnapshotV2>("/api/v2/fleet")).missions.find((value) => value.id === targetMission.id);
      if (!current) throw new Error("Mission is no longer available.");
      const routePreview = await api<FleetPreviewV2>(`/api/v2/missions/${targetMission.id}/plans/${chosen.id}:preview`, {
        method: "POST",
        body: JSON.stringify({request_id: requestID("preview"), idempotency_key: requestID("preview-key"), expected_version: current.version}),
      });
      setPreview(routePreview);
      current = (await api<FleetSnapshotV2>("/api/v2/fleet")).missions.find((value) => value.id === targetMission.id);
      if (!current) throw new Error("Mission is no longer available.");
      const authority = await api<FleetLeaseV2>(`/api/v2/missions/${targetMission.id}/plans/${chosen.id}:authorize`, {
        method: "POST",
        body: JSON.stringify({request_id: requestID("authorize"), idempotency_key: requestID("authorize-key"), expected_version: current.version, plan_hash: chosen.content_hash, operator_id: "demo-operator"}),
      });
      setLease(authority);
      current = (await api<FleetSnapshotV2>("/api/v2/fleet")).missions.find((value) => value.id === targetMission.id);
      if (!current) throw new Error("Mission is no longer available.");
      await api(`/api/v2/missions/${targetMission.id}/plans/${chosen.id}:start`, {
        method: "POST",
        body: JSON.stringify({request_id: requestID("start"), idempotency_key: `start-${authority.id}`, expected_version: current.version, plan_hash: chosen.content_hash, lease_id: authority.id}),
      });
      setPendingPlanID("");
      await refresh();
    } catch (reason) {
      setError(reason instanceof KeelMeshError ? `${reason.code}: ${reason.message}` : String(reason));
    } finally {
      setBusy(false);
    }
  }
  async function speak(text: string, global = false) {
    speechAbort.current?.abort();
    audio.current?.pause();
    const controller = new AbortController();
    speechAbort.current = controller;
    setSpeechState("synthesizing");
    if (global) {
      setGlobalVoicePhase("synthesizing");
      setGlobalVoiceProgress(0.82);
    }
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
      player.onended = () => {
        setSpeechState("ready");
        if (global) {
          setGlobalVoicePhase("idle");
          setGlobalVoiceProgress(0);
        }
      };
      setSpeechState("speaking");
      if (global) {
        setGlobalVoiceProgress(1);
        setGlobalVoicePhase("speaking");
      }
      await player.play();
    } catch (e) {
      if ((e as Error).name !== "AbortError") {
        setSpeechState("text fallback");
        setError((e as Error).message);
        if (global) {
          setGlobalVoicePhase("idle");
          setGlobalVoiceProgress(0);
        }
      }
    }
  }
  async function captureTranscription(
    onTranscript: (text: string) => Promise<void>,
    global = false,
  ) {
    if (recorder.current?.state === "recording") return;
    speechAbort.current?.abort();
    audio.current?.pause();
    stopRequested.current = false;
    setError("");
    setSpeechState("requesting microphone");
    if (global) {
      setGlobalVoicePhase("listening");
      setGlobalVoiceProgress(0);
    }
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
        if (global) setGlobalVoicePhase("idle");
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
        if (global) {
          setGlobalVoicePhase("transcribing");
          setGlobalVoiceProgress(0.16);
        }
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
          // Releasing the microphone is a send action, so do not leave the
          // accepted transcript behind as if it were an unsent draft.
          setCommand("");
          setSpeechState(
            `${result.route} · RTF ${result.real_time_factor ?? "—"}`,
          );
          await onTranscript(result.text);
        } catch (e) {
          setSpeechState("typed fallback");
          setError((e as Error).message);
          if (global) {
            setGlobalVoicePhase("idle");
            setGlobalVoiceProgress(0);
          }
        }
      };
      active.start(200);
      setSpeechState("listening · release to send");
    } catch {
      setSpeechState("typed fallback");
      setError(
        "Microphone access requires HTTPS. Use the ephemeral Cloudflare tunnel or typed input.",
      );
      if (global) {
        setGlobalVoicePhase("idle");
        setGlobalVoiceProgress(0);
      }
    }
  }
  function endTranscription() {
    stopRequested.current = true;
    if (recorder.current?.state === "recording") recorder.current.stop();
  }

  function normalizeWindowTarget(target: string) {
    const value = target.toLowerCase().trim().replaceAll(" ", "_");
    if (["mission", "mission_planner", "planner"].includes(value)) return "planner";
    if (["fleet", "fleet_groups", "groups"].includes(value)) return "fleet";
    if (["autonomy_engineer", "engineer"].includes(value)) return "engineer";
    if (["infrastructure", "live_cutaway", "cutaway"].includes(value)) return "cutaway";
    if (["fleet_arena", "arena"].includes(value)) return "arena";
    if (["quiet_fleet", "quiet"].includes(value)) return "quiet";
    if (["resilience_drill", "resilience"].includes(value)) return "resilience";
    return value;
  }

  function resolveVessel(target: string) {
    const needle = target.toLowerCase().trim();
    return fleet?.vessels.find((value) =>
      [value.id, value.callsign, value.designation, value.display_name]
        .some((candidate) => candidate.toLowerCase() === needle),
    );
  }

  function resolveGroup(target: string) {
    const needle = target.toLowerCase().trim();
    return fleet?.groups.find((value) =>
      [value.id, value.code, value.name, `${value.color_name} team`, `${value.color_name} group`]
        .some((candidate) => candidate.toLowerCase() === needle),
    );
  }

  function resolveContact(target: string) {
    const needle = target.toLowerCase().trim();
    return fleet?.surface_contacts.find((value) =>
      [value.id, value.name, value.boat_id, value.callsign]
        .some((candidate) => candidate.toLowerCase() === needle),
    );
  }

  function resolveMission(target: string) {
    const needle = target.toLowerCase().trim();
    return fleet?.missions.find((value) =>
      [value.id, value.name].some((candidate) => candidate.toLowerCase() === needle),
    );
  }

  function resolveActionVesselIDs(action: WorkspaceAssistantActionV1) {
	const aliases = action.target_ids?.length > 0 ? action.target_ids : [...selected];
    return [...new Set(aliases.map((value) => resolveVessel(value)?.id).filter((value): value is string => Boolean(value)))];
  }

  async function applyAssistantAction(action: WorkspaceAssistantActionV1) {
    const target = normalizeWindowTarget(action.target);
    if (action.kind === "open_window") open(target);
    if (action.kind === "close_window") {
      if (target === "planner") setPlannerVisible(false);
      setWindows((current) => {
        const next = new Set(current);
        next.delete(target);
        return next;
      });
    }
    if (action.kind === "select_all") {
      revealFleet();
      setSelected(new Set(fleet?.vessels.map((value) => value.id) ?? []));
    }
    if (action.kind === "clear_selection") setSelected(new Set());
    if (action.kind === "select_group") {
      const group = resolveGroup(action.target);
      if (group) selectGroup(group.id);
    }
    if (action.kind === "select_vessel") {
      const vessel = resolveVessel(action.target);
      if (vessel) select([vessel.id], "replace");
    }
    if (action.kind === "inspect_group") {
      const group = resolveGroup(action.target);
      if (group) openGroupManager(group.id);
    }
    if (action.kind === "inspect_vessel") {
      const vessel = resolveVessel(action.target);
      if (vessel) openVesselInspector(vessel.id);
    }
    if (action.kind === "inspect_contact") {
      const contact = resolveContact(action.target);
      if (contact) openContactInspector(contact.id);
    }
    if (action.kind === "set_theme") setPirate(action.target.toLowerCase() === "pirate");
    if (action.kind === "set_simulation_rate") {
      const rate = [0, 1, 5, 20, 100, 500].includes(action.value) ? action.value : 20;
      await setSimulationRate(rate as FleetSnapshotV2["simulation_rate"]);
    }
	if (action.kind === "open_mission") {
	  const targetMission = resolveMission(action.target);
	  if (targetMission) {
		setActiveMissionID(targetMission.id);
		setSelected(new Set(targetMission.target_ids));
		open("planner");
	  }
	}
	if (action.kind === "pause_mission" || action.kind === "resume_mission") {
	  const targetMission = resolveMission(action.target);
	  if (targetMission) await setMissionStatus(targetMission.id, action.kind === "pause_mission" ? "paused" : "executing");
	}
	if (action.kind === "delete_mission") {
	  const targetMission = resolveMission(action.target);
	  if (targetMission) deleteMission(targetMission.id);
	}
	if (action.kind === "create_group") {
	  const vesselIDs = resolveActionVesselIDs(action);
	  if (vesselIDs.length > 0) await createGroupFor(vesselIDs, action.name);
	}
	if (action.kind === "delete_group") {
	  const group = resolveGroup(action.target);
	  if (group) await deleteGroup(group.id);
	}
	if (action.kind === "move_vessel_to_group") {
	  const vessel = resolveVessel(action.target);
	  const destination = action.secondary_target ?? "";
	  const group = resolveGroup(destination);
	  if (vessel && (group || destination.toLowerCase().trim() === "unassigned"))
		await moveVessel(vessel.id, group?.id ?? "unassigned");
	}
  }

  function requestsMissionOptions(text: string) {
	return /\b(options?|alternatives?|choices?|compare|comparison|strateg(?:y|ies)|several|multiple|(?:two|three|2|3)\s+(?:plans?|ways?|routes?|formations?)|different\s+(?:plans?|routes?|formations?))\b/i.test(text);
  }

  function planOptionPayload() {
    return plans.slice(0, 3).map((value, index) => ({
      label: String.fromCharCode(65 + index),
      plan_id: value.id,
      name: value.name,
      content_hash: value.content_hash,
      policy_status: value.policy_status,
    }));
  }

  async function askWorkspaceAssistant(text: string, presentScene = true) {
    const turn = await api<AssistantTurnV2>("/api/v4/assistant/turns", {
      method: "POST",
      body: JSON.stringify({
        schema_version: 1,
        request_id: requestID("workspace-assistant"),
        idempotency_key: requestID("workspace-assistant-key"),
        text,
        persona: pirate ? "pirate" : "navy",
        selected_ids: [...selected],
        open_windows: [...windows],
        active_mission_id: mission?.id ?? "",
        plan_options: planOptionPayload(),
        actor_identity: "demo-operator",
        session_id: sceneSessionID,
        workspace_version: fleet?.fleet_version ?? 0,
      }),
    });
    setCommandScenes((current) => [turn.scene, ...current.map((scene) => scene.state === "active" && !scene.pinned && !scene.critical && !scene.pending_approval ? { ...scene, state: "replaced" } : scene).filter((scene) => scene.id !== turn.scene.id)].slice(0, 50));
	if (presentScene) {
	  focusCommandScene(turn.scene.id);
	  if (turn.scene.type === "mission_canvas") open("planner");
	  else open(`scene-${turn.scene.id}`);
	}
    return turn.assistant;
  }

  async function mutateScene(scene: CommandSceneV1, operation: "pin" | "unpin" | "dismiss") {
    const value = await api<CommandSceneV1>(`/api/v4/scenes/${scene.id}:${operation}`, {
      method: "POST",
      body: JSON.stringify({
        request_id: requestID(`scene-${operation}`),
        idempotency_key: requestID(`scene-${operation}-key`),
        actor_identity: "demo-operator",
        session_id: sceneSessionID,
        workspace_version: scene.workspace_version,
      }),
    });
    setCommandScenes((current) => operation === "dismiss" ? current.filter((item) => item.id !== scene.id) : current.map((item) => item.id === scene.id ? value : item));
    if (operation === "dismiss") {
      releaseCommandScene(scene.id);
      setWindows((current) => { const next = new Set(current); next.delete(`scene-${scene.id}`); return next; });
    }
  }

  async function applySceneAction(scene: CommandSceneV1, action: CommandSceneV1["suggested_actions"][number]) {
    if (action.kind === "pin_scene") { await mutateScene(scene, "pin"); return; }
    if (action.kind === "dismiss_scene") { await mutateScene(scene, "dismiss"); return; }
    if (action.kind === "frame_entities" && scene.map_camera) {
      focusCommandScene(scene.id);
      return;
    }
    if (action.kind === "open_window") open("planner");
    if (action.kind === "open_edit_drawer") open("planner");
    await api<CommandSceneV1>(`/api/v4/scenes/${scene.id}/actions`, {
      method: "POST",
      body: JSON.stringify({
        request_id: requestID("scene-action"), idempotency_key: requestID("scene-action-key"),
        actor_identity: "demo-operator", session_id: sceneSessionID, workspace_version: scene.workspace_version,
        action_id: action.id, action_hash: action.action_hash, confirmed: action.authority_class !== "effect",
      }),
    });
  }

  function chosenPlan(action: WorkspaceAssistantActionV1) {
    if (action.kind !== "choose_plan") return null;
    const target = action.target.trim().toLowerCase();
    return plans.slice(0, 3).find((value, index) =>
      target === String.fromCharCode(97 + index) ||
      target === value.id.toLowerCase() ||
      target === value.name.toLowerCase(),
    ) ?? null;
  }

  async function missionWithObjective(target: MissionWorkspaceV2, objective: string) {
    const snapshot = await api<FleetSnapshotV2>("/api/v2/fleet");
    let current = snapshot.missions.find((value) => value.id === target.id);
    if (!current) return null;
    const nextObjective = objective.trim();
    if (nextObjective && nextObjective !== current.objective) {
      const updated = await mutate(() =>
        api<MissionWorkspaceV2>(`/api/v2/missions/${current!.id}`, {
          method: "PATCH",
          body: JSON.stringify({
            request_id: requestID("mission-objective"),
            idempotency_key: requestID("mission-objective-key"),
            expected_version: current!.version,
            objective: nextObjective,
          }),
        }),
      ).catch(() => null);
      if (!updated) return null;
      current = updated;
      await refresh();
    }
    return current;
  }

  async function generateManualMission(missionType: string, objective: string) {
    if (!mission) return;
    const current = await missionWithObjective(mission, objective);
    if (!current) return;
    const intent = objective.trim() || current.objective.trim() || `${missionType.replaceAll("_", " ")} mission`;
    await createPlans(current, intent, "manual", missionType, "", true, 1, true);
  }

  async function refineMissionWithAI(missionType: string, objective: string, instruction: string, alternatives: boolean) {
    if (!mission) return;
    const current = await missionWithObjective(mission, objective);
    if (!current) return;
    const base = objective.trim() || current.objective.trim() || `${missionType.replaceAll("_", " ")} mission`;
    const intent = instruction.trim() ? `${base}\n\nRefinement instruction: ${instruction.trim()}` : `Review and refine this ${base}.`;
    await createPlans(current, intent, "ai_assisted", missionType, "", false, alternatives ? 3 : 1, true);
  }

  async function handleGlobalTranscript(text: string) {
    setGlobalVoicePhase("thinking");
    setGlobalVoiceProgress(0.3);
    try {
	  const wantsOptions = requestsMissionOptions(text);
	  const response = await askWorkspaceAssistant(text, wantsOptions);
      setGlobalVoicePhase("received");
      setGlobalVoiceProgress(0.68);
      const choice = response.actions.map(chosenPlan).find((value) => value !== null);
      if (choice) {
		await enactPlan(choice, false);
        await speak(response.speech, true);
        return;
      }
      for (const action of response.actions) {
        if (action.kind !== "create_mission" && action.kind !== "none")
          await applyAssistantAction(action);
      }
      if (response.mode === "mission") {
        const intent = response.mission_intent.trim() || text;
		const created = await createMissionFor([], "ai", intent, wantsOptions);
        if (created) {
		  const compiled = await createPlans(created, intent, "ai_assisted", "", "", true, wantsOptions ? 3 : 1, wantsOptions);
          await speak(compiled?.advisor?.summary || response.speech, true);
          return;
        }
      }
      await speak(response.speech, true);
    } catch (e) {
      setGlobalVoicePhase("idle");
      setGlobalVoiceProgress(0);
      setError(e instanceof KeelMeshError ? `${e.code}: ${e.message}` : String(e));
    }
  }

  async function beginGlobalTranscription() {
    await captureTranscription(handleGlobalTranscript, true);
  }
  async function handleGlobalTypedMessage(text: string) {
    const value = text.trim();
    if (!value || assistantChatBusy) return;
    setAssistantChatBusy(true);
    setAssistantChatInput("");
    setError("");
    try {
	  const wantsOptions = requestsMissionOptions(value);
	  const response = await askWorkspaceAssistant(value, wantsOptions);
      const choice = response.actions.map(chosenPlan).find((candidate) => candidate !== null);
      if (choice) {
		await enactPlan(choice, false);
      } else {
        for (const action of response.actions) {
          if (action.kind !== "create_mission" && action.kind !== "none")
            await applyAssistantAction(action);
        }
        if (response.mode === "mission") {
          const intent = response.mission_intent.trim() || value;
		  const created = await createMissionFor([], "ai", intent, wantsOptions);
		  if (created) await createPlans(created, intent, "ai_assisted", "", "", true, wantsOptions ? 3 : 1, wantsOptions);
        }
      }
      const history = await api<{ turns?: ConversationTurnV1[] }>(`/api/v4/assistant/history?actor_identity=demo-operator&session_id=${encodeURIComponent(sceneSessionID)}`);
      setAssistantTurns(history.turns ?? []);
    } catch (reason) {
      setError(reason instanceof KeelMeshError ? `${reason.code}: ${reason.message}` : String(reason));
    } finally {
      setAssistantChatBusy(false);
    }
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
      initialDock: "left",
      title: pirate ? "Flotilla" : "Fleet",
      icon: <Ship />,
      activation: windowActivations.fleet,
      toggleActivation: windowToggleActivations.fleet,
      minWidth: 245,
      minHeight: 180,
      initial: { x: 10, y: 92, width: 245, height: 600 },
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
            openGroupManager(id);
          }}
          onInspect={openVesselInspector}
          onMove={moveVessel}
          onCreateGroup={createGroup}
          onCreateGroupFromVessel={createGroupFromVessel}
          onDeleteGroup={deleteGroup}
          onHoldGroupAtVessel={holdGroupAtVessel}
        />
      ),
    });
  for (const [index, group] of fleet.groups.filter((item) => windows.has(`group-manager-${item.id}`)).entries())
    defs.push({
      id: `group-manager-${group.id}`,
      kind: "context",
      activation: windowActivations[`group-manager-${group.id}`],
      minWidth: 340,
      minHeight: 240,
      title: `${pirate ? "Crew" : "Group"} · ${group.code}`,
      icon: <Users />,
      initial: { x: 280 + (index % 5) * 28, y: 96 + (index % 5) * 24, width: 390, height: 440 },
      content: (
        <GroupManager
          key={`${group.id}-${group.revision}`}
          group={group}
          vessels={vesselsByID}
          onSave={(v) => patchGroup(group.id, v)}
          onDelete={() => deleteGroup(group.id)}
        />
      ),
    });
  for (const [index, vessel] of fleet.vessels.filter((item) => windows.has(`inspector-${item.id}`)).entries())
    defs.push({
      id: `inspector-${vessel.id}`,
      kind: "context",
      activation: windowActivations[`inspector-${vessel.id}`],
      minWidth: 340,
      minHeight: 300,
      title: vessel.display_name,
      icon: <Eye />,
      initial: { x: 310 + (index % 7) * 26, y: 92 + (index % 7) * 22, width: 390, height: 610 },
      content: (
        <VesselInspectorWindow
          pirate={pirate}
          vessel={vessel}
          lookup={vesselsByID}
          onRename={(name) => renameVessel(vessel.id, name)}
        />
      ),
    });
  for (const [index, contact] of fleet.surface_contacts.filter((item) => windows.has(`contact-inspector-${item.id}`)).entries())
    defs.push({
      id: `contact-inspector-${contact.id}`,
      kind: "context",
      activation: windowActivations[`contact-inspector-${contact.id}`],
      minWidth: 310,
      minHeight: 300,
      title: contact.name,
      icon: <Ship />,
      initial: { x: 370 + (index % 6) * 25, y: 105 + (index % 6) * 22, width: 360, height: 540 },
      content: <SurfaceContactInspector contact={contact} />,
    });
  if (windows.has("planner"))
    defs.push({
      id: "planner",
      kind: "primary",
      maximizable: true,
      preferredDock: "right",
      initialDock: "right",
      onVisibilityChange: (visible) => setPlannerVisible(visible),
      activation: windowActivations.planner,
      toggleActivation: windowToggleActivations.planner,
      autoSize: false,
      minWidth: 350,
      minHeight: 245,
      title: pirate ? "Voyage" : "Mission",
      icon: <Route />,
      initial: { x: window.innerWidth - 370, y: 92, width: 350, height: 680 },
      content: (
        <MissionCanvas
          pirate={pirate}
          mission={mission}
          groups={fleet.groups}
          plans={plans}
          activePlan={activePlan}
          preview={preview}
          lease={lease}
          busy={busy}
          tool={tool}
          contactSeed={plannerContactSeed}
          geometryFocus={geometryFocus}
          onFormation={(f) => updateMission({ formation: f })}
          onLoop={(loop) => {
            setDraft(null);
            setPlans([]);
            setPlanID("");
            setPreview(null);
            setLease(null);
            void updateMission({ loop });
          }}
          onObjective={(objective) => updateMission({ objective })}
          onArea={(kind) => setTool(kind)}
          onTool={setTool}
          onGenerateManual={(missionType, objective) => void generateManualMission(missionType, objective)}
          onRefineAI={(missionType, objective, instruction, alternatives) => void refineMissionWithAI(missionType, objective, instruction, alternatives)}
          onOpenConstraints={() => open("constraints")}
          onApplyContactSeed={applyContactSeed}
          onClearContactSeed={() => setPlannerContactSeed(null)}
          onUndoGeometry={() => void undoGeometry()}
          onClearGeometry={(kind) => void clearGeometry(kind)}
          onDeleteGeometry={(focus) => void deleteGeometry(focus)}
          onFocusGeometry={setGeometryFocus}
          onReorderWaypoint={(index, direction) => void reorderWaypoint(index, direction)}
          onChoose={(id) => {
            setPlanID(id);
          }}
          onConfirmPlan={(id) => setPendingPlanID(id)}
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
      minWidth: 320,
      minHeight: 340,
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
      toggleActivation: windowToggleActivations.engineer,
      autoSize: false,
      maximizable: true,
      minWidth: 760,
      minHeight: 540,
      title: pirate ? "Fleet Shipwright" : "Autonomy Engineer",
      icon: <Bot />,
      initial: {
        x: 70,
        y: 82,
        width: Math.min(1140, window.innerWidth - 120),
        height: Math.min(740, window.innerHeight - 146),
      },
      content: (
        <EngineerView value={agent} memory={memory} onChange={setAgent} onError={setError} />
      ),
    });
  if (windows.has("cutaway") && platform && legacy)
    defs.push({
      id: "cutaway",
      kind: "primary",
      activation: windowActivations.cutaway,
      toggleActivation: windowToggleActivations.cutaway,
      autoSize: false,
      maximizable: true,
      minWidth: 900,
      minHeight: 520,
      title: pirate ? "Below Deck Systems" : "Live Infrastructure Cutaway",
      icon: <Network />,
      initial: {
        x: 80,
        y: 82,
        width: Math.min(1220, window.innerWidth - 100),
        height: Math.min(680, window.innerHeight - 146),
      },
      content: (
        <PlatformCutaway
          value={platform}
          fleet={legacy.snapshot}
          memory={memory}
          onError={setError}
        />
      ),
    });
  if (windows.has("resilience") && legacy)
    defs.push({
      id: "resilience",
      kind: "primary",
      activation: windowActivations.resilience,
      toggleActivation: windowToggleActivations.resilience,
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
      toggleActivation: windowToggleActivations.quiet,
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
      toggleActivation: windowToggleActivations.arena,
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
  for (const [index, scene] of commandScenes.entries()) {
    if (scene.state !== "active" || scene.type === "mission_canvas" || !windows.has(`scene-${scene.id}`)) continue;
    defs.push({
      id: `scene-${scene.id}`,
      kind: "context",
      activation: windowActivations[`scene-${scene.id}`],
      minWidth: 320,
      minHeight: 220,
      maximizable: true,
      title: scene.title,
      icon: <Sparkles />,
      initial: { x: Math.max(280, window.innerWidth - 760 - index * 24), y: 110 + index * 26, width: 420, height: 390 },
      onVisibilityChange: (visible) => {
        if (visible) setActiveSceneID(scene.id);
        else releaseCommandScene(scene.id);
      },
      content: <SceneArtifact scene={scene} onAction={(action) => void applySceneAction(scene, action)} onPin={() => void mutateScene(scene, scene.pinned ? "unpin" : "pin")} onDismiss={() => void mutateScene(scene, "dismiss")} />,
    });
  }
  if (windows.has("assistant-chat"))
    defs.push({
      id: "assistant-chat", kind: "context", title: pirate ? "Ship's Intelligence" : "KeelMesh Assistant", icon: <MessageCircle />,
      toggleActivation: windowToggleActivations["assistant-chat"],
      initial: { x: Math.max(20, window.innerWidth - 470), y: Math.max(90, window.innerHeight - 620), width: 430, height: 520 }, minWidth: 310, minHeight: 260,
      preferredDock: "right", maximizable: true, minimizable: false, toggleMode: "close",
      content: <AssistantChat turns={assistantTurns} value={assistantChatInput} busy={assistantChatBusy} pirate={pirate} onChange={setAssistantChatInput} onSend={(text) => void handleGlobalTypedMessage(text)} onOpenScene={(scene) => { focusCommandScene(scene.id); if (scene.type === "mission_canvas") open("planner"); else open(`scene-${scene.id}`); }} scenes={commandScenes} />,
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
        <nav>
          <button aria-label={words.fleet} title="Show or hide fleet and operational groups" onClick={() => toggleWindow("fleet")}>
            <Ship />
            <span>{words.fleet}</span>
          </button>
          <button aria-label={words.mission} title="Show or hide the active mission planner" onClick={() => {
            if (mission) {
              missionSelectionSync.current = `${mission.id}:${[...mission.target_ids].sort().join(",")}`;
              setSelected(new Set(mission.target_ids));
            }
            toggleWindow("planner");
          }}>
            <Route />
            <span>{words.mission}</span>
          </button>
          <button aria-label={words.engineer} title="Show or hide autonomy incident investigation" onClick={() => toggleWindow("engineer")}>
            <Bot />
            <span>{words.engineer}</span>
          </button>
          <button aria-label={words.cutaway} title="Show or hide the live infrastructure cutaway" onClick={() => toggleWindow("cutaway")}>
            <Network />
            <span>{words.cutaway}</span>
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
                const sameMission = m.id === mission?.id;
                missionSelectionSync.current = `${m.id}:${[...m.target_ids].sort().join(",")}`;
                setSelected(new Set(m.target_ids));
                setActiveMissionID(m.id);
                setPlannerContactSeed(null);
                setPlans([]);
                setDraft(null);
                setPreview(null);
                setLease(null);
                if (sameMission) toggleWindow("planner");
                else open("planner");
              }}
              title={`Show or hide ${m.name}`}
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
        activeMissionPlans={activeMissionPlans}
        tool={tool}
        editingEnabled={plannerVisible && !!mission}
        focusedGeometry={geometryFocus}
        onSelect={(ids, mode) => {
          select(ids, mode);
        }}
        onGroup={selectGroup}
        onVessel={openVesselInspector}
        onOpenFleet={revealFleet}
        onContact={openContactInspector}
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
        sceneAnnotations={commandScenes.find((scene) => scene.id === activeSceneID && scene.state === "active")?.map_annotations ?? []}
        sceneCamera={commandScenes.find((scene) => scene.id === sceneCameraRequest?.sceneID && scene.state === "active")?.map_camera}
        sceneCameraRequest={sceneCameraRequest?.token}
      />
      <button className="assistant-chat-trigger" aria-label="Toggle text chat with KeelMesh AI" title="Open or close text chat" onClick={() => toggleWindow("assistant-chat")}><MessageCircle /><span className="assistant-chat-dots" aria-hidden="true"><i /><i /><i /></span></button>
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
      {pendingPlan && (
        <div className="mission-delete-backdrop">
          <section className="mission-delete-dialog plan-confirm-dialog" role="dialog" aria-modal="true" aria-labelledby="plan-confirm-title">
            <header>
              <ShieldCheck />
              <div>
                <small>{pirate ? "CAPTAIN'S APPROVAL CARD" : `APPROVAL CARD · OPTION ${String.fromCharCode(65 + plans.findIndex((value) => value.id === pendingPlan.id))}`}</small>
                <h2 id="plan-confirm-title">{pendingPlan.name}</h2>
              </div>
            </header>
            <p>{pendingPlan.description}</p>
            <dl>
              <span><small>RESERVE</small>{reservePercent(pendingPlan.minimum_reserve)}%</span>
              <span><small>DURATION</small>{pendingPlan.duration_minutes.toFixed(1)} min</span>
              <span><small>SEPARATION</small>{pendingPlan.minimum_separation_m} m</span>
            </dl>
            <code>SHA-256 {pendingPlan.content_hash.slice(0, 24)}…</code>
            <p className="confirmation-note">This single confirmation previews, authorizes this exact hash, and starts the mission. Any stale or changed state fails closed.</p>
            <div>
              <button autoFocus onClick={() => setPendingPlanID("")}>{pirate ? "Belay that" : "Cancel"}</button>
              <button className="confirm" disabled={busy || pendingPlan.policy_status === "prohibited"} onClick={() => void enactPlan(pendingPlan)}>
                <Play />{busy ? "VALIDATING…" : pirate ? "Confirm and make sail" : "Confirm and execute"}
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
      <HoverHelp />
      <div
        className={`global-voice-orb ${globalVoicePhase}`}
        style={{ "--voice-progress": `${globalVoiceProgress * 360}deg` } as CSSProperties}
        data-phase={globalVoicePhase}
      >
        <button
          type="button"
          aria-label={
            globalVoicePhase === "listening"
              ? "Listening; release to send"
              : globalVoicePhase === "speaking"
                ? "Hold to interrupt and speak"
                : globalVoicePhase === "idle"
                  ? "Hold to speak to KeelMesh AI"
                  : `KeelMesh AI ${globalVoicePhase}`
          }
          title="Hold to speak to KeelMesh AI"
          disabled={!["idle", "listening", "speaking"].includes(globalVoicePhase)}
          onContextMenu={(event) => event.preventDefault()}
          onPointerDown={(event) => {
            if (event.button !== 0 || !["idle", "speaking"].includes(globalVoicePhase)) return;
            event.preventDefault();
            event.currentTarget.setPointerCapture(event.pointerId);
            void beginGlobalTranscription();
          }}
          onPointerUp={(event) => {
            if (event.currentTarget.hasPointerCapture(event.pointerId))
              event.currentTarget.releasePointerCapture(event.pointerId);
            endTranscription();
          }}
          onPointerCancel={endTranscription}
          onKeyDown={(event) => {
            if ((event.key === " " || event.key === "Enter") && !event.repeat && ["idle", "speaking"].includes(globalVoicePhase)) {
              event.preventDefault();
              void beginGlobalTranscription();
            }
          }}
          onKeyUp={(event) => {
            if (event.key === " " || event.key === "Enter") {
              event.preventDefault();
              endTranscription();
            }
          }}
        >
          <Mic />
        </button>
      </div>
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
        <span className="status-clock">{new Date(fleet.generated_at).toLocaleTimeString()}</span>
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
function vesselFleetHelp(vessel: VesselProfileV2) {
  const telemetry = vessel.telemetry;
  const group = vessel.group_code
    ? `${vessel.group_color_name} · ${vessel.group_code}`
    : "Unassigned";
  return [
    `${vessel.callsign} · ${vessel.designation} · ${vessel.class.name}`,
    `${telemetry.mode.replaceAll("_", " ")} · ${telemetry.speed_mps.toFixed(1)} m/s · heading ${Math.round(telemetry.heading_deg)}°`,
    `Reserve ${reservePercent(telemetry.reserve)}% · projected ${reservePercent(telemetry.projected_reserve)}%`,
    `PNT ${telemetry.pnt_integrity} ±${Math.round(telemetry.uncertainty_m)} m · ${telemetry.health}`,
    `Group ${group}`,
  ].join("\n");
}

function groupFleetHelp(group: OperationalGroupV2, fleet: FleetSnapshotV2) {
  const members = group.member_ids
    .map((id) => fleet.vessels.find((vessel) => vessel.id === id))
    .filter((vessel): vessel is VesselProfileV2 => !!vessel);
  const averageReserve = members.length
    ? members.reduce((sum, vessel) => sum + vessel.telemetry.reserve, 0) / members.length
    : 0;
  const underway = members.filter((vessel) => vessel.telemetry.speed_mps > 0.1).length;
  const available = members.filter((vessel) => vessel.available).length;
  const decisionNode = members.find((vessel) => vessel.id === group.decision_node_id);
  return [
    `${group.code} · ${group.name} · ${group.color_name} team`,
    `${available}/${members.length} available · ${underway} underway`,
    `${group.formation.replaceAll("_", " ")} · ${group.formation_spacing_m} m spacing · heading ${Math.round(group.formation_heading_deg)}°`,
    `Average reserve ${reservePercent(averageReserve)}% · ${group.route_mode.replaceAll("_", " ")}`,
    `Decision node ${decisionNode?.callsign || "Unavailable"} · epoch ${group.decision_epoch}`,
  ].join("\n");
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
  onHoldGroupAtVessel: (groupID: string, vesselID: string) => void;
}) {
  const [dropGroup, setDropGroup] = useState(""),
    [menu, setMenu] = useState<VesselGroupMenuState | null>(null);
  const longPressMenu = useLongPressContext<VesselProfileV2>((vessel, point) =>
    setMenu({ vessel, x: point.x, y: point.y }),
  );
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
            data-group-drop={g.code}
            aria-label={`Drop a vessel into ${g.code} ${g.name}`}
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
          >
            <div className="group-row-wrap">
              <button
                aria-label={`${g.code} ${g.name}`}
                className="group-row"
                data-help={groupFleetHelp(g, fleet)}
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
                  tabIndex={0}
                  aria-label={`${v.callsign}, ${v.designation}. Long press or press Shift F10 for vessel actions.`}
                  {...longPressMenu(v)}
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
                  data-help={vesselFleetHelp(v)}
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
                  <em>{reservePercent(v.telemetry.reserve)}%</em>
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
                  tabIndex={0}
                  aria-label={`${v.callsign}, ${v.designation}. Long press or press Shift F10 for vessel actions.`}
                  {...longPressMenu(v)}
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
                  data-help={vesselFleetHelp(v)}
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
                  <em>{reservePercent(v.telemetry.reserve)}%</em>
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
  const longPressMenu = useLongPressContext<VesselProfileV2>((vessel, point) =>
    setMenu({ vessel, x: point.x, y: point.y }),
  );
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
                  tabIndex={0}
                  aria-label={`${v.callsign}, ${v.designation}. Long press or press Shift F10 for vessel actions.`}
                  {...longPressMenu(v)}
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
                  <em>{reservePercent(v.telemetry.reserve)}%</em>
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
                <em>{reservePercent(v.telemetry.reserve)}%</em>
                <Eye />
              </button>
            ))}
          </section>
        );
      })}
    </div>
  );
}
function FormationPreview({ formation, color, count }: { formation: string; color: string; count: number }) {
  const patterns: Record<string, [number, number][]> = {
    column: [[50,14],[50,29],[50,44],[50,59],[50,74],[50,89]],
    trail: [[50,14],[50,29],[50,44],[50,59],[50,74],[50,89]],
    line_abreast: [[12,52],[28,52],[44,52],[60,52],[76,52],[92,52]],
    wedge: [[50,16],[36,34],[64,34],[22,56],[78,56],[50,76]],
    echelon_left: [[78,18],[66,32],[54,46],[42,60],[30,74],[18,88]],
    echelon_right: [[22,18],[34,32],[46,46],[58,60],[70,74],[82,88]],
    parallel_columns: [[36,24],[64,24],[36,50],[64,50],[36,76],[64,76]],
    dispersed_screen: [[18,26],[50,14],[82,30],[28,67],[66,60],[88,84]],
    ring: [[50,14],[78,31],[78,65],[50,84],[22,65],[22,31]],
    orbit: [[50,14],[78,31],[78,65],[50,84],[22,65],[22,31]],
    search_grid: [[22,28],[50,28],[78,28],[22,70],[50,70],[78,70]],
  };
  const points = patterns[formation] ?? patterns.column;
  return <svg viewBox="0 0 100 100" role="img" aria-label={`${formation.replaceAll("_", " ")} formation preview`}>
    <path d="M50 8 L50 92" />
    {points.slice(0, Math.max(1, Math.min(count, 6))).map(([x,y], index) => <g key={`${x}-${y}`}><circle cx={x} cy={y} r={index === 0 ? 5 : 4} style={{ fill: index === 0 ? "#f3efe5" : color }} /><path className="formation-bow" d={`M${x} ${y-6} l3 4 h-6 z`} /></g>)}
  </svg>;
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
      <div className="group-hero" style={{ "--group-color": color } as CSSProperties}>
        <div className="group-formation-card">
          <FormationPreview formation={formation} color={color} count={members.length} />
        </div>
        <div className="group-identity">
          <small>PRIMARY OPERATIONAL GROUP · REV {group.revision}</small>
          <h2>{group.code} · {name}</h2>
          <span>{group.color_name} team · {members.length} exclusive members</span>
          <div className="group-state-chips">
            <b><Activity /> {attention ? `${attention} ATTENTION` : "NOMINAL"}</b>
            <b><Navigation /> {moving ? `${moving} UNDERWAY` : "STATION KEEP"}</b>
          </div>
        </div>
      </div>
      <div className="group-insight-strip">
        <div><small>ENERGY</small><strong>{reservePercent(averageReserve)}%</strong><span>min {reservePercent(minimumReserve)}%</span></div>
        <div><small>FORMATION</small><strong>{formation.replaceAll("_", " ")}</strong><span>{spacing} m spacing</span></div>
        <div><small>DECISION NODE</small><strong>{decisionNode?.callsign || "Unavailable"}</strong><span>epoch {group.decision_epoch}</span></div>
      </div>
      <div className="group-decision-note">
        <ShieldCheck />
        <span><b>BOUNDED GROUP AUTONOMY</b> {group.decision_policy.replaceAll("_", " ")} · outside-guardrail decisions request operator instruction.</span>
      </div>
      <details className="group-config-section">
        <summary><SlidersHorizontal /><b>IDENTITY & STATION POLICY</b><span>EDIT</span><ChevronDown /></summary>
        <div className="group-config-grid">
          <label className="wide-field"><span><Pencil /> GROUP NAME</span><input value={name} onChange={(e) => setName(e.target.value)} /></label>
          <label><span>IDENTITY COLOR</span><select value={color} onChange={(e) => setColor(e.target.value)}>
            {!groupPalette.some((candidate) => candidate.hex === color) && <option value={color}>{group.color_name}</option>}
            {groupPalette.map((candidate) => <option value={candidate.hex} key={candidate.name}>{candidate.name[0].toUpperCase() + candidate.name.slice(1)}</option>)}
          </select></label>
          <label><span>MAP PATTERN</span><select value={pattern} onChange={(e) => setPattern(e.target.value)}>
            <option>solid</option><option>diagonal</option><option>dots</option><option>crosshatch</option><option>rings</option><option>chevron</option>
          </select></label>
          <label className="wide-field"><span>IDLE FORMATION</span><select value={formation} onChange={(e) => setFormation(e.target.value)}>
            {formations.map((value) => <option key={value} value={value}>{value.replaceAll("_", " ")}</option>)}
          </select></label>
          <label><span>SPACING · METRES</span><input type="number" min={15} max={1000} step={5} value={spacing} onChange={(e) => setSpacing(Number(e.target.value))} /></label>
          <label><span>HEADING · ° TRUE</span><input type="number" min={0} max={359} step={5} value={heading} onChange={(e) => setHeading(Number(e.target.value))} /></label>
        </div>
        <button className="group-config-save" onClick={() => onSave({ name: name.trim(), color, pattern, formation, formation_spacing_m: spacing, formation_heading_deg: heading })} disabled={!name.trim() || spacing < 15 || spacing > 1000 || heading < 0 || heading >= 360}>
          <Save /> Save station policy
        </button>
      </details>
      <details className="assembly-control">
        <summary>
          <MapPinned />
          <span>
            <b>ASSEMBLY POINT</b>
            <small>
              {group.assembly_point
                ? `${group.assembly_point[1].toFixed(4)}°N · ${Math.abs(group.assembly_point[0]).toFixed(4)}°W`
                : "Not assigned"}
            </small>
          </span>
          <ChevronDown />
        </summary>
        <p>Idle vessels station-keep here using the formation and spacing above.</p>
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
      </details>
      <details className="group-members">
        <summary><Users /><b>MEMBERS</b><span>{members.length}</span><ChevronDown /></summary>
        <div>{members.map((v) => <span key={v.id}><i style={{ background: v.telemetry.health === "nominal" ? "#70b88f" : "#cf6d62" }} />{v.callsign}<small>{v.designation}</small></span>)}</div>
      </details>
      <div className="group-manager-actions">
      <button className="danger" onClick={onDelete}>
        <Trash2 /> Delete group · vessels become unassigned
      </button>
      </div>
    </div>
  );
}
function VesselInspectorWindow({
  pirate,
  vessel,
  lookup,
  onRename,
}: {
  pirate: boolean;
  vessel: VesselProfileV2;
  lookup: Map<string, VesselProfileV2>;
  onRename: (name: string) => void;
}) {
  const [reachability, setReachability] = useState<ReachabilityV2 | null>(null);
  useEffect(() => {
    let active = true;
    setReachability(null);
    void api<ReachabilityV2>(`/api/v2/vessels/${vessel.id}/reachability`)
      .then((value) => { if (active) setReachability(value); })
      .catch(() => { if (active) setReachability(null); });
    return () => { active = false; };
  }, [vessel.id]);
  return <VesselInspector pirate={pirate} vessel={vessel} reachability={reachability} lookup={lookup} onRename={onRename} />;
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
  const reserveValue = Math.max(0, Math.min(100, t.reserve * 100));
  const projectedValue = Math.max(0, Math.min(100, t.projected_reserve * 100));
  const bufferPercent = Math.min(100, Math.round((t.tape_depth_seconds / 60) * 100));
  return (
    <div className="vessel-inspector">
      <div className="vessel-hero">
        <div className="vessel-portrait" style={{ "--vessel-color": vessel.group_color } as CSSProperties}><img src={vesselAsset(vessel.class.id, pirate)} /><i /></div>
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
          <div className="vessel-state-chips"><b className={t.health === "nominal" ? "nominal" : "warning"}><Activity />{t.health}</b><b><Navigation />{t.mode.replaceAll("_", " ")}</b></div>
        </div>
      </div>
      <div className="vessel-readiness">
        <div className="reserve-ring" style={{ "--reserve-angle": `${reserveValue * 3.6}deg`, "--ring-color": reserveValue < 30 ? "#cf6d62" : "#70b88f" } as CSSProperties}>
          <span><BatteryCharging /><strong>{reservePercent(t.reserve)}%</strong><small>ENERGY</small></span>
        </div>
        <div className="readiness-bars">
          <StatusBar icon={<BatteryCharging />} label="PROJECTED MISSION END" value={`${reservePercent(t.projected_reserve)}%`} percent={projectedValue} />
          <StatusBar icon={<Route />} label="HOT EXECUTION BUFFER" value={`${t.tape_depth_seconds}s`} percent={bufferPercent} />
          <StatusBar icon={<Gauge />} label="BATTERY-ONLY RANGE" value={`${(t.reserve * vessel.class.nominal_range_nm).toFixed(1)} nm`} percent={reserveValue} />
        </div>
      </div>
      <div className="hot-buffer-note">
        <Route /><span><b>FULL MISSION PROGRAM</b><small>The route may be arbitrarily long. This vessel keeps the next 60 seconds validated and armed as a rolling resilient buffer.</small></span>
      </div>
      <div className="vessel-nav-grid">
        <Insight icon={<Gauge />} label="SPEED" value={`${t.speed_mps.toFixed(1)} m/s`} detail={`${vessel.class.max_speed_mps.toFixed(1)} max`} />
        <Insight icon={<Compass />} label="HEADING" value={`${Math.round(t.heading_deg)}°`} detail="true" />
        <Insight icon={<Satellite />} label="PNT" value={t.pnt_integrity} detail={`±${t.uncertainty_m.toFixed(0)} m`} tone={t.pnt_integrity === "trusted" ? "good" : "warn"} />
        <Insight icon={<Sun />} label="SOLAR" value={`${vessel.class.solar_peak_kw.toFixed(1)} kW`} detail="peak" />
      </div>
      <h3 className="insight-heading"><Waves /> LOCAL CONDITIONS <span>NOAA-DERIVED FIXTURE</span></h3>
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
      <h3 className="insight-heading"><Network /> REACHABLE SWARM <span>
          {(reachability?.direct_peers.length ?? 0) +
            (reachability?.relayed_peers.length ?? 0)}
        </span></h3>
      <div className="peer-list">
        {reachability?.direct_peers.map((p) => (
          <Peer key={p.vessel_id} p={p} lookup={lookup} />
        ))}
        {reachability?.relayed_peers.map((p) => (
          <Peer key={p.vessel_id} p={p} lookup={lookup} />
        ))}
      </div>
      <div className="authority-note">
        <ShieldCheck /><span><b>Reachability ≠ authority</b>
        <span>{reachability?.authority ?? "Loading scoped authority…"}</span>
        </span>
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
          v={`${contact.speed_mps.toFixed(1)} m/s`}
          sub={`${contact.speed_knots.toFixed(1)} kn · ${contact.navigation_state}`}
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
function StatusBar({ icon, label, value, percent }: { icon: ReactNode; label: string; value: string; percent: number }) {
  return <div className="status-bar"><header>{icon}<span>{label}</span><b>{value}</b></header><i><span style={{ width: `${Math.max(0, Math.min(100, percent))}%` }} /></i></div>;
}
function Insight({ icon, label, value, detail, tone = "" }: { icon: ReactNode; label: string; value: string; detail: string; tone?: string }) {
  return <div className={`insight-tile ${tone}`}>{icon}<span><small>{label}</small><strong>{value}</strong></span><em>{detail}</em></div>;
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
function MissionCanvas({ pirate, mission, groups, plans, activePlan, busy, tool, contactSeed, geometryFocus, onFormation, onLoop, onObjective, onArea, onTool, onGenerateManual, onRefineAI, onOpenConstraints, onApplyContactSeed, onClearContactSeed, onUndoGeometry, onClearGeometry, onDeleteGeometry, onFocusGeometry, onReorderWaypoint, onChoose, onConfirmPlan, onStatus, onRename, onDelete }: {
  pirate: boolean;
  mission: MissionWorkspaceV2 | null;
  groups: FleetSnapshotV2["groups"];
  plans: FleetPlanV2[];
  activePlan: FleetPlanV2 | null;
  preview: FleetPreviewV2 | null;
  lease: FleetLeaseV2 | null;
  busy: boolean;
  tool: Tool;
  contactSeed: SurfaceContactV2 | null;
  geometryFocus: GeometryFocus | null;
  onFormation: (value: string) => void;
  onLoop: (loop: boolean) => void;
  onObjective: (value: string) => void;
  onArea: (kind: "include" | "exclude") => void;
  onTool: (tool: Tool) => void;
  onGenerateManual: (missionType: string, objective: string) => void;
  onRefineAI: (missionType: string, objective: string, instruction: string, alternatives: boolean) => void;
  onOpenConstraints: () => void;
  onApplyContactSeed: (createNew: boolean) => void;
  onClearContactSeed: () => void;
  onUndoGeometry: () => void;
  onClearGeometry: (kind: "include" | "exclude" | "waypoint" | "poi") => void;
  onDeleteGeometry: (focus: GeometryFocus) => void;
  onFocusGeometry: (focus: GeometryFocus) => void;
  onReorderWaypoint: (index: number, direction: -1 | 1) => void;
  onChoose: (id: string) => void;
  onConfirmPlan: (id: string) => void;
  onStatus: (status: "paused" | "executing") => void;
  onRename: (name: string) => void;
  onDelete: () => void;
}) {
  const [expandedPlans, setExpandedPlans] = useState<Set<string>>(new Set());
  const [missionType, setMissionType] = useState("patrol");
  const [manualObjective, setManualObjective] = useState(mission?.objective ?? "");
  const [aiRefineOpen, setAIRefineOpen] = useState(false);
  const [aiInstruction, setAIInstruction] = useState("");
  const [aiAlternatives, setAIAlternatives] = useState(false);
  useEffect(() => { if (contactSeed) setMissionType("follow_contact"); }, [contactSeed]);
  useEffect(() => setManualObjective(mission?.objective ?? ""), [mission?.id, mission?.objective]);
  useEffect(() => {
    setExpandedPlans(new Set());
    setAIRefineOpen(false);
    setAIInstruction("");
    setAIAlternatives(false);
  }, [mission?.id]);
  if (!mission) return <div className="window-empty planner-seed-empty">{contactSeed ? <><Ship /><b>{contactSeed.name}</b><span>{contactSeed.boat_id} · {contactSeed.class} · uncommitted planning context</span><button className="wide amber" onClick={() => onApplyContactSeed(true)} disabled={busy}><Plus /> {pirate ? "Create shadowing voyage" : "Create follow mission"}</button><button className="wide" onClick={onClearContactSeed}>Cancel</button></> : pirate ? "Chart a new voyage from the + tab." : "Create a mission from the + tab."}</div>;

  const coveredTargets = new Set<string>();
  const assignedGroups = groups.filter((group) => {
    const assigned = group.member_ids.length > 0 && group.member_ids.every((id) => mission.target_ids.includes(id));
    if (assigned) group.member_ids.forEach((id) => coveredTargets.add(id));
    return assigned;
  });
  const individualCount = mission.target_ids.filter((id) => !coveredTargets.has(id)).length;
  const scopeParts = [...assignedGroups.map((group) => `${group.code} · ${group.name}`), ...(individualCount ? [`${individualCount} individual${individualCount === 1 ? "" : "s"}`] : [])];

  return <div className="planner mission-editor">
    <div className="planner-layout">
      <section className="planner-chat-pane planner-editor-pane">
        <div className="mission-summary">
          <div className="mission-summary-actions">
            <span>{mission.status}</span>
            <button aria-label={`${mission.status === "paused" ? "Resume" : "Pause"} ${mission.name}`} title={mission.status === "paused" ? "Resume mission" : "Pause mission"} disabled={mission.status !== "executing" && mission.status !== "paused"} onClick={() => onStatus(mission.status === "paused" ? "executing" : "paused")}>{mission.status === "paused" ? <Play /> : <Pause />}</button>
            <button className="delete" aria-label={`Delete ${mission.name}`} title="Delete mission" onClick={onDelete}><Trash2 /></button>
          </div>
          <EditableTitle value={mission.name} label="mission" onSave={onRename} />
          <p>{mission.target_ids.length} {pirate ? "sworn hands" : "assigned assets"} · geometry r{mission.geometry.revision} · {pirate ? "voyage" : "mission"} v{mission.version}</p>
        </div>
        <div className="mission-scope-strip" title="Mission membership mirrors the current Fleet selection.">
          <Ship /><span><small>{pirate ? "VOYAGE CREW" : "MISSION ASSETS"}</small><b>{scopeParts.length ? scopeParts.join(" · ") : pirate ? "Choose ships in Flotilla" : "Select vessels or groups in Fleet"}</b></span><em>{mission.target_ids.length}</em>
          <button type="button" className={mission.loop ? "active" : ""} aria-label={mission.loop ? "Disable mission loop" : "Enable mission loop"} aria-pressed={mission.loop} title={mission.loop ? "Loop enabled" : "Hold at the final marker"} disabled={busy} onClick={() => onLoop(!mission.loop)}><RotateCcw />{mission.loop ? "LOOP" : "HOLD AT END"}</button>
        </div>
        {contactSeed && <div className="planner-contact-seed"><i style={{ background: contactSeed.color }} /><span><b>{contactSeed.name}</b><small>{contactSeed.boat_id} · uncommitted objective</small></span><button onClick={() => onApplyContactSeed(false)}>Use here</button><button onClick={() => onApplyContactSeed(true)}>New mission</button><button aria-label="Dismiss contact planning context" onClick={onClearContactSeed}><X /></button></div>}
        <details className="planner-section objective-section" open>
          <summary>MISSION DEFINITION <em>{missionType.replaceAll("_", " ")}</em></summary>
          <label>MISSION TYPE<select aria-label="MISSION TYPE" value={missionType} onChange={(event) => setMissionType(event.target.value)}><option value="transit">Transit</option><option value="patrol">Patrol</option><option value="search">Search</option><option value="follow_contact">Follow contact</option><option value="hold">Hold</option><option value="orbit">Orbit</option><option value="custom_route">Custom route</option></select></label>
          <label>OBJECTIVE<textarea aria-label="OBJECTIVE" value={manualObjective} onChange={(event) => setManualObjective(event.target.value)} placeholder="Describe the mission outcome." /></label>
          {mission.target_ids.length > 1 ? <label className="formation-control">{pirate ? "SAILING FORMATION" : "FORMATION"}<select value={mission.formation} onChange={(event) => onFormation(event.target.value)}>{formations.map((formation) => <option value={formation} key={formation}>{formation.replaceAll("_", " ")}</option>)}</select></label> : mission.target_ids.length === 1 ? <div className="solo-mode"><Ship /><span><b>INDEPENDENT VESSEL</b><small>Formation controls do not apply.</small></span></div> : null}
        </details>
        <details className="planner-section map-authoring">
          <summary>MAP &amp; ROUTE AUTHORING <em>{mission.geometry.waypoints.length + mission.geometry.pois.length} markers</em></summary>
          <div className="authoring-status"><b>{mission.name.toUpperCase()}</b><span>{tool === "select" ? "READY · SELECT OR EDIT" : `${tool.replaceAll("_", " ").toUpperCase()} ACTIVE · ESC TO CANCEL`}</span></div>
          <div className="geometry-actions">
            <button aria-label="Select or edit mission geometry" className={tool === "select" ? "active" : ""} onClick={() => onTool("select")} title="Select or drag mission geometry"><MousePointer2 /></button>
            <button aria-label="Assign vessels by rectangle" className={tool === "box" ? "active" : ""} onClick={() => onTool("box")} title="Assign vessels inside a rectangle"><BoxSelect /></button>
            <button aria-label="Add operating area" className={tool === "include" ? "active" : ""} onClick={() => onArea("include")} title="Draw an allowed operating area"><Plus /><BoxSelect /></button>
            <button aria-label="Add exclusion area" className={tool === "exclude" ? "active" : ""} onClick={() => onArea("exclude")} title="Draw an exclusion area"><Ban /></button>
            <button aria-label="Add waypoint" className={tool === "waypoint" ? "active" : ""} onClick={() => onTool("waypoint")} title="Add the next waypoint"><MapPinned /></button>
            <button aria-label="Add hold point" className={tool === "hold" ? "active" : ""} onClick={() => onTool("hold")} title="Add a hold point"><CircleDot /></button>
            <button aria-label="Add orbit point" className={tool === "orbit" ? "active" : ""} onClick={() => onTool("orbit")} title="Add an orbit point"><RotateCcw /></button>
            <button aria-label="Undo mission geometry change" onClick={onUndoGeometry} title="Undo the latest geometry change"><Undo2 /></button>
          </div>
          <div className="geometry-summary"><span>{mission.geometry.included_areas.length} operating</span><span>{mission.geometry.exclusion_areas.length} excluded</span><span>{mission.geometry.waypoints.length} waypoints</span><span>{mission.geometry.pois.length} hold/orbit</span></div>
          <div className="geometry-inventory">
            {mission.geometry.included_areas.map((_, index) => <button className={geometryFocus?.kind === "include" && geometryFocus.index === index ? "selected" : ""} key={`include-${index}`} onClick={() => onFocusGeometry({ kind: "include", index })}><span>Operating area {index + 1}</span><Eye /></button>)}
            {mission.geometry.exclusion_areas.map((_, index) => <button className={geometryFocus?.kind === "exclude" && geometryFocus.index === index ? "selected" : ""} key={`exclude-${index}`} onClick={() => onFocusGeometry({ kind: "exclude", index })}><span>Exclusion area {index + 1}</span><Eye /></button>)}
            {mission.geometry.waypoints.map((_, index) => <div className={geometryFocus?.kind === "waypoint" && geometryFocus.index === index ? "selected" : ""} key={`waypoint-${index}`}><button onClick={() => onFocusGeometry({ kind: "waypoint", index })}><span>Waypoint {index + 1}</span><Eye /></button><button disabled={index === 0} onClick={() => onReorderWaypoint(index, -1)}><ChevronUp /></button><button disabled={index === mission.geometry.waypoints.length - 1} onClick={() => onReorderWaypoint(index, 1)}><ChevronDown /></button></div>)}
            {mission.geometry.pois.map((poi, index) => <button className={geometryFocus?.kind === "poi" && geometryFocus.index === index ? "selected" : ""} key={poi.id} onClick={() => onFocusGeometry({ kind: "poi", index })}><span>{poi.kind === "orbit" ? "Orbit" : "Hold"} point {index + 1}</span><Eye /></button>)}
            {geometryFocus && <button className="geometry-delete" onClick={() => onDeleteGeometry(geometryFocus)}><Trash2 />Delete selected</button>}
            <div className="geometry-clear-actions"><button onClick={() => onClearGeometry("include")}>Clear operating</button><button onClick={() => onClearGeometry("exclude")}>Clear exclusions</button><button onClick={() => onClearGeometry("waypoint")}>Clear waypoints</button><button onClick={() => onClearGeometry("poi")}>Clear hold/orbit</button></div>
          </div>
        </details>
        <div className="mission-build-actions">
          <button className="wide" onClick={onOpenConstraints} disabled={busy}><SlidersHorizontal /> Constraints</button>
          <button className="wide" onClick={() => onObjective(manualObjective.trim())} disabled={busy || !manualObjective.trim() || manualObjective.trim() === mission.objective}><Save /> Save details</button>
          <button className="wide amber" onClick={() => onGenerateManual(missionType, manualObjective)} disabled={busy || mission.target_ids.length === 0}><Route />{plans.length ? "Rebuild route" : "Build route"}</button>
          {mission.target_ids.length === 0 && <small>Select vessels or groups in Fleet before building a route.</small>}
        </div>
      </section>
      <section className="planner-options-pane planner-route-pane">
        <header className="route-workbench-header"><span><b>ROUTE &amp; EXECUTION</b><small>{plans.length > 1 ? `${plans.length} alternatives · select to preview on map` : plans.length === 1 ? "1 validated route" : "Build a route from the mission definition"}</small></span>{activePlan && <em>{activePlan.advisor_source === "deterministic" ? "MANUAL" : "AI REFINED"}</em>}</header>
        {mission.trajectory && <div className="trajectory-program-summary"><header><Route /><b>ACTIVE PROGRAM · REVISION {mission.trajectory.active_revision}</b>{mission.trajectory.pending_revision && <em>R{mission.trajectory.pending_revision} ARMED · T+{mission.trajectory.activation_tick}</em>}</header><dl><span><small>PROGRAM</small>{Math.ceil(mission.trajectory.duration_seconds / 60)} min</span><span><small>SEGMENTS</small>{mission.trajectory.total_segments}</span><span><small>BUFFER</small>{mission.trajectory.hot_tape_horizon_seconds}s</span><span><small>CURSOR</small>T+{mission.trajectory.mission_tick}s</span></dl></div>}
        {plans.length === 0 ? <div className="route-workbench-empty"><Route /><b>No route built yet</b><span>Build locally from the mission definition, or optionally ask AI to refine it.</span></div> : <div className="candidate-list mission-route-choices" aria-label="Mission route options">{plans.slice(0, 3).map((plan, index) => {
          const expanded = expandedPlans.has(plan.id), label = String.fromCharCode(65 + index);
          return <article key={plan.id} className={`${activePlan?.id === plan.id ? "selected" : ""} ${expanded ? "expanded" : "collapsed"} ${plan.policy_status}`}><header><button className="candidate-select" aria-label={`Preview option ${label}: ${plan.name}`} aria-pressed={activePlan?.id === plan.id} disabled={busy || plan.policy_status === "prohibited"} onClick={() => onChoose(plan.id)}><span className="option-letter">{label}</span><b>{plan.name}</b></button>{plan.recommended && <em>{pirate ? "CAPTAIN'S PICK" : "RECOMMENDED"}</em>}<button className="candidate-expand" aria-label={`${expanded ? "Collapse" : "Expand"} option ${label}`} onClick={() => setExpandedPlans((current) => { const next = new Set(current); next.has(plan.id) ? next.delete(plan.id) : next.add(plan.id); return next; })}>{expanded ? <ChevronUp /> : <ChevronDown />}</button></header><div className="candidate-quick-metrics"><span>{plan.duration_minutes.toFixed(0)} min</span><span>{reservePercent(plan.minimum_reserve)}% reserve</span><span>{plan.minimum_separation_m} m sep</span></div><div className="candidate-detail"><p>{plan.description}</p><small>{plan.maneuvers.join(" → ")}</small><code>{plan.content_hash.slice(0, 18)}…</code></div></article>;
        })}</div>}
        {activePlan && <button className="mission-start-action" disabled={busy || activePlan.policy_status === "prohibited"} onClick={() => onConfirmPlan(activePlan.id)}><ShieldCheck />Review and start selected route</button>}
        <details className="ai-refine-panel" open={aiRefineOpen} onToggle={(event) => setAIRefineOpen(event.currentTarget.open)}>
          <summary><Sparkles /><span><b>Refine with AI</b><small>Optional assistance for this existing mission</small></span><em>{aiRefineOpen ? "CLOSE" : "OPEN"}</em></summary>
          <div><label>ADDITIONAL INSTRUCTION<textarea aria-label="AI refinement instruction" value={aiInstruction} onChange={(event) => setAIInstruction(event.target.value)} placeholder="Example: reduce shallow-water exposure and preserve more reserve." /></label><label className="ai-alternatives-toggle"><input type="checkbox" checked={aiAlternatives} onChange={(event) => setAIAlternatives(event.target.checked)} /><span><b>Offer alternatives</b><small>Return three routes to compare instead of one recommendation.</small></span></label><button className="wide" disabled={busy || mission.target_ids.length === 0} onClick={() => onRefineAI(missionType, manualObjective, aiInstruction, aiAlternatives)}><Sparkles />{aiAlternatives ? "Generate AI alternatives" : "Apply AI refinement"}</button>{busy && <div className="agent-work-chips"><span><Sparkles /> Reviewing mission</span><span>Checking constraints</span><span>Validating route</span></div>}</div>
        </details>
      </section>
    </div>
  </div>;
}

function LegacyMissionCanvas({
  pirate,
  mission,
  groups,
  command,
  setCommand,
  plans,
  activePlan,
  busy,
  recording,
  tool,
  contactSeed,
  geometryFocus,
  onSpeak,
  onTranscriptionStart,
  onTranscriptionStop,
  onFormation,
  onLoop,
  onObjective,
  onArea,
  onTool,
  onCreate,
  onApplyContactSeed,
  onClearContactSeed,
  onUndoGeometry,
  onClearGeometry,
  onDeleteGeometry,
  onFocusGeometry,
  onReorderWaypoint,
  onChoose,
  onStatus,
  onRename,
  onDelete,
  scene,
}: {
  pirate: boolean;
  mission: MissionWorkspaceV2 | null;
  groups: FleetSnapshotV2["groups"];
  command: string;
  setCommand: (v: string) => void;
  plans: FleetPlanV2[];
  activePlan: FleetPlanV2 | null;
  preview: FleetPreviewV2 | null;
  lease: FleetLeaseV2 | null;
  busy: boolean;
  recording: boolean;
  tool: Tool;
  contactSeed: SurfaceContactV2 | null;
  geometryFocus: GeometryFocus | null;
  onSpeak: (text: string) => void;
  onTranscriptionStart: () => void;
  onTranscriptionStop: () => void;
  onFormation: (v: string) => void;
  onLoop: (loop: boolean) => void;
  onObjective: (v: string) => void;
  onArea: (k: "include" | "exclude") => void;
  onTool: (t: Tool) => void;
  onCreate: (intent: string) => void;
  onApplyContactSeed: (createNew: boolean) => void;
  onClearContactSeed: () => void;
  onUndoGeometry: () => void;
  onClearGeometry: (kind: "include" | "exclude" | "waypoint" | "poi") => void;
  onDeleteGeometry: (focus: GeometryFocus) => void;
  onFocusGeometry: (focus: GeometryFocus) => void;
  onReorderWaypoint: (index: number, direction: -1 | 1) => void;
  onChoose: (id: string) => void;
  onStatus: (status: "paused" | "executing") => void;
  onRename: (name: string) => void;
  onDelete: () => void;
  scene: CommandSceneV1 | null;
}) {
  const chatEnd = useRef<HTMLDivElement | null>(null);
  const [expandedPlans, setExpandedPlans] = useState<Set<string>>(new Set());
  const [missionType, setMissionType] = useState("patrol");
  const [manualObjective, setManualObjective] = useState(mission?.objective ?? "");
  const [historyOpen, setHistoryOpen] = useState(false);
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
  const coveredTargets = new Set<string>(),
    assignedGroups = groups.filter((group) => {
      const assigned = group.member_ids.length > 0 && group.member_ids.every((id) => mission.target_ids.includes(id));
      if (assigned) group.member_ids.forEach((id) => coveredTargets.add(id));
      return assigned;
    }),
    individualCount = mission.target_ids.filter((id) => !coveredTargets.has(id)).length,
    scopeParts = [
      ...assignedGroups.map((group) => `${group.code} · ${group.name}`),
      ...(individualCount > 0 ? [`${individualCount} individual${individualCount === 1 ? "" : "s"}`] : []),
    ];
  return (
    <div className="planner">
      <div className="planner-layout">
      <section className="planner-chat-pane">
      {scene && <div className="mission-scene-surface"><KeelMeshA2UISurface surface={scene.primary_surface} /></div>}
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
      <div className="mission-scope-strip" title="Mission membership mirrors the current selection in Fleet. Changing that selection safely returns an active mission to draft for a new plan and authorization.">
        <Ship />
        <span>
          <small>{pirate ? "VOYAGE CREW" : "MISSION ASSETS"}</small>
          <b>{scopeParts.length > 0 ? scopeParts.join(" · ") : pirate ? "Choose ships in Flotilla" : "Select vessels or groups in Fleet"}</b>
        </span>
        <em>{mission.target_ids.length}</em>
        <button
          type="button"
          className={mission.loop ? "active" : ""}
          aria-label={mission.loop ? "Disable mission loop" : "Enable mission loop"}
          aria-pressed={mission.loop}
          title={mission.loop ? "Loop enabled: after the final marker, return to the first and repeat." : "Loop disabled: finish at the final marker and hold position."}
          disabled={busy}
          onClick={() => onLoop(!mission.loop)}
        >
          <RotateCcw />
          {mission.loop ? "LOOP" : "HOLD AT END"}
        </button>
      </div>
      <div className="mission-canvas-command-row">
        <b>{pirate ? "SHIP'S INTELLIGENCE" : "AI COMMAND"}</b>
        <button type="button" onClick={() => setHistoryOpen((value) => !value)} aria-expanded={historyOpen}><History /> {historyOpen ? "Hide history" : "History"}</button>
      </div>
      {historyOpen && <div className="mission-chat" aria-live="polite">
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
        <div ref={chatEnd} />
      </div>}
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
      {plans.length > 0 && (
        <div className="candidate-list ai-plan-choices" aria-label="AI strategy options">
          {plans.slice(0, 3).map((p, index) => {
            const expanded = expandedPlans.has(p.id);
            const label = String.fromCharCode(65 + index);
            return (
              <article key={p.id} className={`${activePlan?.id === p.id ? "selected" : ""} ${expanded ? "expanded" : "collapsed"} ${p.policy_status}`}>
                <header>
                  <button className="candidate-select" disabled={busy || p.policy_status === "prohibited"} onClick={() => onChoose(p.id)}>
                    <span className="option-letter">{label}</span>
                    <b>{p.name}</b>
                  </button>
                  {p.recommended && <em>{pirate ? "CAPTAIN'S PICK" : "RECOMMENDED"}</em>}
                  <button className="candidate-expand" aria-label={`${expanded ? "Collapse" : "Expand"} option ${label}`} onClick={() => setExpandedPlans((current) => { const next = new Set(current); next.has(p.id) ? next.delete(p.id) : next.add(p.id); return next; })}>
                    {expanded ? <ChevronUp /> : <ChevronDown />}
                  </button>
                </header>
                <div className="candidate-quick-metrics">
                  <span>{p.duration_minutes.toFixed(0)} min</span>
                  <span>{Math.round(p.minimum_reserve * 100)}% reserve</span>
                  <span>{p.minimum_separation_m} m sep</span>
                </div>
                <div className="candidate-detail">
                  <p>{p.description}</p>
                  <small>{p.maneuvers.join(" → ")}</small>
                  <code>{p.content_hash.slice(0, 18)}…</code>
                </div>
              </article>
            );
          })}
        </div>
      )}
      </section>
      <section className="planner-options-pane">
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
      {contactSeed && (
        <div className="planner-contact-seed">
          <i style={{ background: contactSeed.color }} />
          <span><b>{contactSeed.name}</b><small>{contactSeed.boat_id} · uncommitted objective</small></span>
          <button onClick={() => onApplyContactSeed(false)}>Use in this mission</button>
          <button onClick={() => onApplyContactSeed(true)}>Create new mission</button>
          <button aria-label="Dismiss contact planning context" onClick={onClearContactSeed}><X /></button>
        </div>
      )}
      <details className="planner-section objective-section">
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
        {mission.target_ids.length > 1 ? (
          <label className="formation-control">
            {pirate ? "SAILING FORMATION" : "FORMATION PREFERENCE"}
            <select value={mission.formation} onChange={(e) => onFormation(e.target.value)}>
              {formations.map((formation) => <option value={formation} key={formation}>{formation.replaceAll("_", " ")}</option>)}
            </select>
          </label>
        ) : mission.target_ids.length === 1 ? (
          <div className="solo-mode"><Ship /><span><b>INDEPENDENT VESSEL</b><small>Formation controls are not applied.</small></span></div>
        ) : null}
      </details>
      <details className="planner-section map-authoring">
        <summary>MAP AUTHORING</summary>
        <div className="authoring-status">
          <b>{mission.name.toUpperCase()}</b>
          <span>{tool === "select" ? "READY · SELECT OR EDIT" : `${tool.replaceAll("_", " ").toUpperCase()} TOOL ACTIVE · ESC TO CANCEL`}</span>
        </div>
        <div className="geometry-actions">
          <button aria-label="Select or edit mission geometry" className={tool === "select" ? "active" : ""} onClick={() => onTool("select")} title="Select or drag existing mission geometry"><MousePointer2 /></button>
          <button aria-label="Assign vessels by rectangle" className={tool === "box" ? "active" : ""} onClick={() => onTool("box")} title="Assign every controlled vessel inside a rectangle"><BoxSelect /></button>
          <button aria-label="Add operating area" className={tool === "include" ? "active" : ""} onClick={() => onArea("include")} title="Draw an allowed operating area"><Plus /><BoxSelect /></button>
          <button aria-label="Add exclusion area" className={tool === "exclude" ? "active" : ""} onClick={() => onArea("exclude")} title="Draw an area this mission must avoid"><Ban /></button>
          <button aria-label="Add waypoint" className={tool === "waypoint" ? "active" : ""} onClick={() => onTool("waypoint")} title="Add the next numbered route waypoint"><MapPinned /></button>
          <button aria-label="Add hold point" className={tool === "hold" ? "active" : ""} onClick={() => onTool("hold")} title="Add a position for selected vessels to hold"><CircleDot /></button>
          <button aria-label="Add orbit point" className={tool === "orbit" ? "active" : ""} onClick={() => onTool("orbit")} title="Add a point for selected vessels to orbit"><RotateCcw /></button>
          <button aria-label="Undo mission geometry change" onClick={onUndoGeometry} title="Undo the most recent geometry mutation"><Undo2 /></button>
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
      </section>
      </div>
    </div>
  );
}

function SceneArtifact({ scene, onAction, onPin, onDismiss }: { scene: CommandSceneV1; onAction: (action: CommandSceneV1["suggested_actions"][number]) => void; onPin: () => void; onDismiss: () => void }) {
  return <div className={`scene-artifact ${scene.critical ? "critical" : ""}`}>
    <div className="scene-state-strip"><span><Sparkles /> LIVE COMMAND SCENE</span><em>{scene.catalog_id}</em></div>
    <KeelMeshA2UISurface surface={scene.primary_surface} />
    <div className="scene-action-row">
      {scene.suggested_actions.map((action) => <button key={action.id} className={action.authority_class === "effect" ? "amber" : ""} onClick={() => onAction(action)}>{action.label}</button>)}
    </div>
    <footer><span>{scene.bindings.length} live binding{scene.bindings.length === 1 ? "" : "s"}</span><span>{scene.receipts.length} receipt{scene.receipts.length === 1 ? "" : "s"}</span><button onClick={onPin} aria-pressed={scene.pinned}><Pin />{scene.pinned ? "Unpin" : "Keep"}</button><button onClick={onDismiss}><X />Dismiss</button></footer>
  </div>;
}

function SceneHistory({ scenes, onOpen }: { scenes: CommandSceneV1[]; onOpen: (scene: CommandSceneV1) => void }) {
  return <div className="scene-history"><header><History /><span><b>COMMAND HISTORY</b><small>Provider turns, trusted surfaces, bindings, and receipts</small></span></header>{scenes.length === 0 ? <p>No command scenes have been composed in this session.</p> : scenes.map((scene) => <button key={scene.id} onClick={() => onOpen(scene)}><Sparkles /><span><b>{scene.title}</b><small>{scene.summary}</small></span><em>{scene.pinned ? "PINNED" : scene.state}</em></button>)}</div>;
}

function AssistantChat({ turns, value, busy, pirate, onChange, onSend, scenes, onOpenScene }: {
  turns: ConversationTurnV1[];
  value: string;
  busy: boolean;
  pirate: boolean;
  onChange: (value: string) => void;
  onSend: (value: string) => void;
  scenes: CommandSceneV1[];
  onOpenScene: (scene: CommandSceneV1) => void;
}) {
  const end = useRef<HTMLDivElement | null>(null);
  useEffect(() => {
    end.current?.scrollIntoView({ block: "end" });
  }, [turns.length, busy]);
  const recentScenes = scenes.filter((scene) => scene.state === "active" || scene.pinned).slice(0, 3);
  return <div className="assistant-chat-window">
    <header>
      <MessageCircle />
      <span><b>{pirate ? "SHIP'S INTELLIGENCE" : "TEXT CHANNEL"}</b><small>Shared memory with the primary voice assistant · text replies only</small></span>
    </header>
    <div className="assistant-chat-transcript" aria-live="polite">
      {turns.length === 0 && <div className="assistant-chat-empty"><Sparkles /><b>Ask KeelMesh anything</b><span>Type here when voice is not convenient. Mission and workspace requests use the same bounded tools and approvals.</span></div>}
      {turns.map((turn) => <article key={turn.id} className={turn.role === "assistant" ? "assistant" : "operator"}>
        <header><b>{turn.role === "assistant" ? "KEELMESH AI" : "YOU"}</b><time>{new Date(turn.created_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}</time></header>
        <ReactMarkdown remarkPlugins={[remarkGfm]}>{turn.content}</ReactMarkdown>
      </article>)}
      {busy && <div className="assistant-chat-working"><Sparkles /><span>Gathering context</span><span>Composing response</span></div>}
      <div ref={end} />
    </div>
    {recentScenes.length > 0 && <details className="assistant-chat-artifacts"><summary>Recent command artifacts</summary>{recentScenes.map((scene) => <button key={scene.id} onClick={() => onOpenScene(scene)}><Sparkles /><span>{scene.title}</span><em>{scene.pinned ? "PINNED" : scene.state}</em></button>)}</details>}
    <form onSubmit={(event) => { event.preventDefault(); if (value.trim() && !busy) onSend(value.trim()); }}>
      <textarea aria-label="Message KeelMesh AI" placeholder={pirate ? "Type an order or question…" : "Type a question or command…"} value={value} onChange={(event) => onChange(event.target.value)} onKeyDown={(event) => { if (event.key === "Enter" && !event.shiftKey) { event.preventDefault(); if (value.trim() && !busy) onSend(value.trim()); } }} />
      <button type="submit" aria-label="Send text message" disabled={!value.trim() || busy}><Send /></button>
    </form>
  </div>;
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
