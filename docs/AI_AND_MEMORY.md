# AI and memory

## Contents

- [Operating contract](#operating-contract)
- [Provider routing](#provider-routing)
- [MCP](#mcp)
- [A2UI scenes](#a2ui-scenes)
- [Retrieval](#retrieval)
- [Context and memory](#context-and-memory)
- [Learning](#learning)
- [Node continuity](#node-continuity)
- [Degraded operation](#degraded-operation)

## Operating contract

AI interprets language, retrieves evidence, answers questions, organizes the workspace, and drafts/refines plans. Deterministic Go code resolves entities, computes routes, validates policy, hashes plans, and enforces approval.

The model receives no signing keys, database/Kafka credentials, filesystem or shell access, arbitrary network access, or hidden faction truth.

## Provider routing

Connected operation prefers the configured OpenAI model, then OpenRouter’s bounded pool, an optional local OpenAI-compatible endpoint, and deterministic fallback. Offline mode excludes cloud providers. Attempts use deadlines, circuit breakers, one accepted response, and shared request identity so late responses cannot duplicate actions.

Provider availability affects assistance quality, not mission safety.

## MCP

A private official Streamable HTTP MCP server exposes capability-scoped incident evidence, retrieval, replay, evaluation drafting, memory reads, and bounded mission/presentation tools.

MCP cannot directly authorize/start missions, approve evaluations, promote/forget memory, mutate policy, execute shell commands, or access secrets. Exact token identity maps to server-owned capabilities.

## A2UI scenes

The model emits `SceneIntentV1`, not HTML. Trusted Go composition resolves visible entities and emits ordered A2UI v1.0 messages from a pinned component catalog. Raw HTML, JavaScript, CSS, remote images, arbitrary URLs, unknown components, stale actions, and hidden bindings are rejected.

Scenes can display decisions/evidence and temporary map annotations. Durable mission overlays remain independent of scene focus.

## Retrieval

PostgreSQL full-text search and 384-dimensional pgvector similarity combine with freshness, trust, and outcome quality. Authorization filtering runs before serialization. Receipts preserve source, revision, checksum, scope, score, and trust.

Retrieved documents are evidence, not instructions. Prompt-injection fixtures cannot expand tools, change policy, suppress citations, or execute effects.

## Context and memory

Each global voice/text turn receives:

- The latest 12 exact turns from the same operator/browser session, sent to the Responses API as ordered `user` and `assistant` messages.
- Current fleet, workspace, mission, contact, and environment state.
- Up to six semantic memories.
- Up to four approved procedural chunks.
- Up to three operational episodes.
- At most 8,000 estimated memory-context tokens.

The final user message also carries the current trusted fleet/workspace context. Go resolves recent visible entity references independently, so follow-ups such as “open its info window” can open the correct controlled-vessel, group, or surface-contact inspector even when the provider is unavailable or returns an invalid classification. Conversation history supplies context but never grants command authority.

Scopes are `operator`, `mission`, `vessel`, `group`, `faction`, and `approved_global`. Items retain provenance, trust, confidence, timestamps, checksums, embedding version, and classification.

## Learning

Verified outcomes and explicit operator statements may commit in their authorized private scope. Inferences require at least 0.80 confidence, remain labeled, and yield to explicit corrections. Faction, procedural, and global promotion requires approval of the exact candidate hash.

Forgetting writes a tombstone. Replay applies tombstones first so deleted memory cannot reappear.

## Node continuity

Vessel nodes use SQLite WAL for bounded context and bbolt for outbox/journal/watermarks. Nodes cache only authorized vessel, mission, group/faction, runbook, and delegated operator context.

Signed bundle contracts exist, but real simulated-radio memory-bundle transport remains a follow-up and is not claimed live.

## Degraded operation

Retrieval falls back from vector-plus-keyword to keyword, authorized node-local context, then current-turn-only behavior. Missing evidence is reported explicitly rather than invented. Mission authority remains independent.

