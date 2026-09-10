# M1 Contracts

Go types in `internal/domain` are the canonical M1 wire contract. The React
types in `web/src/types.ts` mirror their JSON fields, and both test suites read
the fixtures in this directory. All M1 contract objects carry
`schema_version: 1`; breaking changes require a new version rather than an
in-place reinterpretation.

Later fixtures retain the same rule. `combat-v8.json` mirrors the M15 Go and TypeScript combat, engagement, and repair contracts and labels every effect as fictional simulation state.
