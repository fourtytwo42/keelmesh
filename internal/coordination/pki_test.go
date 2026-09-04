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
