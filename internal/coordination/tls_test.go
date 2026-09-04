package coordination

import (
	"crypto/tls"
	"net"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func TestCoordinationTLSProfilesAndIdentityBoundaries(t *testing.T) {
	directory := t.TempDir()
	now := time.Now().UTC()
	if err := GeneratePKI(PKIConfig{OutputDir: directory, ValidFor: 24 * time.Hour, Now: now}); err != nil {
		t.Fatal(err)
	}
	manifestA, _ := readManifest(filepath.Join(directory, "cells", "a", "manifest.json"))
	manifestB, _ := readManifest(filepath.Join(directory, "cells", "b", "manifest.json"))
	identityA1 := identityForTest("node-a-01", "A", 220)
	identityA2 := identityForTest("node-a-02", "A", 221)
	identityB1 := identityForTest("node-b-01", "B", 229)
	serverA, _, err := loadNodeTLSConfigs(identityA1, manifestA, filepath.Join(directory, "nodes", "node-a-01", "raft.crt"), filepath.Join(directory, "nodes", "node-a-01", "raft.key"), filepath.Join(directory, "nodes", "node-a-01", "cell-ca.crt"), radioPlane, false)
	if err != nil {
		t.Fatal(err)
	}
	_, clientA, err := loadNodeTLSConfigs(identityA2, manifestA, filepath.Join(directory, "nodes", "node-a-02", "raft.crt"), filepath.Join(directory, "nodes", "node-a-02", "raft.key"), filepath.Join(directory, "nodes", "node-a-02", "cell-ca.crt"), radioPlane, false)
	if err != nil {
		t.Fatal(err)
	}
	clientA.ServerName = "10.77.0.220"
	if err := handshakePair(serverA, clientA); err != nil {
		t.Fatalf("same-cell radio handshake failed: %v", err)
	}

	_, clientB, err := loadNodeTLSConfigs(identityB1, manifestB, filepath.Join(directory, "nodes", "node-b-01", "raft.crt"), filepath.Join(directory, "nodes", "node-b-01", "raft.key"), filepath.Join(directory, "nodes", "node-b-01", "cell-ca.crt"), radioPlane, false)
	if err != nil {
		t.Fatal(err)
	}
	clientB.ServerName = "10.77.0.220"
	if err := handshakePair(serverA, clientB); err == nil {
		t.Fatalf("wrong-cell radio peer was not rejected before application handling: %v", err)
	}

	revokedManifest := manifestA
	revokedManifest.RevokedSerials = []string{manifestA.Members[1].RaftTLSSerial}
	revokedServer, _, err := loadNodeTLSConfigs(identityA1, revokedManifest, filepath.Join(directory, "nodes", "node-a-01", "raft.crt"), filepath.Join(directory, "nodes", "node-a-01", "raft.key"), filepath.Join(directory, "nodes", "node-a-01", "cell-ca.crt"), radioPlane, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := handshakePair(revokedServer, clientA); err == nil {
		t.Fatalf("revoked peer was accepted: %v", err)
	}

	expiredServer := serverA.Clone()
	expiredClient := clientA.Clone()
	expiredServer.Time = func() time.Time { return now.Add(48 * time.Hour) }
	expiredClient.Time = expiredServer.Time
	if err := handshakePair(expiredServer, expiredClient); err == nil {
		t.Fatal("expired certificates were accepted")
	}
}

func TestCoordinationTLSDeniesPlaintext(t *testing.T) {
	directory := t.TempDir()
	if err := GeneratePKI(PKIConfig{OutputDir: directory, ValidFor: 24 * time.Hour, Now: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	manifest, _ := readManifest(filepath.Join(directory, "cells", "a", "manifest.json"))
	serverConfig, _, err := loadNodeTLSConfigs(identityForTest("node-a-01", "A", 220), manifest, filepath.Join(directory, "nodes", "node-a-01", "raft.crt"), filepath.Join(directory, "nodes", "node-a-01", "raft.key"), filepath.Join(directory, "nodes", "node-a-01", "cell-ca.crt"), radioPlane, false)
	if err != nil {
		t.Fatal(err)
	}
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	done := make(chan error, 1)
	go func() { done <- tls.Server(serverConn, serverConfig).Handshake() }()
	_, _ = clientConn.Write([]byte("GET / HTTP/1.0\r\n\r\n"))
	clientConn.Close()
	if err := <-done; err == nil {
		t.Fatal("plaintext reached the coordination TLS listener")
	}
}

func identityForTest(nodeID, cell string, vmid int) domain.NodeIdentityV2 {
	host := strconv.Itoa(vmid)
	return domain.NodeIdentityV2{SchemaVersion: 2, NodeID: nodeID, CellID: cell, Faction: cell, VMID: vmid, ManagementIP: "192.168.50." + host, RadioIP: "10.77.0." + host}
}

func handshakePair(serverConfig, clientConfig *tls.Config) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()
	serverResult := make(chan error, 1)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			serverResult <- acceptErr
			return
		}
		defer connection.Close()
		serverResult <- tls.Server(connection, serverConfig).Handshake()
	}()
	rawClient, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		return err
	}
	client := tls.Client(rawClient, clientConfig)
	clientErr := client.Handshake()
	_ = client.Close()
	serverErr := <-serverResult
	if clientErr != nil {
		return clientErr
	}
	return serverErr
}
