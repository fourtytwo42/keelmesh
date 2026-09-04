package coordination

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

type PKIConfig struct {
	OutputDir string
	ValidFor  time.Duration
	Now       time.Time
	Nodes     []PKINodeSpec
}

type PKINodeSpec struct {
	NodeID       string
	CellID       string
	VMID         int
	Host         string
	ManagementIP string
	RadioIP      string
}

type PKIRotationReceipt struct {
	SchemaVersion int       `json:"schema_version"`
	TrustVersion  int       `json:"trust_version"`
	StagedAt      time.Time `json:"staged_at"`
	OverlapUntil  time.Time `json:"overlap_until"`
	PreviousPKI   string    `json:"previous_pki"`
}

func GeneratePKI(cfg PKIConfig) error {
	if cfg.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}
	if cfg.ValidFor <= 0 {
		cfg.ValidFor = 30 * 24 * time.Hour
	}
	if cfg.Now.IsZero() {
		cfg.Now = nowUTC()
	}
	if entries, err := os.ReadDir(cfg.OutputDir); err == nil && len(entries) > 0 {
		return fmt.Errorf("output directory is not empty; refusing to overwrite PKI")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(cfg.OutputDir, 0o700); err != nil {
		return err
	}
	rootPublic, rootPrivate, _ := ed25519.GenerateKey(rand.Reader)
	rootTemplate := certificateTemplate("KeelMesh M12 Demo Root", cfg.Now, cfg.Now.Add(cfg.ValidFor), true)
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, rootPublic, rootPrivate)
	if err != nil {
		return err
	}
	rootCert, _ := x509.ParseCertificate(rootDER)
	if err := writeCertificate(filepath.Join(cfg.OutputDir, "root-ca.crt"), rootDER, 0o444); err != nil {
		return err
	}
	if err := writePrivateKey(filepath.Join(cfg.OutputDir, "root-ca.key"), rootPrivate); err != nil {
		return err
	}

	if len(cfg.Nodes) == 0 {
		for _, spec := range domain.VMFleetSpecs() {
			cfg.Nodes = append(cfg.Nodes, PKINodeSpec{NodeID: spec.NodeID, CellID: spec.Faction, VMID: spec.VMID, Host: finalHost(spec.NodeID), ManagementIP: spec.ManagementIP, RadioIP: fmt.Sprintf("10.77.0.%d", spec.VMID)})
		}
	}
	byCell := map[string][]PKINodeSpec{"A": {}, "B": {}}
	for _, spec := range cfg.Nodes {
		byCell[strings.ToUpper(spec.CellID)] = append(byCell[strings.ToUpper(spec.CellID)], spec)
	}
	for _, cellID := range []string{"A", "B"} {
		cellPublic, cellPrivate, _ := ed25519.GenerateKey(rand.Reader)
		cellTemplate := certificateTemplate("KeelMesh Cell "+cellID+" Intermediate", cfg.Now, cfg.Now.Add(cfg.ValidFor), true)
		cellTemplate.MaxPathLen = 0
		cellTemplate.MaxPathLenZero = true
		cellDER, createErr := x509.CreateCertificate(rand.Reader, cellTemplate, rootCert, cellPublic, rootPrivate)
		if createErr != nil {
			return createErr
		}
		cellCert, _ := x509.ParseCertificate(cellDER)
		cellDir := filepath.Join(cfg.OutputDir, "cells", strings.ToLower(cellID))
		if err := os.MkdirAll(cellDir, 0o700); err != nil {
			return err
		}
		trustPEM := append(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cellDER}), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootDER})...)
		if err := os.WriteFile(filepath.Join(cellDir, "cell-ca.crt"), trustPEM, 0o444); err != nil {
			return err
		}
		manifest := domain.CoordinationCellManifestV1{SchemaVersion: 1, CellID: cellID, ClusterID: "keelmesh-cell-" + strings.ToLower(cellID), Quorum: 4, IssuedAt: cfg.Now.UTC(), ExpiresAt: cfg.Now.Add(cfg.ValidFor).UTC(), TrustVersion: 1}
		if len(byCell[cellID]) != 6 {
			return fmt.Errorf("cell %s must contain exactly six PKI nodes", cellID)
		}
		for _, spec := range byCell[cellID] {
			nodeDir := filepath.Join(cfg.OutputDir, "nodes", spec.NodeID)
			if err := os.MkdirAll(nodeDir, 0o700); err != nil {
				return err
			}
			nodeURI := &url.URL{Scheme: "spiffe", Host: "keelmesh.local", Path: "/cell/" + strings.ToLower(cellID) + "/node/" + spec.NodeID}
			raftSerial, err := writeNodeCertificate(nodeDir, "raft", spec.NodeID+" radio", net.ParseIP(spec.RadioIP), nodeURI, cfg, cellCert, cellPrivate, cellDER)
			if err != nil {
				return err
			}
			managementSerial, err := writeNodeCertificate(nodeDir, "management", spec.NodeID+" management", net.ParseIP(spec.ManagementIP), nodeURI, cfg, cellCert, cellPrivate, cellDER)
			if err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(nodeDir, "cell-ca.crt"), trustPEM, 0o444); err != nil {
				return err
			}
			signingPublic, signingPrivate, _ := ed25519.GenerateKey(rand.Reader)
			if err := writePrivateKey(filepath.Join(nodeDir, "signing.key"), signingPrivate); err != nil {
				return err
			}
			manifest.Members = append(manifest.Members, domain.CoordinationCellMemberV1{NodeID: spec.NodeID, Faction: cellID, VMID: spec.VMID, Host: spec.Host, ManagementAddress: net.JoinHostPort(spec.ManagementIP, "7444"), RadioAddress: net.JoinHostPort(spec.RadioIP, "7443"), RaftTLSSerial: raftSerial, ManagementTLSSerial: managementSerial, SigningPublicKey: base64.StdEncoding.EncodeToString(signingPublic)})
		}
		if err := signManifest(&manifest, rootPrivate); err != nil {
			return err
		}
		manifestJSON, _ := json.MarshalIndent(manifest, "", "  ")
		manifestJSON = append(manifestJSON, '\n')
		if err := os.WriteFile(filepath.Join(cellDir, "manifest.json"), manifestJSON, 0o444); err != nil {
			return err
		}
		for _, member := range manifest.Members {
			if err := os.WriteFile(filepath.Join(cfg.OutputDir, "nodes", member.NodeID, "manifest.json"), manifestJSON, 0o444); err != nil {
				return err
			}
		}
	}

	refPublic, refPrivate, _ := ed25519.GenerateKey(rand.Reader)
	refTemplate := certificateTemplate("referee-214", cfg.Now, cfg.Now.Add(cfg.ValidFor), false)
	refTemplate.IPAddresses = []net.IP{net.ParseIP("192.168.50.214")}
	refTemplate.URIs = []*url.URL{{Scheme: "spiffe", Host: "keelmesh.local", Path: "/referee/referee-214"}}
	refTemplate.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	refDER, err := x509.CreateCertificate(rand.Reader, refTemplate, rootCert, refPublic, rootPrivate)
	if err != nil {
		return err
	}
	refDir := filepath.Join(cfg.OutputDir, "referee")
	if err := os.MkdirAll(refDir, 0o700); err != nil {
		return err
	}
	if err := writeCertificate(filepath.Join(refDir, "referee.crt"), refDER, 0o444); err != nil {
		return err
	}
	if err := writePrivateKey(filepath.Join(refDir, "referee.key"), refPrivate); err != nil {
		return err
	}
	refereeSigningPublic, refereeSigningKey, _ := ed25519.GenerateKey(rand.Reader)
	if err := writePrivateKey(filepath.Join(refDir, "signing.key"), refereeSigningKey); err != nil {
		return err
	}
	refereePublicPEM, err := marshalPublicKey(refereeSigningPublic)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(refDir, "signing.pub"), refereePublicPEM, 0o444); err != nil {
		return err
	}
	for _, spec := range cfg.Nodes {
		if err := os.WriteFile(filepath.Join(cfg.OutputDir, "nodes", spec.NodeID, "referee-signing.pub"), refereePublicPEM, 0o444); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(refDir, "root-ca.crt"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootDER}), 0o444)
}

func StagePKIRotation(currentDir, outputDir string, nodes []PKINodeSpec, overlap time.Duration) error {
	if currentDir == "" || outputDir == "" || currentDir == outputDir {
		return fmt.Errorf("distinct current and output PKI directories are required")
	}
	if overlap <= 0 {
		overlap = 15 * time.Minute
	}
	now := nowUTC()
	if err := GeneratePKI(PKIConfig{OutputDir: outputDir, ValidFor: 30 * 24 * time.Hour, Now: now, Nodes: nodes}); err != nil {
		return err
	}
	newRootKey, err := loadSigningKey(filepath.Join(outputDir, "root-ca.key"))
	if err != nil {
		return err
	}
	trustVersion := 1
	for _, cellID := range []string{"a", "b"} {
		currentManifest, err := readManifest(filepath.Join(currentDir, "cells", cellID, "manifest.json"))
		if err != nil {
			return err
		}
		stagedManifest, err := readManifest(filepath.Join(outputDir, "cells", cellID, "manifest.json"))
		if err != nil {
			return err
		}
		trustVersion = currentManifest.TrustVersion + 1
		stagedManifest.TrustVersion = trustVersion
		currentMembers := map[string]domain.CoordinationCellMemberV1{}
		for _, member := range currentManifest.Members {
			currentMembers[member.NodeID] = member
		}
		for index := range stagedManifest.Members {
			staged := &stagedManifest.Members[index]
			current, ok := currentMembers[staged.NodeID]
			if !ok {
				return fmt.Errorf("CELL_MEMBERSHIP_DENIED: rotation member %s is not in current manifest", staged.NodeID)
			}
			staged.PreviousRaftTLSSerials = appendUnique([]string{current.RaftTLSSerial}, current.PreviousRaftTLSSerials...)
			staged.PreviousManagementTLSSerials = appendUnique([]string{current.ManagementTLSSerial}, current.PreviousManagementTLSSerials...)
			currentSigningKey := filepath.Join(currentDir, "nodes", staged.NodeID, "signing.key")
			stagedSigningKey := filepath.Join(outputDir, "nodes", staged.NodeID, "signing.key")
			if err := copySecureFile(currentSigningKey, stagedSigningKey, 0o400); err != nil {
				return err
			}
			privateKey, err := loadSigningKey(stagedSigningKey)
			if err != nil {
				return err
			}
			staged.SigningPublicKey = base64.StdEncoding.EncodeToString(privateKey.Public().(ed25519.PublicKey))
		}
		if err := signManifest(&stagedManifest, newRootKey); err != nil {
			return err
		}
		manifestJSON, _ := json.MarshalIndent(stagedManifest, "", "  ")
		manifestJSON = append(manifestJSON, '\n')
		if err := os.WriteFile(filepath.Join(outputDir, "cells", cellID, "manifest.json"), manifestJSON, 0o444); err != nil {
			return err
		}
		for _, member := range stagedManifest.Members {
			if err := os.WriteFile(filepath.Join(outputDir, "nodes", member.NodeID, "manifest.json"), manifestJSON, 0o444); err != nil {
				return err
			}
		}
		currentTrust, err := os.ReadFile(filepath.Join(currentDir, "cells", cellID, "cell-ca.crt"))
		if err != nil {
			return err
		}
		stagedTrust, err := os.ReadFile(filepath.Join(outputDir, "cells", cellID, "cell-ca.crt"))
		if err != nil {
			return err
		}
		combinedTrust := append(stagedTrust, currentTrust...)
		if err := os.WriteFile(filepath.Join(outputDir, "cells", cellID, "cell-ca.crt"), combinedTrust, 0o444); err != nil {
			return err
		}
		for _, member := range stagedManifest.Members {
			if err := os.WriteFile(filepath.Join(outputDir, "nodes", member.NodeID, "cell-ca.crt"), combinedTrust, 0o444); err != nil {
				return err
			}
		}
	}
	for _, name := range []string{"signing.key", "signing.pub"} {
		mode := os.FileMode(0o444)
		if strings.HasSuffix(name, ".key") {
			mode = 0o400
		}
		if err := copySecureFile(filepath.Join(currentDir, "referee", name), filepath.Join(outputDir, "referee", name), mode); err != nil {
			return err
		}
	}
	for _, spec := range effectivePKINodes(nodes) {
		if err := copySecureFile(filepath.Join(currentDir, "referee", "signing.pub"), filepath.Join(outputDir, "nodes", spec.NodeID, "referee-signing.pub"), 0o444); err != nil {
			return err
		}
	}
	currentRoot, err := os.ReadFile(filepath.Join(currentDir, "referee", "root-ca.crt"))
	if err != nil {
		return err
	}
	stagedRoot, err := os.ReadFile(filepath.Join(outputDir, "referee", "root-ca.crt"))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outputDir, "referee", "root-ca.crt"), append(stagedRoot, currentRoot...), 0o444); err != nil {
		return err
	}
	receipt := PKIRotationReceipt{SchemaVersion: 1, TrustVersion: trustVersion, StagedAt: now, OverlapUntil: now.Add(overlap), PreviousPKI: filepath.Base(currentDir)}
	encoded, _ := json.MarshalIndent(receipt, "", "  ")
	return os.WriteFile(filepath.Join(outputDir, "rotation.json"), append(encoded, '\n'), 0o444)
}

func appendUnique(values []string, extras ...string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values)+len(extras))
	for _, value := range append(values, extras...) {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

func effectivePKINodes(nodes []PKINodeSpec) []PKINodeSpec {
	if len(nodes) > 0 {
		return nodes
	}
	result := make([]PKINodeSpec, 0, 12)
	for _, spec := range domain.VMFleetSpecs() {
		result = append(result, PKINodeSpec{NodeID: spec.NodeID, CellID: spec.Faction, VMID: spec.VMID, Host: finalHost(spec.NodeID), ManagementIP: spec.ManagementIP, RadioIP: fmt.Sprintf("10.77.0.%d", spec.VMID)})
	}
	return result
}

func copySecureFile(source, destination string, mode os.FileMode) error {
	encoded, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.WriteFile(destination, encoded, mode); err != nil {
		return err
	}
	return os.Chmod(destination, mode)
}

func certificateTemplate(name string, start, end time.Time, ca bool) *x509.Certificate {
	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, _ := rand.Int(rand.Reader, serialLimit)
	keyUsage := x509.KeyUsageDigitalSignature
	if ca {
		keyUsage |= x509.KeyUsageCertSign | x509.KeyUsageCRLSign
	}
	return &x509.Certificate{SerialNumber: serial, Subject: pkix.Name{CommonName: name, Organization: []string{"KeelMesh Demo"}}, NotBefore: start.Add(-time.Minute), NotAfter: end, BasicConstraintsValid: true, IsCA: ca, KeyUsage: keyUsage, SignatureAlgorithm: x509.PureEd25519}
}

func writeNodeCertificate(directory, profile, name string, ip net.IP, identityURI *url.URL, cfg PKIConfig, issuer *x509.Certificate, issuerKey ed25519.PrivateKey, issuerDER []byte) (string, error) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	template := certificateTemplate(name, cfg.Now, cfg.Now.Add(cfg.ValidFor), false)
	template.IPAddresses = []net.IP{ip}
	template.URIs = []*url.URL{identityURI}
	template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	der, err := x509.CreateCertificate(rand.Reader, template, issuer, publicKey, issuerKey)
	if err != nil {
		return "", err
	}
	chain := append(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: issuerDER})...)
	if err := os.WriteFile(filepath.Join(directory, profile+".crt"), chain, 0o444); err != nil {
		return "", err
	}
	if err := writePrivateKey(filepath.Join(directory, profile+".key"), privateKey); err != nil {
		return "", err
	}
	return template.SerialNumber.Text(16), nil
}

func writeCertificate(path string, der []byte, mode os.FileMode) error {
	return os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), mode)
}

func writePrivateKey(path string, key ed25519.PrivateKey) error {
	encoded, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	return os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encoded}), 0o400)
}

func marshalPublicKey(key ed25519.PublicKey) ([]byte, error) {
	encoded, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: encoded}), nil
}

func manifestPayload(manifest domain.CoordinationCellManifestV1) ([]byte, error) {
	manifest.Checksum = ""
	manifest.Signature = ""
	return json.Marshal(manifest)
}

func signManifest(manifest *domain.CoordinationCellManifestV1, rootKey ed25519.PrivateKey) error {
	payload, err := manifestPayload(*manifest)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(payload)
	manifest.Checksum = hex.EncodeToString(digest[:])
	manifest.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(rootKey, digest[:]))
	return nil
}

func VerifyManifest(manifest domain.CoordinationCellManifestV1, rootCertificate *x509.Certificate) error {
	payload, err := manifestPayload(manifest)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(payload)
	if !strings.EqualFold(manifest.Checksum, hex.EncodeToString(digest[:])) {
		return fmt.Errorf("CELL_MEMBERSHIP_DENIED: manifest checksum mismatch")
	}
	signature, err := base64.StdEncoding.DecodeString(manifest.Signature)
	if err != nil {
		return err
	}
	publicKey, ok := rootCertificate.PublicKey.(ed25519.PublicKey)
	if !ok || !ed25519.Verify(publicKey, digest[:], signature) {
		return fmt.Errorf("CELL_MEMBERSHIP_DENIED: invalid manifest signature")
	}
	return validateManifest(manifest)
}

func finalHost(nodeID string) string {
	switch nodeID {
	case "node-a-01", "node-a-02", "node-b-01", "node-b-04":
		return "fourtyfour"
	case "node-a-03", "node-a-06", "node-b-02", "node-b-03":
		return "mini42"
	default:
		return "mini43"
	}
}
