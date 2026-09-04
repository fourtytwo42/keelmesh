# Agent Instructions

This file is the repository-local agent guide. Use the uppercase filename `AGENTS.md` so coding agents can find it by convention.

`MEMORY.md` is the local durable context store for this working copy. It is intentionally ignored by Git so operational history, local infrastructure details, and agent handoffs do not become public repository documentation. Chat context may be compressed or lost, so anything future local agents need to retain must be written there.

## Startup Checklist

- Read this `AGENTS.md` file at the start of work in this repository.
- If `MEMORY.md` exists, read it before planning or making changes.
- If `MEMORY.md` is absent, create it locally with `Active Handoff`, `Current State`, `Verification Ledger`, `Open Follow-ups`, and `Do Not Forget` sections. Bootstrap facts from current source, `docs/STATUS.md`, and fresh verification; do not reconstruct private history from guesses.
- Start with these `MEMORY.md` sections: `Active Handoff`, `Current State`, `Open Follow-ups`, and `Do Not Forget`.
- If context was compressed, the session was resumed, or prior work is unclear, follow the `Compression Recovery` workflow below.
- When in doubt, check `MEMORY.md` before assuming project history, user preferences, architecture, setup steps, or unresolved work.

## Memory-First Rule

Use the local `MEMORY.md` as retained working-copy context when present. Never stage or commit it. If chat history conflicts with it, verify current repository state before acting and update the local file with any correction.

## Source Of Truth

When sources conflict, prefer them in this order:

1. Current files in the workspace.
2. Latest command output or freshly verified tool output.
3. Local `MEMORY.md` when present.
4. Maintained `docs/` status and architecture records.
5. Chat history.

Treat browser state, email state, authentication state, and external service results as stale unless verified in the current session. Treat documented repo structure, user preferences, and decisions as durable unless current files or fresh verification contradict them.

## When To Update `MEMORY.md`

Update `MEMORY.md` consistently throughout the work, not only at the end. Add or revise entries whenever information should survive context loss, including:

- User preferences or instructions that affect future work.
- Project setup, commands, environment details, credentials-handling notes, or workflow requirements.
- Architecture decisions, implementation plans, and important tradeoffs.
- Bugs found, fixes applied, test results, and known regressions.
- Unresolved questions, blocked tasks, follow-ups, and assumptions.
- File paths, modules, APIs, or integration details that future agents should not have to rediscover.

## Checkpoint Rules

Update `MEMORY.md` after any meaningful checkpoint:

- A durable decision is made.
- A non-obvious setup or command is discovered.
- A bug is diagnosed or fixed.
- Tests or verification produce important results.
- Work is paused, blocked, or handed off.
- A previous memory is found to be stale or wrong.

## Proxmox Snapshot Safety

- Treat Proxmox thin-pool capacity as constrained and potentially over-provisioned.
- Before creating any VM snapshot, inspect current thin-pool data/metadata utilization, free capacity, and the target VM's existing snapshots.
- Do not create routine or redundant snapshots. Prefer the smallest number of named recovery checkpoints required by the implementation plan.
- Never assume virtual provisioned capacity is physically available.
- Do not delete, consolidate, or roll back a snapshot without explicit user authorization and exact target verification.
- Record every created snapshot, its purpose, and the observed storage state in `MEMORY.md`.
- If utilization or over-provisioning makes another snapshot unsafe, stop and report the storage condition instead of creating it.

## Entry Quality

- Keep entries concise, factual, and easy to scan.
- Prefer dated entries with short headings.
- Include file paths, commands, and test names when they matter.
- Use `Last verified YYYY-MM-DD` for facts that can drift.
- Do not store secrets, tokens, passwords, private keys, or sensitive personal data.
- For email-related work, do not store full email bodies, private addresses, case details, or sensitive agency data. Store only brief summaries, message IDs, dates, and action state when needed.
- Do not turn `MEMORY.md` into a transcript. Store conclusions, decisions, setup details, blockers, and verified results.
- If an old memory becomes inaccurate, correct it in place or add a clearly dated correction.
- Keep `Active Handoff` and `Current State` short. Move old completed details into `Completed Work` or `Archive`.

## Memory Entry Template

Use this format when adding a meaningful dated entry:

```md
### YYYY-MM-DD - Short Title

- Context:
- Decision:
- Files:
- Commands/tests:
- Result:
- Follow-up:
```

## Compression Recovery

When context is missing, stale, compressed, or ambiguous:

1. Read this `AGENTS.md`.
2. Read local `MEMORY.md` when present, especially `Active Handoff`, `Current State`, `Open Follow-ups`, and `Do Not Forget`; otherwise initialize it from current source and maintained documentation.
3. Inspect the files related to the current task.
4. Re-run or inspect relevant verification only when needed.
5. Update `MEMORY.md` with the recovered state before making broad changes.

## Before Final Response

Before finishing substantial work:

- Review whether `MEMORY.md` needs a final update.
- Record important files changed, decisions made, test results, and follow-ups.
- Make sure `Active Handoff`, `Current State`, `Verification Ledger`, and `Open Follow-ups` reflect the actual handoff state.

## Default Operating Rule

If there is any uncertainty about retained context, read local `MEMORY.md` first when it exists, then verify against current source and maintained documentation.
