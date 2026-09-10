# Persistent maritime combat simulation

## Purpose and boundary

M15 adds a fictional, deterministic combat workload to the Fleet operating picture. It exists to demonstrate bounded effect authority, replay, idempotency, world-time simulation, and resilient state handling. It is not a real weapon-control capability and does not describe real platform performance.

## Blackwake

`HOSTILE-0001 / Blackwake` is a persistent medium pirate raider. Its state machine progresses through roam, detection, stalking, interception, engagement, withdrawal, repair, and return. Destinations come from water-safe offshore points, recent segments are penalized for thirty world minutes, and the deterministic seed, route history, target, damage, cooldowns, and spawn generation survive a core restart.

Blackwake has 170 hull, medium armor, a 3.0 m/s speed ceiling, twin simulated deck cannons, and a finite simulated rocket load. It predicts interception from target course and speed, projects ordinary traffic onto its known closed route, and rejects pursuits that cannot reach weapon range inside a bounded window. Targets at or above 95% of its available speed are pursued only when crossing geometry offers a short intercept, so the raider does not trail faster traffic indefinitely. Its authoritative motion is distance-bounded on every simulation step; only a documented post-sinking respawn may relocate it to a different safe offshore entry. Commercial contacts escape but remain unarmed. Fictional patrol contacts may defend themselves. Blackwake withdraws below 25% integrity, after material propulsion/sensor/weapon degradation, or when three or more armed controlled vessels create an overwhelming local response. Component recovery occurs only after it reaches its offshore repair point.

Traffic speeds intentionally vary from 1.0 m/s working craft to 3.4 m/s ferries, cargo traffic, and patrol vessels. Blackwake can overtake most commercial targets but cannot catch every ship. Controlled Kestrel, Mariner, and Atlas classes top out at 4.1, 3.8, and 3.6 m/s respectively, leaving every Fleet class a deliberate overtaking margin over the fastest NPC while preserving class and energy tradeoffs.

## Authority model

Controlled vessels never fire merely because an AI, MCP client, or map menu requested it.

1. An operator or external agent drafts an engagement against any named non-Fleet contact.
2. Core resolves the controlled participants and builds a bounded `EngagementProgramV1`.
3. The program binds target scope, exact eligible contacts, participants, weapons, operating geography, range, duration, effect count, disengagement floor, reserve floor, and a SHA-256 content hash.
4. The UI presents the exact program for confirmation.
5. Only the hash-confirmed program becomes active. Shots inside that envelope do not require repeated prompts.

Controlled vessels expose separate weapons-ready and automatic-defense controls. Automatic defense is disabled by default: a missionless vessel that is attacked stays in station keeping, raises one deduplicated critical notification, and waits for the operator to maintain station, retreat, defend itself, or send nearby/group backup. Enabling automatic defense selects either retreat priority or retaliation when armed. Arming alone makes weapons available but never initiates movement, pursuit, or fire.

Every mission also carries an explicit response-if-attacked policy: notify and maintain the mission, retreat priority, or retaliation permitted. The mission policy overrides the vessel-level setting while that mission is active. A retaliation policy can auto-arm only the assigned mission participants, and they are returned to their previous safe state when mission authority ends. A confirmed attack mission automatically arms its assigned vessels inside its exact engagement envelope. Operators and the AI can inspect or change vessel-level arm/automatic-defense state, while spoken mission instructions such as `retreat if attacked` or `retaliation permitted` are preserved in the confirmed mission policy. Capability-scoped MCP remains unable to authorize its own engagement.

## Damage and repair

Every entity has hull, armor, propulsion, sensors, and weapons condition. Hit resolution is deterministic from the engagement ID, shot sequence, source, target, weapon, range falloff, accuracy, target size, armor, and component condition. The backend records every authoritative effect while the map throttles only presentation at high simulation rates.

Surviving entities regenerate one percent of maximum hull per world minute after one full undamaged world minute. A controlled vessel may receive an explicit twenty-percent repair once every five world minutes. Repair requests are idempotent, cannot target a sunk vessel, and clear at most one degraded component.

At zero hull:

- Commercial and patrol contacts disappear from navigation and rejoin their original route after three world minutes.
- Blackwake returns from a different offshore entry after five world minutes with a new spawn generation.
- A controlled vessel is disabled, leaves active mission execution, and recovers after five world minutes at its safe recovery point. It retains group membership but holds rather than rejoining the old mission.

## Operator surfaces

Tap any controlled vessel or contact to open its inspector. Long-press or right-click any non-Fleet contact to inspect it, plan a mission, or draft an engagement with the Fleet selection. Mission supports designated-contact, hostile-contact, or any-contact scopes plus area, range, effect-count, duration, and disengagement limits. Damaged vessels receive hull rings; selected or engaged armed entities receive range rings. Active effects use bounded real-time projectile, impact, damage, smoke, fire, and wreck presentation without stealing focus or opening unsolicited windows.

## API and persistence

The public read/mutation family is `/api/v8`:

- `GET /api/v8/combat`
- `GET /api/v8/combat/entities/{id}`
- `GET /api/v8/combat/engagements`
- `POST /api/v8/combat/engagements`
- `POST /api/v8/combat/engagements/{id}:authorize`
- `POST /api/v8/combat/engagements/{id}:stop`
- `POST /api/v8/combat/vessels/{id}:repair`
- `POST /api/v8/combat/vessels/{id}:arm`
- `POST /api/v8/combat/vessels/{id}:disarm`
- `POST /api/v8/combat/vessels/{id}:defense`
- `GET /api/v8/combat/events`
- `POST /api/v8/scenarios/combat:reset`

PostgreSQL stores combat projections, engagement programs, immutable events, and repair receipts. Controlled engagement, repair, disablement, and recovery mutations pass through the existing coordination middleware; effect authority remains separate from AI availability.

## Verification

The M15 suite covers balance contracts, exact-hash approval, deterministic hit resolution, world-time regeneration and respawn, repair cooldown/idempotency, default-off automatic defense, mission policy precedence, idle station keeping under attack, deduplicated attack choices, assistant action classification, MCP approval boundaries, full Go tests/vet, TypeScript/Vitest, production UI build, browser interaction, API compatibility, and live deployment checks.
