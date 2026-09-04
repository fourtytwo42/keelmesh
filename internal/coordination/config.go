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
	ManifestFile                string
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

type NodeConfigFile struct {
	NodeID       string `json:"node_id"`
	CellID       string `json:"cell_id"`
	VMID         int    `json:"vm_id"`
	ManagementIP string `json:"management_ip"`
	RadioIP      string `json:"radio_ip"`
	DataDir      string `json:"data_dir"`
	PKIDir       string `json:"pki_dir"`
	Bootstrap    bool   `json:"bootstrap"`
}

func ConfigFromFile(path string) (Config, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var value NodeConfigFile
	if err := json.Unmarshal(encoded, &value); err != nil {
		return Config{}, err
	}
	manifest, err := readManifest(filepath.Join(value.PKIDir, "manifest.json"))
	if err != nil {
		return Config{}, err
	}
	if err := verifyManifestTrust(manifest, filepath.Join(value.PKIDir, "cell-ca.crt")); err != nil {
		return Config{}, err
	}
	if value.CellID != manifest.CellID {
		return Config{}, fmt.Errorf("CELL_MEMBERSHIP_DENIED: node config cell differs from manifest")
	}
	identity := domain.NodeIdentityV2{SchemaVersion: 2, NodeID: value.NodeID, CellID: value.CellID, Faction: value.CellID, VMID: value.VMID, ManagementIP: value.ManagementIP, RadioIP: value.RadioIP}
	config := Config{
		Mode: ModeRaft, Identity: identity, Manifest: manifest, ManifestFile: filepath.Join(value.PKIDir, "manifest.json"), DataDir: value.DataDir,
		RaftAddress: net.JoinHostPort(value.RadioIP, "7443"), ManagementAddress: net.JoinHostPort(value.ManagementIP, "7444"),
		RaftCertificateFile: filepath.Join(value.PKIDir, "raft.crt"), RaftTLSKeyFile: filepath.Join(value.PKIDir, "raft.key"),
		ManagementCertificateFile: filepath.Join(value.PKIDir, "management.crt"), ManagementTLSKeyFile: filepath.Join(value.PKIDir, "management.key"),
		TrustBundleFile: filepath.Join(value.PKIDir, "cell-ca.crt"), SigningKeyFile: filepath.Join(value.PKIDir, "signing.key"), RefereeSigningPublicKeyFile: filepath.Join(value.PKIDir, "referee-signing.pub"),
		ApplyTimeout: 3 * time.Second, SnapshotInterval: 5 * time.Minute, SnapshotThreshold: 1024, Bootstrap: value.Bootstrap,
	}
	if err := validateConfigIdentity(config); err != nil {
		return Config{}, err
	}
	return config, nil
}

func LoadManifest(path, trustBundle string, requireProductionNetworks bool) (domain.CoordinationCellManifestV1, error) {
	manifest, err := readManifest(path)
	if err != nil {
		return manifest, err
	}
	if err := verifyManifestTrust(manifest, trustBundle); err != nil {
		return manifest, err
	}
	if requireProductionNetworks {
		if err := validateProductionNetworks(manifest); err != nil {
			return manifest, err
		}
	}
	return manifest, nil
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
	if err := validateProductionNetworks(manifest); err != nil {
		return Config{}, err
	}
	trustBundle := filepath.Join(base, "cell-ca.crt")
	if err := verifyManifestTrust(manifest, trustBundle); err != nil {
		return Config{}, err
	}
	vmid := strconv.Itoa(spec.VMID)
	identity := domain.NodeIdentityV2{SchemaVersion: 2, NodeID: spec.NodeID, CellID: spec.Faction, Faction: spec.Faction, VMID: spec.VMID, ManagementIP: spec.ManagementIP, RadioIP: "10.77.0." + vmid}
	return Config{
		Mode: mode, Identity: identity, Manifest: manifest, ManifestFile: manifestPath,
		DataDir:     env("KEELMESH_COORDINATION_DATA_DIR", filepath.Join("/var/lib/keelmesh-node/coordination", strings.ToLower(spec.Faction))),
		RaftAddress: env("KEELMESH_RAFT_ADDRESS", identity.RadioIP+":7443"), ManagementAddress: env("KEELMESH_COORDINATION_MANAGEMENT_ADDRESS", identity.ManagementIP+":7444"),
		RaftCertificateFile: filepath.Join(base, "raft.crt"), RaftTLSKeyFile: filepath.Join(base, "raft.key"), ManagementCertificateFile: filepath.Join(base, "management.crt"), ManagementTLSKeyFile: filepath.Join(base, "management.key"), TrustBundleFile: trustBundle, SigningKeyFile: filepath.Join(base, "signing.key"), RefereeSigningPublicKeyFile: filepath.Join(base, "referee-signing.pub"),
		ApplyTimeout: 5 * time.Second, SnapshotInterval: 5 * time.Minute, SnapshotThreshold: 1024,
		Bootstrap: strings.EqualFold(strings.TrimSpace(os.Getenv("KEELMESH_COORDINATION_BOOTSTRAP")), "true"),
	}, nil
}

func GatewayConfigFromEnv() (GatewayConfig, error) {
	mode := Mode(strings.ToLower(strings.TrimSpace(env("KEELMESH_COORDINATION_MODE", string(ModeSimulated)))))
	config := GatewayConfig{Mode: mode, OperationTimeout: 8 * time.Second, Manifests: map[string]domain.CoordinationCellManifestV1{}, ManifestFiles: map[string]string{}}
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
		if err := validateProductionNetworks(manifest); err != nil {
			return config, err
		}
		if err := verifyManifestTrust(manifest, filepath.Join(base, "root-ca.crt")); err != nil {
			return config, err
		}
		config.Manifests[cellID] = manifest
		config.ManifestFiles[cellID] = path
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
	foundRoot := false
	for len(encoded) > 0 {
		block, rest := pem.Decode(encoded)
		encoded = rest
		if block == nil || block.Type != "CERTIFICATE" {
			continue
		}
		certificate, parseErr := x509.ParseCertificate(block.Bytes)
		if parseErr == nil && certificate.IsCA && certificate.CheckSignatureFrom(certificate) == nil {
			foundRoot = true
			if nowUTC().After(manifest.ExpiresAt) {
				return fmt.Errorf("PEER_CERTIFICATE_EXPIRED: coordination manifest expired")
			}
			if VerifyManifest(manifest, certificate) == nil {
				return nil
			}
		}
	}
	if !foundRoot {
		return fmt.Errorf("CELL_MEMBERSHIP_DENIED: root certificate not found in trust bundle")
	}
	return fmt.Errorf("CELL_MEMBERSHIP_DENIED: manifest signature is not trusted")
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
		if member.NodeID == "" || member.Faction != manifest.CellID || managementErr != nil || radioErr != nil || managementPort != "7444" || radioPort != "7443" || net.ParseIP(managementHost) == nil || net.ParseIP(radioHost) == nil || managementHost == radioHost || member.RaftTLSSerial == "" || member.ManagementTLSSerial == "" || publicKeyErr != nil || len(publicKey) != 32 || seenIDs[member.NodeID] || seenRadio[member.RadioAddress] || seenManagement[member.ManagementAddress] {
			return fmt.Errorf("CELL_MEMBERSHIP_DENIED: invalid or duplicate member")
		}
		seenIDs[member.NodeID] = true
		seenRadio[member.RadioAddress] = true
		seenManagement[member.ManagementAddress] = true
	}
	return nil
}

func validateProductionNetworks(manifest domain.CoordinationCellManifestV1) error {
	for _, member := range manifest.Members {
		managementHost, _, _ := net.SplitHostPort(member.ManagementAddress)
		radioHost, _, _ := net.SplitHostPort(member.RadioAddress)
		if !strings.HasPrefix(managementHost, "192.168.50.") || !strings.HasPrefix(radioHost, "10.77.0.") {
			return fmt.Errorf("CELL_MEMBERSHIP_DENIED: production manifest uses an invalid network plane")
		}
	}
	return nil
}

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
