package domain

import "testing"

func TestVMFleetSpecsKeepTwoVotersPerCellPerHost(t *testing.T) {
	counts := map[string]map[string]int{}
	for _, spec := range VMFleetSpecs() {
		if counts[spec.Faction] == nil {
			counts[spec.Faction] = map[string]int{}
		}
		counts[spec.Faction][spec.Host]++
	}
	for _, cell := range []string{"A", "B"} {
		for _, host := range []string{"fourtyfour", "mini42", "mini43"} {
			if counts[cell][host] != 2 {
				t.Fatalf("cell %s has %d voters on %s, want 2", cell, counts[cell][host], host)
			}
		}
	}
}
