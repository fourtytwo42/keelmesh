package domain

// NodeFleetSpec is the single source of truth binding the twelve operating
// vessels to their provisioned VM nodes. Positions are deterministic open-water
// points in Rhode Island Sound; they are deliberately dispersed rather than
// arranged as operational groups.
type NodeFleetSpec struct {
	NodeID       string
	VesselID     string
	Faction      string
	VMID         int
	ManagementIP string
	Host         string
	Callsign     string
	ClassSlot    int
	Position     GeoPointV2
	HeadingDeg   float64
}

func VMFleetSpecs() []NodeFleetSpec {
	return []NodeFleetSpec{
		{"node-a-01", "vm-vessel-220", "A", 220, "192.168.50.220", "fourtyfour", "Gannet", 0, GeoPointV2{-71.50, 41.10}, 72},
		{"node-a-02", "vm-vessel-221", "A", 221, "192.168.50.221", "fourtyfour", "Osprey", 1, GeoPointV2{-71.40, 41.10}, 105},
		{"node-a-03", "vm-vessel-222", "A", 222, "192.168.50.222", "fourtyfour", "Tern", 2, GeoPointV2{-71.30, 41.10}, 48},
		{"node-a-04", "vm-vessel-223", "A", 223, "192.168.50.223", "fourtyfour", "Petrel", 3, GeoPointV2{-71.20, 41.10}, 315},
		{"node-a-05", "vm-vessel-224", "A", 224, "192.168.50.224", "fourtyfour", "Shearwater", 4, GeoPointV2{-71.10, 41.10}, 258},
		{"node-a-06", "vm-vessel-225", "A", 225, "192.168.50.225", "mini42", "Cormorant", 5, GeoPointV2{-71.45, 41.20}, 122},
		{"node-b-01", "vm-vessel-229", "B", 229, "192.168.50.229", "mini42", "Harrier", 0, GeoPointV2{-71.35, 41.20}, 80},
		{"node-b-02", "vm-vessel-231", "B", 231, "192.168.50.231", "mini42", "Kite", 1, GeoPointV2{-71.25, 41.20}, 196},
		{"node-b-03", "vm-vessel-232", "B", 232, "192.168.50.232", "mini42", "Merlin", 2, GeoPointV2{-71.15, 41.20}, 34},
		{"node-b-04", "vm-vessel-233", "B", 233, "192.168.50.233", "mini43", "Plover", 3, GeoPointV2{-71.42, 41.30}, 146},
		{"node-b-05", "vm-vessel-234", "B", 234, "192.168.50.234", "mini43", "Skua", 4, GeoPointV2{-71.30, 41.30}, 225},
		{"node-b-06", "vm-vessel-236", "B", 236, "192.168.50.236", "mini43", "Fulmar", 5, GeoPointV2{-71.15, 41.30}, 286},
	}
}
