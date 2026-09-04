package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/coordination"
	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func main() {
	pkiDir := flag.String("pki", "", "path to generated local coordination PKI")
	stateFile := flag.String("state", "", "path for durable gateway test state")
	runID := flag.String("run-id", "baseline", "unique suffix for this verification transaction")
	timeout := flag.Duration("timeout", 12*time.Second, "overall verification timeout")
	flag.Parse()
	if *pkiDir == "" || *stateFile == "" {
		fmt.Fprintln(os.Stderr, "--pki and --state are required")
		os.Exit(2)
	}
	root := filepath.Join(*pkiDir, "referee", "root-ca.crt")
	manifestA, err := coordination.LoadManifest(filepath.Join(*pkiDir, "cells", "a", "manifest.json"), root, false)
	if err != nil {
		fail(err)
	}
	manifestB, err := coordination.LoadManifest(filepath.Join(*pkiDir, "cells", "b", "manifest.json"), root, false)
	if err != nil {
		fail(err)
	}
	gateway, err := coordination.NewGateway(coordination.GatewayConfig{Mode: coordination.ModeRaft, Manifests: map[string]domain.CoordinationCellManifestV1{"A": manifestA, "B": manifestB}, CertificateFile: filepath.Join(*pkiDir, "referee", "referee.crt"), TLSKeyFile: filepath.Join(*pkiDir, "referee", "referee.key"), TrustBundleFile: root, SigningKeyFile: filepath.Join(*pkiDir, "referee", "signing.key"), OperationTimeout: 3 * time.Second, StateFile: *stateFile})
	if err != nil {
		fail(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	result := map[string]any{"cells": map[string]any{}}
	for _, cell := range []string{"A", "B"} {
		leader, err := gateway.DiscoverLeader(ctx, cell)
		if err != nil {
			debugContext, debugCancel := context.WithTimeout(context.Background(), 4*time.Second)
			_ = json.NewEncoder(os.Stderr).Encode(gateway.Cells(debugContext))
			debugCancel()
			fail(err)
		}
		command, _ := gateway.CanonicalCommand(cell, "lab-command-"+*runID+"-"+cell, "lab-request-"+*runID+"-"+cell, "lab-idempotency-"+*runID+"-"+cell, "lab-operator", "mission.revise", "lab-mission-"+cell, 1, map[string]any{"name": "Local Cell " + cell, "run": *runID}, nil)
		receipt, proof, err := gateway.Commit(ctx, command)
		if err != nil {
			fail(err)
		}
		if err := gateway.AcceptEffect(proof); err != nil {
			fail(err)
		}
		result["cells"].(map[string]any)[cell] = map[string]any{"leader": leader.NodeID, "term": receipt.Term, "index": receipt.LogIndex, "signatures": len(proof.Acknowledgements), "state_hash": receipt.ResultingStateHash}
		manifest := manifestA
		if cell == "B" {
			manifest = manifestB
		}
		var followerID string
		var forwardedProof domain.QuorumCommitProofV1
		var forwardErr error
		for _, member := range manifest.Members {
			if member.NodeID == leader.NodeID {
				continue
			}
			candidate, _ := gateway.CanonicalCommand(cell, "lab-forwarded-"+*runID+"-"+cell, "lab-forwarded-request-"+*runID+"-"+cell, "lab-forwarded-idempotency-"+*runID+"-"+cell, "lab-operator", "mission.revise", "lab-forwarded-mission-"+cell, 1, map[string]any{"forwarded": true, "run": *runID}, nil)
			_, forwardedProof, forwardErr = gateway.CommitViaNode(ctx, candidate, member.NodeID)
			if forwardErr == nil {
				followerID = member.NodeID
				break
			}
		}
		if forwardErr != nil || followerID == "" {
			fail(fmt.Errorf("follower forwarding: %w", forwardErr))
		}
		result["cells"].(map[string]any)[cell].(map[string]any)["forwarded_via"] = followerID
		result["cells"].(map[string]any)[cell].(map[string]any)["forwarded_signatures"] = len(forwardedProof.Acknowledgements)
	}
	activation := time.Now().UTC().Add(10 * time.Second)
	operation, err := gateway.CrossCell(ctx, "lab-cross-cell-"+*runID, "lab-cross-request-"+*runID, "lab-cross-idempotency-"+*runID, "lab-operator", "mission.start", "lab-shared-mission", 1, map[string]any{"mission": "lab-shared-mission", "run": *runID}, activation)
	if err != nil {
		fail(err)
	}
	result["cross_cell"] = map[string]any{"state": operation.State, "activation_at": operation.ActivationAt, "prepare_cells": len(operation.PrepareProofs)}
	_ = json.NewEncoder(os.Stdout).Encode(result)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
