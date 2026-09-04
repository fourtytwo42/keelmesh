package coordination

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

func (m *Manager) ReloadCredentials() error {
	if m.cfg.Mode == ModeSimulated || m.cfg.ManifestFile == "" {
		return nil
	}
	manifest, err := LoadManifest(m.cfg.ManifestFile, m.cfg.TrustBundleFile, false)
	if err != nil {
		return err
	}
	if !sameFixedMembership(m.cfg.Manifest, manifest) {
		return fmt.Errorf("CELL_MEMBERSHIP_DENIED: certificate reload cannot change fixed membership")
	}
	raftServer, raftClient, err := loadNodeTLSConfigs(m.cfg.Identity, manifest, m.cfg.RaftCertificateFile, m.cfg.RaftTLSKeyFile, m.cfg.TrustBundleFile, radioPlane, false)
	if err != nil {
		return err
	}
	managementServer, managementClient, err := loadNodeTLSConfigs(m.cfg.Identity, manifest, m.cfg.ManagementCertificateFile, m.cfg.ManagementTLSKeyFile, m.cfg.TrustBundleFile, managementPlane, true)
	if err != nil {
		return err
	}
	signKey, err := loadSigningKey(m.cfg.SigningKeyFile)
	if err != nil {
		return err
	}
	refereeKey, err := loadSigningPublicKey(m.cfg.RefereeSigningPublicKeyFile)
	if err != nil {
		return err
	}
	m.raftTLS.replace(raftServer, raftClient)
	if m.managementTLS != nil {
		m.managementTLS.replace(managementServer, managementClient)
	}
	updatedClient := &http.Client{Transport: &http.Transport{TLSClientConfig: managementClient, MaxIdleConns: 12, MaxIdleConnsPerHost: 2, IdleConnTimeout: 30 * time.Second}, Timeout: m.cfg.ApplyTimeout + 2*time.Second}
	m.mu.Lock()
	oldClient := m.client
	m.cfg.Manifest = manifest
	m.signKey = signKey
	m.refereeKey = refereeKey
	m.client = updatedClient
	m.mu.Unlock()
	if oldClient != nil {
		oldClient.CloseIdleConnections()
	}
	return nil
}

func sameFixedMembership(left, right domain.CoordinationCellManifestV1) bool {
	if left.CellID != right.CellID || left.ClusterID != right.ClusterID || left.Quorum != right.Quorum || len(left.Members) != len(right.Members) {
		return false
	}
	canonical := func(manifest domain.CoordinationCellManifestV1) []string {
		values := make([]string, 0, len(manifest.Members))
		for _, member := range manifest.Members {
			values = append(values, fmt.Sprintf("%s|%s|%d|%s|%s", member.NodeID, member.Faction, member.VMID, member.ManagementAddress, member.RadioAddress))
		}
		sort.Strings(values)
		return values
	}
	leftValues, rightValues := canonical(left), canonical(right)
	for index := range leftValues {
		if leftValues[index] != rightValues[index] {
			return false
		}
	}
	return true
}

func (g *Gateway) ReloadCredentials() error {
	if g.cfg.Mode == ModeSimulated {
		return nil
	}
	manifests := make(map[string]domain.CoordinationCellManifestV1, len(g.cfg.ManifestFiles))
	for cellID, path := range g.cfg.ManifestFiles {
		manifest, err := LoadManifest(path, g.cfg.TrustBundleFile, true)
		if err != nil {
			return err
		}
		if previous, ok := g.cfg.Manifests[cellID]; ok && !sameFixedMembership(previous, manifest) {
			return fmt.Errorf("CELL_MEMBERSHIP_DENIED: gateway reload cannot change fixed membership")
		}
		manifests[cellID] = manifest
	}
	updated := g.cfg
	updated.Manifests = manifests
	tlsConfig, err := loadRefereeClientTLS(updated)
	if err != nil {
		return err
	}
	signKey, err := loadSigningKey(updated.SigningKeyFile)
	if err != nil {
		return err
	}
	transport := &http.Transport{TLSClientConfig: tlsConfig, MaxIdleConns: 24, MaxIdleConnsPerHost: 2, IdleConnTimeout: 30 * time.Second}
	g.mu.Lock()
	oldClient := g.client
	g.cfg = updated
	g.signKey = signKey
	g.client = &http.Client{Transport: transport, Timeout: updated.OperationTimeout}
	g.mu.Unlock()
	if oldClient != nil {
		oldClient.CloseIdleConnections()
	}
	return nil
}
