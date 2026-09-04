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

	byCell := map[string][]domain.NodeFleetSpec{"A": {}, "B": {}}
	for _, spec := range domain.VMFleetSpecs() {
		byCell[spec.Faction] = append(byCell[spec.Faction], spec)
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
		for _, spec := range byCell[cellID] {
			nodeDir := filepath.Join(cfg.OutputDir, "nodes", spec.NodeID)
			if err := os.MkdirAll(nodeDir, 0o700); err != nil {
				return err
			}
			nodeURI := &url.URL{Scheme: "spiffe", Host: "keelmesh.local", Path: "/cell/" + strings.ToLower(cellID) + "/node/" + spec.NodeID}
			raftSerial, err := writeNodeCertificate(nodeDir, "raft", spec.NodeID+" radio", net.ParseIP(fmt.Sprintf("10.77.0.%d", spec.VMID)), nodeURI, cfg, cellCert, cellPrivate, cellDER)
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
			manifest.Members = append(manifest.Members, domain.CoordinationCellMemberV1{NodeID: spec.NodeID, Faction: cellID, VMID: spec.VMID, Host: finalHost(spec.NodeID), ManagementAddress: fmt.Sprintf("%s:7444", spec.ManagementIP), RadioAddress: fmt.Sprintf("10.77.0.%d:7443", spec.VMID), RaftTLSSerial: raftSerial, ManagementTLSSerial: managementSerial, SigningPublicKey: base64.StdEncoding.EncodeToString(signingPublic)})
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
	for _, spec := range domain.VMFleetSpecs() {
		if err := os.WriteFile(filepath.Join(cfg.OutputDir, "nodes", spec.NodeID, "referee-signing.pub"), refereePublicPEM, 0o444); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(refDir, "root-ca.crt"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootDER}), 0o444)
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
