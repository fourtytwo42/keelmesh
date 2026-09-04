package coordination

import (
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/hashicorp/raft"
)

type inmemNode struct {
	id        raft.ServerID
	address   raft.ServerAddress
	raft      *raft.Raft
	transport *raft.InmemTransport
	fsm       *stateMachine
}

func TestSixVoterClusterConvergesAndMajorityReelects(t *testing.T) {
	nodes := make([]*inmemNode, 0, 6)
	for index := 0; index < 6; index++ {
		id := raft.ServerID(fmt.Sprintf("node-a-%02d", index+1))
		address, transport := raft.NewInmemTransport(raft.ServerAddress(id))
		config := raft.DefaultConfig()
		config.LocalID = id
		config.HeartbeatTimeout = 80 * time.Millisecond
		config.ElectionTimeout = 80 * time.Millisecond
		config.LeaderLeaseTimeout = 50 * time.Millisecond
		config.CommitTimeout = 10 * time.Millisecond
		config.LogOutput = io.Discard
		fsm := newStateMachine()
		nodeRaft, err := raft.NewRaft(config, fsm, raft.NewInmemStore(), raft.NewInmemStore(), raft.NewInmemSnapshotStore(), transport)
		if err != nil {
			t.Fatal(err)
		}
		nodes = append(nodes, &inmemNode{id: id, address: address, raft: nodeRaft, transport: transport, fsm: fsm})
	}
	defer func() {
		for _, node := range nodes {
			_ = node.raft.Shutdown().Error()
		}
	}()
	connectAll(nodes)
	configuration := raft.Configuration{}
	for _, node := range nodes {
		configuration.Servers = append(configuration.Servers, raft.Server{ID: node.id, Address: node.address, Suffrage: raft.Voter})
	}
	if err := nodes[0].raft.BootstrapCluster(configuration).Error(); err != nil {
		t.Fatal(err)
	}
	leader := waitForLeader(t, nodes, 3*time.Second, "")
	applyDirect(t, leader, testCommand(t, "epoch-1", "epoch-1", "coordination.epoch_advance", 1, map[string]int{"term": 1}))
	applyDirect(t, leader, testCommand(t, "mission-1", "mission-1", "mission.create", 1, map[string]string{"name": "Sound Watch"}))
	waitForConvergence(t, nodes, "mission-1", 3*time.Second)

	oldLeaderID := string(leader.id)
	for _, node := range nodes {
		if node == leader {
			continue
		}
		node.transport.Disconnect(leader.address)
		leader.transport.Disconnect(node.address)
	}
	newLeader := waitForLeader(t, nodes, 4*time.Second, oldLeaderID)
	if newLeader.id == leader.id {
		t.Fatal("isolated leader retained leadership")
	}
}

func TestSixVoterClusterRejectsThreeThreePartition(t *testing.T) {
	nodes := makeTestCluster(t, 6)
	defer shutdownTestCluster(nodes)
	connectAll(nodes)
	configuration := raft.Configuration{}
	for _, node := range nodes {
		configuration.Servers = append(configuration.Servers, raft.Server{ID: node.id, Address: node.address, Suffrage: raft.Voter})
	}
	if err := nodes[0].raft.BootstrapCluster(configuration).Error(); err != nil {
		t.Fatal(err)
	}
	leader := waitForLeader(t, nodes, 3*time.Second, "")
	applyDirect(t, leader, testCommand(t, "epoch-1", "epoch-1", "coordination.epoch_advance", 1, map[string]int{"term": 1}))
	left := []*inmemNode{leader}
	for _, node := range nodes {
		if node != leader && len(left) < 3 {
			left = append(left, node)
		}
	}
	right := make([]*inmemNode, 0, 3)
	for _, node := range nodes {
		included := false
		for _, candidate := range left {
			included = included || candidate == node
		}
		if !included {
			right = append(right, node)
		}
	}
	for _, a := range left {
		for _, b := range right {
			a.transport.Disconnect(b.address)
			b.transport.Disconnect(a.address)
		}
	}
	encoded, _ := json.Marshal(testCommand(t, "must-not-commit", "must-not-commit", "mission.create", 1, map[string]string{"name": "No quorum"}))
	if err := leader.raft.Apply(encoded, 350*time.Millisecond).Error(); err == nil {
		t.Fatal("three voters committed a write that requires four")
	}
	for _, node := range nodes {
		if _, found := node.fsm.receipt("must-not-commit"); found {
			t.Fatalf("node %s applied an uncommitted 3/3-partition write", node.id)
		}
	}
}

func makeTestCluster(t *testing.T, count int) []*inmemNode {
	t.Helper()
	nodes := make([]*inmemNode, 0, count)
	for index := 0; index < count; index++ {
		id := raft.ServerID(fmt.Sprintf("node-a-%02d", index+1))
		address, transport := raft.NewInmemTransport(raft.ServerAddress(id))
		config := raft.DefaultConfig()
		config.LocalID = id
		config.HeartbeatTimeout = 80 * time.Millisecond
		config.ElectionTimeout = 80 * time.Millisecond
		config.LeaderLeaseTimeout = 50 * time.Millisecond
		config.CommitTimeout = 10 * time.Millisecond
		config.LogOutput = io.Discard
		fsm := newStateMachine()
		nodeRaft, err := raft.NewRaft(config, fsm, raft.NewInmemStore(), raft.NewInmemStore(), raft.NewInmemSnapshotStore(), transport)
		if err != nil {
			t.Fatal(err)
		}
		nodes = append(nodes, &inmemNode{id: id, address: address, raft: nodeRaft, transport: transport, fsm: fsm})
	}
	return nodes
}

func shutdownTestCluster(nodes []*inmemNode) {
	for _, node := range nodes {
		_ = node.raft.Shutdown().Error()
	}
}

func connectAll(nodes []*inmemNode) {
	for _, source := range nodes {
		for _, target := range nodes {
			if source != target {
				source.transport.Connect(target.address, target.transport)
			}
		}
	}
}

func waitForLeader(t *testing.T, nodes []*inmemNode, timeout time.Duration, excluded string) *inmemNode {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, node := range nodes {
			if string(node.id) != excluded && node.raft.State() == raft.Leader {
				return node
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("leader was not elected")
	return nil
}

func applyDirect(t *testing.T, node *inmemNode, command interface{}) {
	t.Helper()
	encoded, err := json.Marshal(command)
	if err != nil {
		t.Fatal(err)
	}
	future := node.raft.Apply(encoded, time.Second)
	if err := future.Error(); err != nil {
		t.Fatal(err)
	}
	if result := future.Response().(applyResult); result.err != nil {
		t.Fatal(result.err)
	}
}

func waitForConvergence(t *testing.T, nodes []*inmemNode, commandID string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		matched := 0
		for _, node := range nodes {
			if _, ok := node.fsm.receipt(commandID); ok {
				matched++
			}
		}
		if matched == len(nodes) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("cluster did not converge")
}
