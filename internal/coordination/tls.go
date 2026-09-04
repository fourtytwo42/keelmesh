package coordination

import (
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/hashicorp/raft"
)

type tlsStreamLayer struct {
	listener net.Listener
	client   *tls.Config
	address  raft.ServerAddress
}

func newTLSStreamLayer(address string, server, client *tls.Config) (*tlsStreamLayer, error) {
	listener, err := tls.Listen("tcp", address, server)
	if err != nil {
		return nil, err
	}
	return &tlsStreamLayer{listener: listener, client: client, address: raft.ServerAddress(address)}, nil
}

func (s *tlsStreamLayer) Dial(address raft.ServerAddress, timeout time.Duration) (net.Conn, error) {
	host, _, err := net.SplitHostPort(string(address))
	if err != nil {
		return nil, err
	}
	config := s.client.Clone()
	config.ServerName = host
	dialer := &net.Dialer{Timeout: timeout}
	return tls.DialWithDialer(dialer, "tcp", string(address), config)
}

func (s *tlsStreamLayer) Accept() (net.Conn, error) { return s.listener.Accept() }
func (s *tlsStreamLayer) Close() error              { return s.listener.Close() }
func (s *tlsStreamLayer) Addr() net.Addr            { return s.listener.Addr() }

func loadTLSConfig(identity domain.NodeIdentityV2, manifest domain.CoordinationCellManifestV1, certFile, keyFile, caFile string, server bool) (*tls.Config, error) {
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load node certificate: %w", err)
	}
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read cell trust bundle: %w", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("cell trust bundle contains no certificates")
	}
	allowed := map[string]domain.CoordinationCellMemberV1{}
	for _, member := range manifest.Members {
		allowed[member.NodeID] = member
	}
	verify := func(connection tls.ConnectionState) error {
		if len(connection.PeerCertificates) == 0 {
			return fmt.Errorf("PEER_IDENTITY_INVALID: peer certificate missing")
		}
		leaf := connection.PeerCertificates[0]
		nodeID, cellID := identityFromURIs(leaf.URIs)
		if nodeID == "" || cellID != identity.CellID {
			return fmt.Errorf("CELL_MEMBERSHIP_DENIED: peer identity is not in cell %s", identity.CellID)
		}
		member, ok := allowed[nodeID]
		if !ok {
			return fmt.Errorf("CELL_MEMBERSHIP_DENIED: node %s is not a voter", nodeID)
		}
		peerIP := member.RadioAddress
		if server {
			peerIP = member.ManagementAddress
		}
		peerIP = strings.Split(peerIP, ":")[0]
		if peerIP != "" && !containsIP(leaf.IPAddresses, net.ParseIP(peerIP)) {
			return fmt.Errorf("PEER_IDENTITY_INVALID: certificate IP does not match manifest")
		}
		return nil
	}
	config := &tls.Config{
		MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{certificate},
		RootCAs: roots, ClientCAs: roots, ClientAuth: tls.RequireAndVerifyClientCert,
		VerifyConnection: verify,
	}
	return config, nil
}

func identityFromURIs(values []*url.URL) (string, string) {
	for _, value := range values {
		if value == nil || value.Scheme != "spiffe" || value.Host != "keelmesh.local" {
			continue
		}
		parts := strings.Split(strings.Trim(value.Path, "/"), "/")
		if len(parts) == 4 && parts[0] == "cell" && parts[2] == "node" {
			return parts[3], strings.ToUpper(parts[1])
		}
	}
	return "", ""
}

func containsIP(values []net.IP, expected net.IP) bool {
	if expected == nil {
		return false
	}
	for _, value := range values {
		if value.Equal(expected) {
			return true
		}
	}
	return false
}

func loadSigningKey(path string) (ed25519.PrivateKey, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(encoded)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("invalid application signing key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	privateKey, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("application signing key is not Ed25519")
	}
	return privateKey, nil
}

func listenTLS(ctx context.Context, address string, config *tls.Config) (net.Listener, error) {
	listener, err := tls.Listen("tcp", address, config)
	if err != nil {
		return nil, err
	}
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()
	return listener, nil
}
