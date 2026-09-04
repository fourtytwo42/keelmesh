package coordination

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

type Mode string

const (
	ModeSimulated Mode = "simulated"
	ModeShadow    Mode = "shadow"
	ModeRaft      Mode = "raft"
)

type Config struct {
	Mode                        Mode
	Identity                    domain.NodeIdentityV2
	Manifest                    domain.CoordinationCellManifestV1
	DataDir                     string
	RaftAddress                 string
	ManagementAddress           string
	RaftCertificateFile         string
	RaftTLSKeyFile              string
	ManagementCertificateFile   string
	ManagementTLSKeyFile        string
	TrustBundleFile             string
	SigningKeyFile              string
	RefereeSigningPublicKeyFile string
	ApplyTimeout                time.Duration
	SnapshotInterval            time.Duration
	SnapshotThreshold           uint64
	Bootstrap                   bool
}

func ConfigFromEnv() (Config, error) {
	mode := Mode(strings.ToLower(strings.TrimSpace(env("KEELMESH_COORDINATION_MODE", string(ModeSimulated)))))
	if mode != ModeSimulated && mode != ModeShadow && mode != ModeRaft {
		return Config{}, fmt.Errorf("invalid KEELMESH_COORDINATION_MODE %q", mode)
	}
	if mode == ModeSimulated {
		return Config{Mode: mode}, nil
	}
	nodeID := strings.TrimSpace(os.Getenv("KEELMESH_NODE_ID"))
	if nodeID == "" {
		return Config{Mode: mode}, nil
	}
	var spec *domain.NodeFleetSpec
	for _, candidate := range domain.VMFleetSpecs() {
		if candidate.NodeID == nodeID {
			copy := candidate
			spec = &copy
			break
		}
	}
	if spec == nil {
		return Config{}, fmt.Errorf("unknown node identity %q", nodeID)
	}
	base := env("KEELMESH_COORDINATION_PKI_DIR", "/etc/keelmesh/coordination")
	manifestPath := env("KEELMESH_COORDINATION_MANIFEST", filepath.Join(base, "manifest.json"))
	manifest, err := readManifest(manifestPath)
	if err != nil {
		return Config{}, err
	}
	trustBundle := filepath.Join(base, "cell-ca.crt")
	if err := verifyManifestTrust(manifest, trustBundle); err != nil {
		return Config{}, err
	}
	vmid := strconv.Itoa(spec.VMID)
	identity := domain.NodeIdentityV2{SchemaVersion: 2, NodeID: spec.NodeID, CellID: spec.Faction, Faction: spec.Faction, VMID: spec.VMID, ManagementIP: spec.ManagementIP, RadioIP: "10.77.0." + vmid}
	return Config{
		Mode: mode, Identity: identity, Manifest: manifest,
		DataDir:     env("KEELMESH_COORDINATION_DATA_DIR", filepath.Join("/var/lib/keelmesh-node/coordination", strings.ToLower(spec.Faction))),
		RaftAddress: env("KEELMESH_RAFT_ADDRESS", identity.RadioIP+":7443"), ManagementAddress: env("KEELMESH_COORDINATION_MANAGEMENT_ADDRESS", identity.ManagementIP+":7444"),
		RaftCertificateFile: filepath.Join(base, "raft.crt"), RaftTLSKeyFile: filepath.Join(base, "raft.key"), ManagementCertificateFile: filepath.Join(base, "management.crt"), ManagementTLSKeyFile: filepath.Join(base, "management.key"), TrustBundleFile: trustBundle, SigningKeyFile: filepath.Join(base, "signing.key"), RefereeSigningPublicKeyFile: filepath.Join(base, "referee-signing.pub"),
		ApplyTimeout: 5 * time.Second, SnapshotInterval: 5 * time.Minute, SnapshotThreshold: 1024,
		Bootstrap: strings.EqualFold(strings.TrimSpace(os.Getenv("KEELMESH_COORDINATION_BOOTSTRAP")), "true"),
	}, nil
}

func GatewayConfigFromEnv() (GatewayConfig, error) {
	mode := Mode(strings.ToLower(strings.TrimSpace(env("KEELMESH_COORDINATION_MODE", string(ModeSimulated)))))
	config := GatewayConfig{Mode: mode, OperationTimeout: 8 * time.Second, Manifests: map[string]domain.CoordinationCellManifestV1{}}
	if mode == ModeSimulated {
		return config, nil
	}
	base := env("KEELMESH_COORDINATION_REFEREE_PKI_DIR", "/etc/keelmesh/coordination/referee")
	for _, cellID := range []string{"A", "B"} {
		path := env("KEELMESH_COORDINATION_CELL_"+cellID+"_MANIFEST", filepath.Join("/etc/keelmesh/coordination/cells", strings.ToLower(cellID), "manifest.json"))
		manifest, err := readManifest(path)
		if err != nil {
			return config, err
		}
		if err := verifyManifestTrust(manifest, filepath.Join(base, "root-ca.crt")); err != nil {
			return config, err
		}
		config.Manifests[cellID] = manifest
	}
	config.CertificateFile = filepath.Join(base, "referee.crt")
	config.TLSKeyFile = filepath.Join(base, "referee.key")
	config.SigningKeyFile = filepath.Join(base, "signing.key")
	config.TrustBundleFile = filepath.Join(base, "root-ca.crt")
	config.StateFile = env("KEELMESH_COORDINATION_GATEWAY_STATE", "/var/lib/keelmesh/coordination-gateway/state.json")
	return config, nil
}

func verifyManifestTrust(manifest domain.CoordinationCellManifestV1, trustBundle string) error {
	encoded, err := os.ReadFile(trustBundle)
	if err != nil {
		return fmt.Errorf("read manifest trust bundle: %w", err)
	}
	for len(encoded) > 0 {
		block, rest := pem.Decode(encoded)
		encoded = rest
		if block == nil || block.Type != "CERTIFICATE" {
			continue
		}
		certificate, parseErr := x509.ParseCertificate(block.Bytes)
		if parseErr == nil && certificate.IsCA && certificate.CheckSignatureFrom(certificate) == nil {
			if nowUTC().After(manifest.ExpiresAt) {
				return fmt.Errorf("PEER_CERTIFICATE_EXPIRED: coordination manifest expired")
			}
			return VerifyManifest(manifest, certificate)
		}
	}
	return fmt.Errorf("CELL_MEMBERSHIP_DENIED: root certificate not found in trust bundle")
}

func readManifest(path string) (domain.CoordinationCellManifestV1, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return domain.CoordinationCellManifestV1{}, fmt.Errorf("read coordination manifest: %w", err)
	}
	var manifest domain.CoordinationCellManifestV1
	if err := json.Unmarshal(encoded, &manifest); err != nil {
		return manifest, fmt.Errorf("decode coordination manifest: %w", err)
	}
	if err := validateManifest(manifest); err != nil {
		return manifest, err
	}
	return manifest, nil
}

func validateManifest(manifest domain.CoordinationCellManifestV1) error {
	if manifest.SchemaVersion != 1 || manifest.CellID == "" || manifest.ClusterID == "" || len(manifest.Members) != 6 || manifest.Quorum != 4 {
		return fmt.Errorf("CELL_MEMBERSHIP_DENIED: invalid fixed cell manifest")
	}
	seenIDs := map[string]bool{}
	seenRadio := map[string]bool{}
	seenManagement := map[string]bool{}
	for _, member := range manifest.Members {
		managementHost, managementPort, managementErr := net.SplitHostPort(member.ManagementAddress)
		radioHost, radioPort, radioErr := net.SplitHostPort(member.RadioAddress)
		publicKey, publicKeyErr := base64.StdEncoding.DecodeString(member.SigningPublicKey)
		if member.NodeID == "" || member.Faction != manifest.CellID || managementErr != nil || radioErr != nil || managementPort != "7444" || radioPort != "7443" || !strings.HasPrefix(managementHost, "192.168.50.") || !strings.HasPrefix(radioHost, "10.77.0.") || member.RaftTLSSerial == "" || member.ManagementTLSSerial == "" || publicKeyErr != nil || len(publicKey) != 32 || seenIDs[member.NodeID] || seenRadio[member.RadioAddress] || seenManagement[member.ManagementAddress] {
			return fmt.Errorf("CELL_MEMBERSHIP_DENIED: invalid or duplicate member")
		}
		seenIDs[member.NodeID] = true
		seenRadio[member.RadioAddress] = true
		seenManagement[member.ManagementAddress] = true
	}
	return nil
}

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
