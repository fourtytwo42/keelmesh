export type GuidedDemoFocus =
  | "map"
  | "fleet"
  | "assistant"
  | "mission"
  | "resilience"
  | "system-network"
  | "system-data"
  | "ai-lab"
  | "memory"
  | "closing";

export type GuidedDemoAction =
  | "reset"
  | "inspect-vessel"
  | "create-group"
  | "create-ai-mission"
  | "review-plan"
  | "execute-plan"
  | "author-manual-mission"
  | "run-resilience"
  | "show-consensus"
  | "show-data-plane"
  | "show-ai-lab"
  | "show-memory"
  | "finish";

export type GuidedDemoBeat = {
  id: string;
  title: string;
  focus: GuidedDemoFocus;
  action: GuidedDemoAction;
  audio: { navy: string; pirate: string };
  transcript: { navy: string; pirate: string };
};

const asset = (persona: "navy" | "pirate", id: string) =>
  `/assets/demo/${persona}/${id}.mp3`;

export const guidedDemoBeats: GuidedDemoBeat[] = [
  {
    id: "01-orientation", title: "Failure-first fleet operations", focus: "map", action: "reset",
    audio: { navy: asset("navy", "01-orientation"), pirate: asset("pirate", "01-orientation") },
    transcript: {
      navy: "Welcome to KeelMesh: a failure-first autonomy and agent infrastructure demonstration. These twelve vessels are twelve Ubuntu virtual machines distributed across three Proxmox hosts. The map is a deterministic Rhode Island maritime simulation, and the interface you are watching is live. This tour is hands-off: every view, command, mission artifact, and failure drill is being driven through the same typed application interfaces available to an operator or external agent.",
      pirate: "Welcome aboard KeelMesh, ye bold builders. These twelve ships be twelve Ubuntu vessels spread across three Proxmox islands, with a deterministic Rhode Island sea beneath their keels. This voyage runs hands-free. Every chart, order, window, and calamity comes through the very same typed interfaces used by captain and agent alike. We built for storms and mutiny, not merely calm water and a flattering screenshot.",
    },
  },
  {
    id: "02-vessel-intelligence", title: "Ask the fleet, not a static dashboard", focus: "assistant", action: "inspect-vessel",
    audio: { navy: asset("navy", "02-vessel-intelligence"), pirate: asset("pirate", "02-vessel-intelligence") },
    transcript: {
      navy: "The assistant receives live, authorized fleet context rather than a screenshot. I am asking for one vessel by name. It resolves the persistent identity, centers the map, and opens current position, energy, environment, PNT confidence, connectivity, and mission authority. The answer is retained in the shared conversation ledger, so follow-up voice or text can refer to the same vessel without starting over.",
      pirate: "Ask for a ship by name and the ship's intelligence finds her in the living fleet ledger, swings the chart to her bow, and opens her present position, stores, weather, navigation confidence, signal paths, and orders. This is no painted dashboard. The answer joins our shared log, so the next command may say that ship or her without the oracle suddenly forgetting which hull we meant.",
    },
  },
  {
    id: "03-group", title: "AI-operated fleet organization", focus: "fleet", action: "create-group",
    audio: { navy: asset("navy", "03-group"), pirate: asset("pirate", "03-group") },
    transcript: {
      navy: "Now the assistant creates an operational group from three named vessels. The change is visible immediately in Fleet and uses the same versioned group mutation as manual drag and drop. Group identity, membership, formation policy, spacing, color, and assembly behavior are durable state. The model proposes semantic actions; deterministic Go services validate identity, versions, scope, and policy before anything changes.",
      pirate: "Now the oracle musters three named ships into a proper crew. Fleet changes before our eyes through the same versioned machinery as a sailor dragging names by hand. Crew, formation, spacing, colors, and rally point endure beyond the moment. The talking model may propose; the Go quartermaster checks identity, version, scope, and standing orders before a single soul changes berth.",
    },
  },
  {
    id: "04-ai-mission", title: "Natural language to an exact plan", focus: "mission", action: "create-ai-mission",
    audio: { navy: asset("navy", "04-ai-mission"), pirate: asset("pirate", "04-ai-mission") },
    transcript: {
      navy: "A plain-language patrol request now becomes a typed mission draft. The language model resolves assets and intent, but it does not draw arbitrary commands or authorize movement. Deterministic planners expand the group, derive route geometry from the live chart, enforce depth, separation, speed, energy, PNT, and communications constraints, then produce a content-addressed plan for review.",
      pirate: "A spoken patrol order now becomes a typed voyage, not a bag of mysterious prose. The oracle resolves crew and intent, yet cannot secretly seize the helm. Deterministic navigators expand the crew, read the chart, test depth, spacing, speed, stores, navigation confidence, and signal limits, then seal one exact course by hash for the captain's review.",
    },
  },
  {
    id: "05-approval", title: "Human authority remains explicit", focus: "mission", action: "review-plan",
    audio: { navy: asset("navy", "05-approval"), pirate: asset("pirate", "05-approval") },
    transcript: {
      navy: "This is the authority boundary. The candidate shows affected assets, formation, duration, reserve, separation, and the exact plan hash. In ordinary operation a person confirms this card by touch or voice. For this prerecorded tour, the next scripted step supplies that same explicit confirmation. The assistant cannot approve its own work, loosen policy, or bypass the lease.",
      pirate: "Here lies the captain's line in the sand. The card names every ship, formation, duration, reserve, spacing, and the exact wax seal of the plan. A living captain normally confirms by voice or touch. On this recorded voyage, the next scripted beat gives that same deliberate aye. The oracle may never approve its own scheme or slip past the lease in darkness.",
    },
  },
  {
    id: "06-execution", title: "Committed work, accelerated simulation", focus: "map", action: "execute-plan",
    audio: { navy: asset("navy", "06-execution"), pirate: asset("pirate", "06-execution") },
    transcript: {
      navy: "The approved mission is now executing at one hundred times simulation speed. Routes are trajectory programs with timed intermediate states, not a sixty-second demo tape. Boats consume completed path segments like a queue, maintain formation, respond to wind and current, and spend or recover energy under a day-night solar model. Passed route and waypoint graphics disappear while immutable receipts remain available for replay.",
      pirate: "The sealed voyage is underway at one hundred tides of time. These courses be long trajectory programs with timed waymarks, not a tiny sixty-second trick. Ships consume the path ahead, keep station, answer wind and current, and spend or harvest power beneath sun and night. Old lines vanish from the chart like crumbs before gulls, while the immutable ship's log keeps every proof.",
    },
  },
  {
    id: "07-manual", title: "Manual planning has full parity", focus: "mission", action: "author-manual-mission",
    audio: { navy: asset("navy", "07-manual"), pirate: asset("pirate", "07-manual") },
    transcript: {
      navy: "AI is first class, but it is optional. This second draft demonstrates the manual path: select assets in Fleet, choose a mission type, draw an operating area and exclusion, place and reorder waypoints, add a hold, set formation and constraints, then save or build routes. Manual and AI-assisted missions converge on the same planner, policy checks, hash, quorum-backed approval, and execution interfaces.",
      pirate: "The oracle sits beside the captain, never atop him. This second draft shows the hand-charted path: muster ships in Fleet, choose the voyage, mark allowed water and forbidden shoals, drop numbered waymarks and a holding point, then set formation and standing orders. Hand and oracle both arrive at the same navigator, policy gate, hash, quorum, and execution machinery.",
    },
  },
  {
    id: "08-resilience", title: "Connectivity and PNT degrade safely", focus: "resilience", action: "run-resilience",
    audio: { navy: asset("navy", "08-resilience"), pirate: asset("pirate", "08-resilience") },
    transcript: {
      navy: "KeelMesh is built around real failure conditions. This repeatable drill removes simulated Starlink mission traffic, routes through the simulated HaLow mesh, isolates a vessel, injects an impossible GNSS jump, and restores contact. Management and model access stay out of band. The vessel rejects suspicious GPS, lowers PNT confidence, consumes only cached approved work, then enters safe behavior. Rejoin discards expired commands instead of replaying a dangerous backlog.",
      pirate: "Now for foul weather. We cut the simulated Starlink road, pass orders through the simulated HaLow fleet mesh, maroon a ship, feed her a lying GNSS jump, and restore contact. Management and the oracle's road remain untouched. The vessel calls the false position cursed, lowers trust, spends only cached approved orders, and turns safe. On reunion she burns stale commands rather than obeying ghosts from yesterday.",
    },
  },
  {
    id: "09-consensus", title: "Two real six-voter authority cells", focus: "system-network", action: "show-consensus",
    audio: { navy: asset("navy", "09-consensus"), pirate: asset("pirate", "09-consensus") },
    transcript: {
      navy: "System exposes the real node fabric. The twelve VMs form two independent six-voter Hashicorp Raft cells, distributed two-two-two across the three hosts. Four signatures are required for authority. Raft uses only the isolated radio-plane IP network; management and inference use a separate plane. Ed25519 mTLS verifies node, cell, and address. A minority partition cannot elect authority, and cross-cell work requires matching future-activation certificates.",
      pirate: "Below deck stand twelve real VM ships in two six-vote councils, spread two-two-two across three Proxmox islands. Four signed hands are required to command. Raft sails only the isolated radio-plane network while management and inference keep a separate passage. Ed25519 mTLS checks ship, crew, and address. A mutinous minority gains no crown, and cross-cell orders need matching future-activation charters.",
    },
  },
  {
    id: "10-data", title: "Recoverable streaming and deterministic replay", focus: "system-data", action: "show-data-plane",
    audio: { navy: asset("navy", "10-data"), pirate: asset("pirate", "10-data") },
    transcript: {
      navy: "The data plane separates telemetry ingestion from processing. Kafka KRaft buffers partitioned events; supervised workers validate and project them into PostgreSQL and pgvector with idempotent writes, quarantine, and redrive. If a worker dies, another takes the partitions, drains lag, and produces the same projection without duplicate effects. Shadow replay from retained offsets compares counts and checksums, turning a resilience claim into inspectable evidence.",
      pirate: "Telemetry enters Kafka's partitioned hold before supervised workers inspect and stow it in PostgreSQL and pgvector. Slay a worker and another claims its cargo, drains the lag, and writes no duplicate treasure. Poison records go to quarantine for repair and redrive. A shadow replay begins from retained offsets and compares counts and seals, proving recovery with evidence instead of a captain's tall tale.",
    },
  },
  {
    id: "11-ai-lab", title: "Bounded agents, MCP, RAG, and evaluations", focus: "ai-lab", action: "show-ai-lab",
    audio: { navy: asset("navy", "11-ai-lab"), pirate: asset("pirate", "11-ai-lab") },
    transcript: {
      navy: "AI Lab makes the agent path inspectable. A private, capability-scoped MCP boundary exposes typed tools rather than arbitrary browser control. Retrieval combines authorized conversation, entities, missions, incidents, and runbooks. Tool receipts, citations, provider attempts, latency, replay checksums, prompt-injection defenses, and regression evaluations remain visible. OpenAI can fail and deterministic mission authority still operates; model output is advice, never the source of truth.",
      pirate: "The Shipwright's lab shows every gear in the oracle. A private capability-scoped MCP hatch exposes typed tools, never a skeleton key to the browser. Authorized conversations, vessels, voyages, incidents, and runbooks feed retrieval. Receipts, citations, provider attempts, timing, replay seals, prompt-injection defenses, and evaluations stay in plain sight. Cut the cloud oracle and deterministic authority still steers; eloquent words are counsel, never law.",
    },
  },
  {
    id: "12-memory", title: "Distributed, scoped operational memory", focus: "memory", action: "show-memory",
    audio: { navy: asset("navy", "12-memory"), pirate: asset("pirate", "12-memory") },
    transcript: {
      navy: "The memory plane preserves conversation continuity and operational learning without turning remembered text into authority. PostgreSQL and pgvector hold scoped memories, sources, revisions, contradictions, tombstones, retrieval receipts, and entity relationships. Kafka carries candidates and invalidations. Nodes retain authorized local context for disconnection and reconcile by signed watermarks. Private learning may be automatic; faction or global promotion requires human approval.",
      pirate: "Memory keeps the oracle from waking anew after every sentence, but no remembered rumor becomes an order. PostgreSQL and pgvector hold scoped memories, sources, revisions, contradictions, tombstones, receipts, and relationships. Kafka carries candidates and invalidations. Each ship keeps only authorized local lore for isolation, then reconciles by signed watermark. Private lessons may settle automatically; fleetwide doctrine still demands a human aye.",
    },
  },
  {
    id: "13-closing", title: "Evidence over theater", focus: "closing", action: "finish",
    audio: { navy: asset("navy", "13-closing"), pirate: asset("pirate", "13-closing") },
    transcript: {
      navy: "The working boundary is explicit: twelve nodes, real Raft replication, mTLS identities, quorum receipts, mission authority, telemetry recovery, memory, agent tools, and deterministic simulations are implemented. Starlink, HaLow radio behavior, GNSS faults, and the maritime world are simulated; physical radios, hardware-rooted keys, sea trials, and validation at thirty to one hundred vessels are the next phase. KeelMesh does not merely demonstrate autonomy when everything works. It provides evidence that authority remains bounded when everything does not.",
      pirate: "Here be the honest edge of the chart. Twelve nodes, real Raft replication, mTLS identities, quorum receipts, mission authority, streaming recovery, memory, agent tools, and deterministic simulation are built and breathing. Starlink, HaLow radio behavior, GNSS faults, and the sea itself are simulated. Physical radios, hardware-rooted keys, sea trials, and thirty to one hundred vessels lie beyond the horizon. KeelMesh proves not merely that a fleet can sail, but that authority stays chained when the storm takes everything else.",
    },
  },
];

export const guidedDemoEstimatedSeconds = 420;
