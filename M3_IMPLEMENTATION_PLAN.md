# M3 Infrastructure at Scale

M3 turns the KeelMesh appliance into a measured single-VM scale laboratory while preserving M1/M2 as an independent control plane.

The shipping topology uses one multi-role Go image for `core`, deterministic `loadgen`, three worker supervisors with real child consumer processes, and one-shot migration/topic initialization. Apache Kafka 4.2.1 runs in KRaft mode and PostgreSQL 17 includes pgvector 0.8.6. Only port 8080 is exposed.

The demonstration sequence is: generate 1,000 vessels at 2 Hz, establish a measured baseline, terminate Worker 2's child process through a signed Kafka control command, watch cooperative partition reassignment and lag, observe supervised respawn and recovery, repair an invalid-checksum quarantine record, and rebuild an isolated shadow projection from Kafka's earliest retained offsets. The Operator/Cutaway switch presents this path as a live system cross-section.

Acceptance is recorded by `scripts/verify_m3.py`. It captures deterministic seed/profile, attempted and accounted events, actual latency/lag, old and new worker PIDs, rebalances, recovery time, duplicate suppression, out-of-order handling, quarantine redrive, pgvector retrieval, replay count/checksum parity, and the VM-scoped hardware label.

No performance result is a production capacity claim. The local Kafka broker is not highly available, fixture embeddings are clearly labeled, and no AWS resources are provisioned. Kubernetes manifests describe production shape only.
