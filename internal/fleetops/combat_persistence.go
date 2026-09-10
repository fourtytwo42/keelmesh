package fleetops

import (
	"context"
	"encoding/json"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// persistCombatAsync snapshots projections while the caller owns m.mu, then
// writes without extending the simulation lock.
func (m *Manager) persistCombatAsync(force bool) {
	m.persistCombatSnapshotAsync(force, false)
}

// resetCombatPersistenceAsync advances the persistence generation so an
// older queued snapshot cannot resurrect pre-reset entities or engagements.
func (m *Manager) resetCombatPersistenceAsync() {
	m.combatPersistSequence.Add(1)
	m.persistCombatSnapshotAsync(true, true)
}

func (m *Manager) persistCombatSnapshotAsync(force, replace bool) {
	if m.databaseURL == "" {
		return
	}
	now := time.Now()
	if !force && !m.combatLastPersist.IsZero() && now.Sub(m.combatLastPersist) < 5*time.Second {
		return
	}
	m.combatLastPersist = now
	entities := make([]domain.CombatEntityStateV1, 0, len(m.combatEntities))
	for _, entity := range m.combatEntities {
		entities = append(entities, entity)
	}
	engagements := make([]domain.EngagementProgramV1, 0, len(m.combatEngagements))
	for _, engagement := range m.combatEngagements {
		engagements = append(engagements, engagement)
	}
	repairs := make([]domain.RepairReceiptV1, 0, len(m.combatRepairs))
	for _, receipt := range m.combatRepairs {
		repairs = append(repairs, receipt)
	}
	events := append([]domain.CombatEventV1(nil), m.combatEvents...)
	worldTickMS, sequence, stateVersion := m.simTickMS, m.combatSequence, m.combatVersion
	generation := m.combatPersistSequence.Load()
	go func() {
		m.persistenceMu.Lock()
		defer m.persistenceMu.Unlock()
		if generation != m.combatPersistSequence.Load() {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		pool, err := pgxpool.New(ctx, m.databaseURL)
		if err != nil {
			m.logger.Warn("combat persistence unavailable", "error", err)
			return
		}
		defer pool.Close()
		if replace {
			if _, err := pool.Exec(ctx, `TRUNCATE combat_events, combat_repairs, combat_engagements, combat_entities, combat_runtime`); err != nil {
				m.logger.Warn("combat reset persistence failed", "error", err)
				return
			}
		}
		_, _ = pool.Exec(ctx, `INSERT INTO combat_runtime(id,world_tick_ms,sequence,state_version,updated_at) VALUES('primary',$1,$2,$3,$4) ON CONFLICT(id) DO UPDATE SET world_tick_ms=EXCLUDED.world_tick_ms,sequence=EXCLUDED.sequence,state_version=EXCLUDED.state_version,updated_at=EXCLUDED.updated_at WHERE combat_runtime.world_tick_ms <= EXCLUDED.world_tick_ms`, worldTickMS, sequence, stateVersion, now.UTC())
		for _, entity := range entities {
			payload, _ := json.Marshal(entity)
			_, _ = pool.Exec(ctx, `INSERT INTO combat_entities(entity_id,state_version,payload,updated_at) VALUES($1,$2,$3,$4) ON CONFLICT(entity_id) DO UPDATE SET state_version=EXCLUDED.state_version,payload=EXCLUDED.payload,updated_at=EXCLUDED.updated_at WHERE combat_entities.state_version <= EXCLUDED.state_version`, entity.EntityID, entity.StateVersion, payload, entity.UpdatedAt)
		}
		for _, engagement := range engagements {
			payload, _ := json.Marshal(engagement)
			_, _ = pool.Exec(ctx, `INSERT INTO combat_engagements(id,target_id,status,content_hash,state_version,payload,created_at) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(id) DO UPDATE SET status=EXCLUDED.status,state_version=EXCLUDED.state_version,payload=EXCLUDED.payload WHERE combat_engagements.state_version <= EXCLUDED.state_version`, engagement.ID, engagement.TargetID, engagement.Status, engagement.ContentHash, stateVersion, payload, engagement.CreatedAt)
		}
		for _, receipt := range repairs {
			payload, _ := json.Marshal(receipt)
			_, _ = pool.Exec(ctx, `INSERT INTO combat_repairs(id,vessel_id,idempotency_key,payload,created_at) VALUES($1,$2,$3,$4,$5) ON CONFLICT(id) DO NOTHING`, receipt.ID, receipt.VesselID, receipt.IdempotencyKey, payload, receipt.CreatedAt)
		}
		for _, event := range events {
			payload, _ := json.Marshal(event)
			_, _ = pool.Exec(ctx, `INSERT INTO combat_events(id,kind,entity_id,target_id,world_tick_ms,payload,created_at) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(id) DO NOTHING`, event.ID, event.Kind, event.EntityID, event.TargetID, event.WorldTickMS, payload, event.CreatedAt)
		}
	}()
}
