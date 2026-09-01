# Role Alignment Audit

Status: implementation coverage contract  
Sources: recruiter transcript and supplied AI Infrastructure Engineer posting  
Last reviewed: 2026-09-01

## 1. Audit Conclusion

The project covers the fleet/autonomy product problem well, but the interview role
is not simply a C2 product role. It is an internal AI-infrastructure role serving
Autonomy, Simulation, Software, Data, and Operations teams. The build therefore
needs two connected demonstrations:

1. **Operator workflow:** create, preview, authorize, execute, and safely degrade
   a simulated multi-vessel mission.
2. **Autonomy Engineer workflow:** investigate the resulting incident through
   secure agent tools, retrieve evidence, replay it, curate it into an evaluation
   case, and run a regression before trusting a model or workflow change.

The operator workflow proves domain understanding. The engineer workflow proves
fit for the actual job.

## 2. Recruiter Transcript Coverage

| Recruiter signal | Required proof in the build | Evidence mode |
|---|---|---|
| Linux and Go platform | Go core and worker roles run in Linux containers | Live |
| AWS-backed environment | Explicit EKS/MSK-or-Kafka/RDS/S3 production mapping with workload identity and multi-zone placement | Architecture artifact |
| Heavy containers and multi-zone operation | Pinned Compose runtime plus Kubernetes manifests with probes, PDBs, anti-affinity, topology spread, and resource policy | Live + artifact |
| React, TypeScript, WebGL geospatial UI | MapLibre operator map and live cutaway | Live |
| Human orchestration moving toward agents | Suggestions and incident agent remain bounded by typed tools, policy, and approval | Live |
| Autonomy under communications loss | Peer relay, partition, mission tape, PNT degradation, and safe rejoin | Live |
| Agents for ETL and autonomy tooling | Incident agent queries telemetry/logs, launches replay, and promotes an eval candidate | Live |
| Observability and security workflows | End-to-end trace, audit timeline, forbidden-tool tests, injection case, and approval record | Live + tests |
| MCP-like servers/resources/prompts and Semantic Kernel patterns | Real scoped MCP server/client, resources, tools, prompt registry, schemas, and audit | Live |
| Vector databases and RAG | PostgreSQL/pgvector runbooks and historical incidents with citations and trust labels | Live |
| Kubernetes and Docker | Docker Compose live; Kubernetes production-shape manifests validated in CI | Live + CI artifact |
| Infrastructure at scale from an immature baseline | Measured load, partitions, multiple consumers, rebalance, lag, backpressure, dedupe, replay, and recovery report | Live + evidence |

## 3. Job Responsibility Coverage

| Posting requirement | Concrete implementation | Priority |
|---|---|---|
| Connect models to tools, APIs, data lakes, telemetry, simulation, repositories, docs, logs, and workflows | MCP resources/tools for fleet state, telemetry, logs, policy, replay, runbooks, similar incidents, Git commit/build metadata, and eval promotion; object archive remains an interface/optional profile | P0 except object archive |
| Agentic task automation and analysis | Typed intent assistance plus incident investigation and eval-case preparation | P0 |
| MCP servers, clients, tools, resources, prompts, connectors, secure patterns | Go MCP server, Python client, scoped capability registry, versioned schemas/prompts, tool audit, and denied-tool tests | P0 |
| Retrieval, RAG, context, document processing, embeddings, search | Seed ingestion, chunk/provenance records, pgvector retrieval, citations, trust labels, context budget, and injection fixtures | P0 narrow corpus |
| Data preparation and dataset curation | Human-approved incident-to-eval manifest with source IDs, scenario seed, expected behavior, lineage, and review state | P0 |
| Experiment tracking and model evaluation | Versioned evaluation runner records provider/model/prompt/tool versions, latency, validity, task success, and failures; MLflow export is P1 | P0 runner, P1 MLflow |
| Fine-tuning support | Curated manifests are exportable training/eval inputs; no Friday fine-tuning claim | Design/evidence |
| Model deployment | Provider registry, health, timeout, circuit breaker, local/cloud/mock routing, and rollback-safe configuration | P0 |
| Agent/tool/pipeline observability | OpenTelemetry spans, structured logs, metrics, trace IDs, audit timeline, and evidence export | P0 |
| Cross-functional autonomy tooling | Operator mode, Autonomy Engineer mode, simulation replay, telemetry analysis, and reusable templates | P0 |
| Least privilege and sandboxing | Read/propose tool scopes, no execution tools, schema validation, process/container limits, redaction, and human approval | P0 |
| Prompt injection and misuse mitigation | Retrieved-data trust boundary, content labeling, forbidden instruction fixture, tool policy outside prompts, and audit | P0 |
| Secrets and sensitive data | Environment/file injection outside images, log redaction, browser isolation, no secrets in evidence or Git, classification labels in fixtures | P0 |
| Documentation/templates/best practices | One-command guide, MCP tool template, eval fixture template, runbook template, architecture decisions, and verification report | P0/P1 |
| GitHub/CI/CD integration | Repository, GitHub Actions, commit/build metadata resource, schema/eval/container/Kubernetes checks | P0 narrow integration |

## 4. Scale Proof Contract

Scale must be demonstrated as system behavior, not a large number printed on a
dashboard. The minimum credible Scale Lab has:

- The same event envelope and ingestion path for visible and background vessels.
- Kafka topics partitioned by fleet/vessel key with separate telemetry, mission,
  audit, retry, and dead-letter concerns.
- At least two real Go consumer processes in one consumer group.
- PostgreSQL uniqueness on the logical event key and monotonic projection guards.
- Out-of-order, duplicate, corrupt, slow-consumer, worker-loss, provider-timeout,
  Kafka-loss, and PostgreSQL-loss fault cases.
- Mission control isolated from telemetry, AI, and batch queues so load cannot
  starve active safety/control state.
- Backpressure and bounded edge buffering instead of unlimited memory growth.
- Rebalance and backlog-drain behavior after worker termination or scale-out.
- Deterministic replay that reproduces the final projection.
- Real events/second, bytes/second, p50/p95/p99 latency, peak/current lag,
  duplicates suppressed, quarantined events, dropped events, recovery time,
  CPU, memory, and database/Kafka health.
- An evidence export tied to git commit, image digest, scenario seed, workload
  parameters, and hardware summary.

The live target begins at 1,000 simulated vessels, but the report must state the
measured sustainable event rate. No production capacity is inferred from a
single-VM benchmark.

## 5. Autonomy Tooling Proof Contract

The Vessel 4 communications/PNT incident becomes the shared artifact across the
operator and engineering workflows:

1. Select the incident from the audit timeline.
2. The agent receives only the `incident-investigator` capability set.
3. It reads trace metadata, bounded telemetry windows, PNT evidence, mission-tape
   lifecycle, policy decisions, logs, and provider/link state.
4. It retrieves a cited communications-loss runbook and one similar historical
   simulation through pgvector.
5. It calls deterministic simulation replay with the original scenario seed and
   proposes a diagnosis and regression assertions.
6. It is denied if it attempts mission authorization, deployment, policy mutation,
   raw vessel command, audit deletion, unrestricted shell, or secret access.
7. A human reviews provenance and approves promotion to a versioned eval case.
8. The eval runner executes the case against cloud/local/mock configurations and
   records task success, tool sequence, schema validity, latency, citations,
   policy compliance, and failure analysis.
9. The result is visible in Cutaway mode and exported as evidence.

This workflow demonstrates an autonomy data flywheel without claiming that the
demo trains or deploys a production navigation model.

## 6. Tooling Choices and Non-Choices

### Used live because they prove a requirement

- Go, Python, React, TypeScript, MapLibre/WebGL
- Docker Compose
- Kafka KRaft
- PostgreSQL and pgvector
- MCP server/client with versioned tools/resources/prompts
- OpenTelemetry SDK and structured logs
- Cloud/local/mock OpenAI-compatible provider adapters
- GitHub Actions
- JSON Schema, deterministic scenarios, and a versioned eval runner

### Supporting artifacts, not live dependencies

- Kubernetes manifests and AWS multi-zone mapping
- S3-compatible archive interface and optional MinIO profile
- MLflow export adapter
- Dagster-style curation asset graph

### Intentionally not added just for name recognition

- Ray, Airflow, LangChain, LangGraph, LlamaIndex, Weights & Biases, and C++

These are valid alternatives named by the posting, not a checklist requiring all
of them. Adding overlapping frameworks without a demonstrated need would weaken
the ownership and maintainability story. The interview explanation should focus
on stable contracts and why each selected tool earns its operational cost.

## 7. Remaining Gaps

The following must be completed before claiming full coverage:

- Implement the real MCP server/client and capability enforcement.
- Implement pgvector ingestion/retrieval with provenance and injection fixtures.
- Run more than one actual Kafka consumer process and capture rebalance evidence.
- Instrument end-to-end OpenTelemetry traces.
- Implement incident replay and approved eval promotion.
- Add GitHub Actions and validate Kubernetes production-shape manifests.
- Measure the VM; no throughput or latency claim exists until then.
- Add secret scanning/redaction checks and document sensitive-data boundaries.

