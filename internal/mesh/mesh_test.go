package mesh

import "testing"

func TestRelayAndPartition(t *testing.T) {
	links := FailDirect(Healthy(), 0)
	if p := RelayPath(links); len(p) != 3 || p[1] != "vessel-03" {
		t.Fatalf("path=%v", p)
	}
	links = PartitionV4(links, 30)
	if RelayPath(links) != nil {
		t.Fatal("partition still routable")
	}
}

func TestBundleValidationAndDedupFailClosed(t *testing.T) {
	key := []byte("mesh-key")
	bundle := NewBundle("bundle", "once", "mission", "plan", "payload", 0, key)
	if err := ValidateBundle(bundle, 0, 2, key); err != nil {
		t.Fatal(err)
	}
	dedup := NewDeduplicator()
	if accepted, _ := dedup.Deliver(bundle); !accepted {
		t.Fatal("first delivery rejected")
	}
	if accepted, _ := dedup.Deliver(bundle); accepted {
		t.Fatal("duplicate executed")
	}
	if err := ValidateBundle(bundle, 31, 2, key); err == nil {
		t.Fatal("expired bundle accepted")
	}
	if err := ValidateBundle(bundle, 0, 4, key); err == nil {
		t.Fatal("excessive hop accepted")
	}
	mutated := bundle
	mutated.DestinationID = "vessel-06"
	if err := ValidateBundle(mutated, 0, 2, key); err == nil {
		t.Fatal("relay mutation accepted")
	}
}
