# M10 — AI Command Surface and A2UI Operational Scenes

## Implemented architecture

M10 adds an AI-first scene layer without moving mission authority into the model. `POST /api/v4/assistant/turns` gathers the current operator-visible fleet snapshot, calls the bounded workspace assistant, and passes its result through a trusted Go scene composer. The composer—not the model—resolves IDs, permissions, live bindings, map callouts, actions, and the ordered A2UI message stream.

The browser renders those messages with pinned `@a2ui/react` 0.11.0 and React 19.2.8. The wire profile is A2UI v1.0 using ordered `createSurface`, `updateComponents`, and `updateDataModel` messages. The package currently exports its runtime from a `v0_9` module path, so a narrow adapter maps only the envelope version and trusted catalog identifier before rendering; content remains server-composed. KeelMesh's catalog endpoint documents the allowed domain components and explicitly rejects HTML, JavaScript, CSS, remote images, arbitrary URLs, and unknown components.

## Operator model

- One active unpinned command scene per browser/operator session; a later request in that session replaces it without affecting another console.
- Pinned, critical, and approval-bearing scenes are protected from replacement.
- Up to four scenes may be pinned, with the latest fifty retained in bounded history.
- Informational requests create movable A2UI artifact windows and scene-owned map callouts.
- Mission requests open Mission Canvas. Conversation history is hidden by default; the populated mission summary, live scene, options, and manual editing controls share one surface.
- Existing exact-plan preview, hash, lease, and start validation remain authoritative. The existing confirmation is presented as an Approval Card.
- Live entity values and map positions refresh from current fleet state without another provider call.
- Deterministic critical triggers cover unsafe PNT, critical cached-tape depth, and critical reserve. They are deduplicated, remain visible until dismissed/resolved, and do not enact revisions.

## Interfaces

- `POST /api/v4/assistant/turns`
- `POST /api/v4/assistant/turns/{id}:cancel`
- `GET /api/v4/assistant/turns/{id}/events`
- `GET /api/v4/scenes`
- `GET /api/v4/scenes/{id}`
- `POST /api/v4/scenes/{id}:pin|unpin|dismiss`
- `POST /api/v4/scenes/{id}/actions`
- `GET /api/v4/assistant/history`
- `GET /api/v4/catalogs/keelmesh-operations-v1`

The private MCP boundary exposes `scene.list`, `scene.get`, and `scene.compose`. It still exposes no mission authorization, mission start, arbitrary network, shell, filesystem, or secret tool.

## Security and authority

All scene reads and mutations bind actor identity, browser session, and workspace version. Artifact actions are selected from server-composed action IDs and hashes. Effect-class actions fail with `APPROVAL_REQUIRED` unless the exact action hash and explicit confirmation are supplied. A2UI is a presentation protocol only; deterministic Go planners and policy remain the execution boundary.
