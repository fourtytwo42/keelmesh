# Persistent maritime combat simulation

## Purpose and boundary

M15 adds a fictional, deterministic combat workload to the Fleet operating picture. It exists to demonstrate bounded effect authority, replay, idempotency, world-time simulation, and resilient state handling. It is not a real weapon-control capability and does not describe real platform performance.

## Blackwake

`HOSTILE-0001 / Blackwake` is a persistent medium pirate raider. Its state machine progresses through roam, detection, stalking, interception, engagement, withdrawal, repair, and return. Destinations come from water-safe offshore points, recent segments are penalized for thirty world minutes, and the deterministic seed, route history, target, damage, cooldowns, and spawn generation survive a core restart.

Blackwake has 170 hull, medium armor, a 2.6 m/s speed ceiling, twin simulated deck cannons, and a finite simulated rocket load. Commercial contacts escape but remain unarmed. Fictional patrol contacts may defend themselves. Blackwake withdraws below 25% integrity or when three or more armed controlled vessels create an overwhelming local response.

## Authority model

Controlled vessels never fire merely because an AI, MCP client, or map menu requested it.

1. An operator or external agent drafts an engagement against a hostile identity.
2. Core resolves the controlled participants and builds a bounded `EngagementProgramV1`.
3. The program binds target, participants, weapons, range, duration, effect count, disengagement floor, reserve floor, and a SHA-256 content hash.
4. The UI presents the exact program for confirmation.
5. Only the hash-confirmed program becomes active. Shots inside that envelope do not require repeated prompts.

AI can inspect combat state, execute an eligible bounded repair, and draft an engagement. Capability-scoped MCP can read status, list engagements, draft an engagement, and request a controlled-vessel repair. MCP cannot authorize its own engagement.

## Damage and repair

Every entity has hull, armor, propulsion, sensors, and weapons condition. Hit resolution is deterministic from the engagement ID, shot sequence, source, target, weapon, range falloff, accuracy, target size, armor, and component condition. The backend records every authoritative effect while the map throttles only presentation at high simulation rates.

Surviving entities regenerate one percent of maximum hull per world minute after one full undamaged world minute. A controlled vessel may receive an explicit twenty-percent repair once every five world minutes. Repair requests are idempotent, cannot target a sunk vessel, and clear at most one degraded component.

At zero hull:

- Commercial and patrol contacts disappear from navigation and rejoin their original route after three world minutes.
- Blackwake returns from a different offshore entry after five world minutes with a new spawn generation.
- A controlled vessel is disabled, leaves active mission execution, and recovers after five world minutes at its safe recovery point. It retains group membership but holds rather than rejoining the old mission.

## Operator surfaces

Tap a contact to inspect integrity, systems, armament, range, target, repair state, and engagement state. Long-press or right-click Blackwake to inspect it, plan an interception, or draft an engagement with the Fleet selection. Damaged vessels receive hull rings; selected or engaged armed entities receive range rings. Active effects use bounded real-time projectile, impact, damage, smoke, fire, and wreck presentation without stealing focus or opening unsolicited windows.

## API and persistence

The public read/mutation family is `/api/v8`:

- `GET /api/v8/combat`
- `GET /api/v8/combat/entities/{id}`
- `GET /api/v8/combat/engagements`
- `POST /api/v8/combat/engagements`
- `POST /api/v8/combat/engagements/{id}:authorize`
- `POST /api/v8/combat/engagements/{id}:stop`
- `POST /api/v8/combat/vessels/{id}:repair`
- `GET /api/v8/combat/events`
- `POST /api/v8/scenarios/combat:reset`

PostgreSQL stores combat projections, engagement programs, immutable events, and repair receipts. Controlled engagement, repair, disablement, and recovery mutations pass through the existing coordination middleware; effect authority remains separate from AI availability.

## Verification

The M15 suite covers balance contracts, exact-hash approval, deterministic hit resolution, world-time regeneration and respawn, repair cooldown/idempotency, assistant action classification, MCP approval boundaries, full Go tests/vet, TypeScript/Vitest, production UI build, browser interaction, API compatibility, and live deployment checks.

