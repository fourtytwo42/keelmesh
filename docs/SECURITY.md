# Security model

## Contents

- [Objectives](#objectives)
- [Assets and boundaries](#assets-and-boundaries)
- [Authority classes](#authority-classes)
- [MCP security](#mcp-security)
- [Data isolation](#data-isolation)
- [Network safety](#network-safety)
- [Secrets](#secrets)
- [AI threats](#ai-threats)

## Objectives

KeelMesh fails closed for mission authority, prevents models from self-authorizing effects, keeps hidden knowledge out of unauthorized serialization, limits simulated fault blast radius, and preserves immutable evidence.

## Assets and boundaries

Protected assets include mission leases/signatures, operator and faction knowledge, provider/MCP credentials, memory scopes/tombstones, operational data, and management-network availability.

The browser is untrusted presentation input. Core is mission/memory authority. Python is a bounded client without database, Kafka, signing, policy, or mission credentials. Provider output and retrieved documents are untrusted.

## Authority classes

| Class | Examples | Behavior |
|---|---|---|
| Presentation | Select, frame, open window | Immediate after visibility checks |
| Reversible draft | Objective, geometry, formation | Versioned; invalidates stale plans |
| Safer bounded | Pause, tighten constraint, hold | Audited and policy checked |
| Consequential | Start, loosen limits, membership, delete, simulated effect | Exact state/hash-bound approval |
| Prohibited | Signing, bypass, hidden truth, shell, secrets | Not exposed |

## MCP security

- Random runtime tokens live on private volumes.
- Exact token identity maps to server-side capabilities.
- Schema, body, deadline, record, window, and tool-budget limits are external to prompts.
- Receipts are immutable.
- Control MCP omits direct authorize/start/effect tools.

## Data isolation

Authorization filters execute before browser/model serialization. Memory retrieval checks actor and scope. Opponent/referee fields never reach player clients. Raw voice audio is discarded after transcription.

## Network safety

Only core `8080` is published. Simulated Starlink/HaLow faults never target Proxmox management. Radio impairment permits only the radio NIC and requires automatic rollback.

## Secrets

- Keep secrets out of Git, images, cloud-init, browser payloads, logs, and evidence.
- Mount key files read-only for the narrowest UID.
- Never export full environments.
- Rotate any credential exposed in conversation or logs.

## AI threats

| Threat | Control |
|---|---|
| Prompt injection | Evidence cannot change tools or policy |
| Hallucinated entity | Core resolves authorized current IDs |
| Duplicate response | Request identity, deadlines, one accepted result |
| Stale approval | Exact hash and state/version validation |
| Tool escalation | Capability allow-list |
| UI injection | Trusted A2UI catalog; no raw HTML/JS/CSS/URLs |
| Memory poisoning | Provenance, precedence, confidence, approval, tombstones |
| Hidden-data leak | Filter before serialization/context assembly |

This demo does not claim formal certification, production penetration testing, or an external security assessment.
