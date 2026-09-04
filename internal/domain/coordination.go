package domain

import (
	"encoding/json"
	"time"
)

type CoordinationCellMemberV1 struct {
	NodeID                       string   `json:"node_id"`
	Faction                      string   `json:"faction"`
	VMID                         int      `json:"vm_id"`
	Host                         string   `json:"host"`
	ManagementAddress            string   `json:"management_address"`
	RadioAddress                 string   `json:"radio_address"`
	RaftTLSSerial                string   `json:"raft_tls_serial,omitempty"`
	ManagementTLSSerial          string   `json:"management_tls_serial,omitempty"`
	PreviousRaftTLSSerials       []string `json:"previous_raft_tls_serials,omitempty"`
	PreviousManagementTLSSerials []string `json:"previous_management_tls_serials,omitempty"`
	SigningPublicKey             string   `json:"signing_public_key,omitempty"`
}

type CoordinationCellManifestV1 struct {
	SchemaVersion  int                        `json:"schema_version"`
	CellID         string                     `json:"cell_id"`
	ClusterID      string                     `json:"cluster_id"`
	Quorum         int                        `json:"quorum"`
	Members        []CoordinationCellMemberV1 `json:"members"`
	IssuedAt       time.Time                  `json:"issued_at"`
	ExpiresAt      time.Time                  `json:"expires_at"`
	TrustVersion   int                        `json:"trust_version"`
	RevokedSerials []string                   `json:"revoked_serials,omitempty"`
	Checksum       string                     `json:"checksum"`
	Signature      string                     `json:"signature,omitempty"`
}

type NodeIdentityV2 struct {
	SchemaVersion         int       `json:"schema_version"`
	NodeID                string    `json:"node_id"`
	CellID                string    `json:"cell_id"`
	Faction               string    `json:"faction"`
	VMID                  int       `json:"vm_id"`
	ManagementIP          string    `json:"management_ip"`
	RadioIP               string    `json:"radio_ip"`
	RaftCertificate       string    `json:"raft_certificate_serial,omitempty"`
	ManagementCertificate string    `json:"management_certificate_serial,omitempty"`
	ExpiresAt             time.Time `json:"certificate_expires_at,omitempty"`
}

type ReplicatedCommandV1 struct {
	SchemaVersion      int             `json:"schema_version"`
	CommandID          string          `json:"command_id"`
	RequestID          string          `json:"request_id"`
	IdempotencyKey     string          `json:"idempotency_key"`
	ActorIdentity      string          `json:"actor_identity"`
	CellID             string          `json:"cell_id"`
	Term               uint64          `json:"term,omitempty"`
	AuthorityEpoch     uint64          `json:"authority_epoch"`
	Kind               string          `json:"kind"`
	EntityID           string          `json:"entity_id,omitempty"`
	ExpectedVersion    int64           `json:"expected_version,omitempty"`
	Payload            json.RawMessage `json:"payload,omitempty"`
	PayloadHash        string          `json:"payload_hash"`
	IssuedAt           time.Time       `json:"issued_at"`
	FutureActivationAt *time.Time      `json:"future_activation_at,omitempty"`
	ParentOperationID  string          `json:"parent_operation_id,omitempty"`
}

type AppliedCommandReceiptV1 struct {
	SchemaVersion      int       `json:"schema_version"`
	CommandID          string    `json:"command_id"`
	CellID             string    `json:"cell_id"`
	Term               uint64    `json:"term"`
	LogIndex           uint64    `json:"log_index"`
	AuthorityEpoch     uint64    `json:"authority_epoch"`
	CommandHash        string    `json:"command_hash"`
	ResultingStateHash string    `json:"resulting_state_hash"`
	AppliedAt          time.Time `json:"applied_at"`
	State              string    `json:"state"`
}

type SignedNodeAcknowledgementV1 struct {
	NodeID             string `json:"node_id"`
	CellID             string `json:"cell_id"`
	Term               uint64 `json:"term"`
	LogIndex           uint64 `json:"log_index"`
	AuthorityEpoch     uint64 `json:"authority_epoch"`
	CommandHash        string `json:"command_hash"`
	ResultingStateHash string `json:"resulting_state_hash"`
	Signature          string `json:"signature"`
}

type QuorumCommitProofV1 struct {
	SchemaVersion      int                           `json:"schema_version"`
	CommandID          string                        `json:"command_id"`
	CellID             string                        `json:"cell_id"`
	Term               uint64                        `json:"term"`
	LogIndex           uint64                        `json:"log_index"`
	AuthorityEpoch     uint64                        `json:"authority_epoch"`
	CommandHash        string                        `json:"command_hash"`
	ResultingStateHash string                        `json:"resulting_state_hash"`
	Required           int                           `json:"required"`
	Acknowledgements   []SignedNodeAcknowledgementV1 `json:"acknowledgements"`
	State              string                        `json:"state"`
	CompletedAt        time.Time                     `json:"completed_at,omitempty"`
}

type CoordinatorAdvertisementV1 struct {
	SchemaVersion  int       `json:"schema_version"`
	CellID         string    `json:"cell_id"`
	NodeID         string    `json:"node_id"`
	Term           uint64    `json:"term"`
	AuthorityEpoch uint64    `json:"authority_epoch"`
	ManagementURL  string    `json:"management_url"`
	CommitIndex    uint64    `json:"commit_index"`
	IssuedAt       time.Time `json:"issued_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	Signature      string    `json:"signature"`
	State          string    `json:"state"`
}

type CoordinationPeerV1 struct {
	NodeID       string    `json:"node_id"`
	Host         string    `json:"host"`
	Role         string    `json:"role"`
	Reachable    bool      `json:"reachable"`
	AppliedIndex uint64    `json:"applied_index"`
	Lag          uint64    `json:"lag"`
	LastContact  time.Time `json:"last_contact,omitempty"`
}

type CoordinationCellSnapshotV1 struct {
	SchemaVersion   int                      `json:"schema_version"`
	CellID          string                   `json:"cell_id"`
	ClusterID       string                   `json:"cluster_id"`
	Mode            string                   `json:"mode"`
	LocalNodeID     string                   `json:"local_node_id,omitempty"`
	LeaderNodeID    string                   `json:"leader_node_id,omitempty"`
	LeaderAddress   string                   `json:"leader_address,omitempty"`
	State           string                   `json:"state"`
	Term            uint64                   `json:"term"`
	AuthorityEpoch  uint64                   `json:"authority_epoch"`
	StateVersion    int64                    `json:"state_version"`
	CommitIndex     uint64                   `json:"commit_index"`
	AppliedIndex    uint64                   `json:"applied_index"`
	LastLogIndex    uint64                   `json:"last_log_index"`
	QuorumRequired  int                      `json:"quorum_required"`
	ReachableVoters int                      `json:"reachable_voters"`
	StateHash       string                   `json:"state_hash"`
	ElectionCount   uint64                   `json:"election_count"`
	LastElectionMS  int64                    `json:"last_election_ms,omitempty"`
	Peers           []CoordinationPeerV1     `json:"peers"`
	LatestReceipt   *AppliedCommandReceiptV1 `json:"latest_receipt,omitempty"`
	UpdatedAt       time.Time                `json:"updated_at"`
	LastError       string                   `json:"last_error,omitempty"`
}

type CrossCellOperationV1 struct {
	SchemaVersion int                               `json:"schema_version"`
	ID            string                            `json:"id"`
	CommandHash   string                            `json:"command_hash"`
	State         string                            `json:"state"`
	ActivationAt  time.Time                         `json:"activation_at"`
	PrepareProofs map[string]QuorumCommitProofV1    `json:"prepare_proofs,omitempty"`
	Certificate   *CrossCellActivationCertificateV1 `json:"certificate,omitempty"`
	CreatedAt     time.Time                         `json:"created_at"`
	UpdatedAt     time.Time                         `json:"updated_at"`
}

type CrossCellActivationCertificateV1 struct {
	SchemaVersion int                            `json:"schema_version"`
	OperationID   string                         `json:"operation_id"`
	CommandHash   string                         `json:"command_hash"`
	ActivationAt  time.Time                      `json:"activation_at"`
	PrepareProofs map[string]QuorumCommitProofV1 `json:"prepare_proofs"`
	IssuedAt      time.Time                      `json:"issued_at"`
	Issuer        string                         `json:"issuer"`
	Signature     string                         `json:"signature"`
}

type PeerTLSStateV1 struct {
	NodeID       string    `json:"node_id"`
	CellID       string    `json:"cell_id"`
	Serial       string    `json:"serial"`
	ExpiresAt    time.Time `json:"expires_at"`
	Trusted      bool      `json:"trusted"`
	LastError    string    `json:"last_error,omitempty"`
	LastVerified time.Time `json:"last_verified,omitempty"`
}
