package api

import (
	"testing"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func TestValidateCoordinatedRequestRejectsUnknownFieldsBeforeAppend(t *testing.T) {
	body := []byte(`{"request_id":"request-1","idempotency_key":"key-1","expected_version":1,"name":"Blue","color":"amber","pattern":"solid","member_ids":[],"unexpected":true}`)
	if err := validateCoordinatedRequest("POST", "/api/v2/groups", body); err == nil {
		t.Fatal("expected unknown coordinated field to fail validation")
	}
}

func TestCoordinatorManagementUIUsesAdvertisedManagementHost(t *testing.T) {
	advertisement := domain.CoordinatorAdvertisementV1{CellID: "B", NodeID: "node-b-04", AuthorityEpoch: 7, Term: 9, ManagementURL: "https://192.168.50.233:7444", CommitIndex: 42, ExpiresAt: time.Now().Add(time.Minute), State: "ready"}
	if got := coordinatorManagementUI(advertisement.ManagementURL); got != "http://192.168.50.233:8080" {
		t.Fatalf("management UI = %q", got)
	}
	coordinator := coordinatorFromAdvertisement(advertisement)
	if coordinator.NodeID != "node-b-04" || coordinator.Epoch != 7 || coordinator.QuorumRequired != 4 || coordinator.State != "ready" {
		t.Fatalf("unexpected coordinator projection: %#v", coordinator)
	}
}

func TestValidateCoordinatedRequestAcceptsActorContracts(t *testing.T) {
	groupBody := []byte(`{"request_id":"request-1","idempotency_key":"key-1","expected_version":1,"actor_identity":"operator-1","name":"Blue","color":"amber","pattern":"solid","member_ids":[]}`)
	if err := validateCoordinatedRequest("POST", "/api/v2/groups", groupBody); err != nil {
		t.Fatalf("valid v2 group request rejected: %v", err)
	}
	arenaBody := []byte(`{"request_id":"request-2","idempotency_key":"key-2","expected_version":1,"actor_id":"player-a"}`)
	if err := validateCoordinatedRequest("POST", "/api/v3/matches", arenaBody); err != nil {
		t.Fatalf("valid v3 arena request rejected: %v", err)
	}
}

func TestValidateCoordinatedRequestRejectsTrailingObjectsAndUnknownRoutes(t *testing.T) {
	if err := validateCoordinatedRequest("DELETE", "/api/v2/missions/mission-1", []byte(`{"request_id":"r","idempotency_key":"k","expected_version":1}{}`)); err == nil {
		t.Fatal("expected trailing object to fail validation")
	}
	if err := validateCoordinatedRequest("POST", "/api/v2/groups/group-1/unknown", []byte(`{}`)); err == nil {
		t.Fatal("expected unsupported route to fail validation")
	}
}
