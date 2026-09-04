package coordination

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGeneratePKIProducesSignedSixMemberCells(t *testing.T) {
	directory := t.TempDir()
	if err := GeneratePKI(PKIConfig{OutputDir: directory, ValidFor: 24 * time.Hour, Now: time.Unix(1000, 0).UTC()}); err != nil {
		t.Fatal(err)
	}
	rootPEM, err := os.ReadFile(filepath.Join(directory, "root-ca.crt"))
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(rootPEM)
	root, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	for _, cell := range []string{"a", "b"} {
		manifest, err := readManifest(filepath.Join(directory, "cells", cell, "manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		if err := VerifyManifest(manifest, root); err != nil {
			t.Fatal(err)
		}
		if len(manifest.Members) != 6 || manifest.Quorum != 4 {
			t.Fatalf("unexpected cell shape: %#v", manifest)
		}
	}
	keyInfo, err := os.Stat(filepath.Join(directory, "nodes", "node-a-01", "raft.key"))
	if err != nil {
		t.Fatal(err)
	}
	if keyInfo.Mode().Perm() != 0o400 {
		t.Fatalf("node key mode = %v", keyInfo.Mode().Perm())
	}
	managementCertificate, err := os.ReadFile(filepath.Join(directory, "nodes", "node-a-01", "management.crt"))
	if err != nil {
		t.Fatal(err)
	}
	leafBlock, remainder := pem.Decode(managementCertificate)
	chainBlock, _ := pem.Decode(remainder)
	if leafBlock == nil || chainBlock == nil {
		t.Fatal("management certificate does not include its intermediate chain")
	}
}

func TestStagePKIRotationPreservesMembershipSigningIdentityAndTrustOverlap(t *testing.T) {
	root := t.TempDir()
	current := filepath.Join(root, "current")
	staged := filepath.Join(root, "staged")
	now := time.Now().UTC()
	if err := GeneratePKI(PKIConfig{OutputDir: current, ValidFor: 24 * time.Hour, Now: now}); err != nil {
		t.Fatal(err)
	}
	if err := GeneratePKI(PKIConfig{OutputDir: current, ValidFor: 24 * time.Hour, Now: now}); err == nil {
		t.Fatal("PKI initializer overwrote an existing authority")
	}
	if err := StagePKIRotation(current, staged, nil, 10*time.Minute); err != nil {
		t.Fatal(err)
	}
	oldManifest, _ := readManifest(filepath.Join(current, "cells", "a", "manifest.json"))
	newManifest, _ := readManifest(filepath.Join(staged, "cells", "a", "manifest.json"))
	if newManifest.TrustVersion != oldManifest.TrustVersion+1 || !sameFixedMembership(oldManifest, newManifest) {
		t.Fatalf("rotation changed fixed membership or trust version: old=%+v new=%+v", oldManifest, newManifest)
	}
	if newManifest.Members[0].SigningPublicKey != oldManifest.Members[0].SigningPublicKey || len(newManifest.Members[0].PreviousRaftTLSSerials) == 0 || len(newManifest.Members[0].PreviousManagementTLSSerials) == 0 {
		t.Fatal("rotation did not preserve application identity and previous TLS serials")
	}
	if err := verifyManifestTrust(newManifest, filepath.Join(staged, "nodes", "node-a-01", "cell-ca.crt")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(staged, "rotation.json")); err != nil {
		t.Fatal(err)
	}
}
