package trajectory

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/fourtytwo42/keelmesh/internal/domain"
)

// NewFullProgram creates the complete finite authority artifact installed on
// assigned vessels. A loop is bounded by the mission's authorized duration;
// non-looping work expires when its planned route completes.
func NewFullProgram(mission domain.MissionWorkspaceV2, plan domain.FleetPlanV2, lease domain.FleetLeaseV2, revision domain.TrajectoryRevisionV1, issuedTick int64, key []byte) domain.TrajectoryProgramV2 {
	expiryTick := revision.ActivationTick + int64(revision.DurationS)
	completion := "hold_at_final_point"
	if mission.Loop {
		completion = "loop_until_authority_expiry"
		maximum := int64(math.Ceil(mission.Constraints.MaximumDurationMinutes * 60))
		if maximum < int64(revision.DurationS) {
			maximum = int64(revision.DurationS)
		}
		expiryTick = revision.ActivationTick + maximum
	}
	program := domain.TrajectoryProgramV2{
		SchemaVersion: 2, ProgramID: fullProgramID(mission.ID, revision.Revision, revision.ContentHash), MissionID: mission.ID,
		PlanID: plan.ID, PlanHash: plan.ContentHash, LeaseID: lease.ID, IssuedAt: time.Now().UTC(), IssuedTick: issuedTick,
		AuthorizationExpiryTick: expiryTick, CompletionPolicy: completion, TerminalContingency: "safe_hold",
		ActiveRevision: revision.Revision, MissionTickMS: revision.ActivationTick * 1000,
		Revisions: map[int]domain.TrajectoryRevisionV1{revision.Revision: revision}, Cursors: map[string]domain.ExecutionCursorV2{}, LastAdaptations: map[string]domain.GroupAdaptationV1{},
	}
	UpdateFullProgramCursors(&program)
	program.ContentHash = fullProgramHash(program)
	program.Signature = sign(program.ContentHash, key)
	return program
}

func UpgradeLegacyProgram(mission domain.MissionWorkspaceV2, legacy domain.TrajectoryProgramV1, key []byte) (domain.TrajectoryProgramV2, error) {
	revision, ok := legacy.Revisions[legacy.ActiveRevision]
	if !ok || !ValidateRevision(revision, key) {
		return domain.TrajectoryProgramV2{}, fmt.Errorf("PROGRAM_HASH_MISMATCH: legacy active revision is invalid")
	}
	plan := domain.FleetPlanV2{ID: revision.PlanID, ContentHash: revision.PlanHash}
	lease := domain.FleetLeaseV2{ID: revision.LeaseID}
	program := NewFullProgram(mission, plan, lease, revision, revision.CreatedTick, key)
	program.MissionTickMS = legacy.MissionTickMS
	program.PendingRevision = legacy.PendingRevision
	program.ActivationTick = legacy.ActivationTick
	for number, candidate := range legacy.Revisions {
		if !ValidateRevision(candidate, key) {
			return domain.TrajectoryProgramV2{}, fmt.Errorf("PROGRAM_HASH_MISMATCH: legacy revision %d is invalid", number)
		}
		program.Revisions[number] = candidate
	}
	UpdateFullProgramCursors(&program)
	SignFullProgram(&program, key)
	return program, nil
}

func AddPendingFullProgram(program *domain.TrajectoryProgramV2, plan domain.FleetPlanV2, lease domain.FleetLeaseV2, mission domain.MissionWorkspaceV2, revision domain.TrajectoryRevisionV1, renewAuthority bool, key []byte) {
	program.Revisions[revision.Revision] = revision
	program.PendingRevision = revision.Revision
	program.ActivationTick = revision.ActivationTick
	program.PlanID = plan.ID
	program.PlanHash = plan.ContentHash
	program.InstallReceipts = nil
	if renewAuthority {
		program.LeaseID = lease.ID
		program.IssuedAt = time.Now().UTC()
		program.IssuedTick = program.MissionTickMS / 1000
		program.CompletionPolicy = "hold_at_final_point"
		program.AuthorizationExpiryTick = revision.ActivationTick + int64(revision.DurationS)
		if mission.Loop {
			program.CompletionPolicy = "loop_until_authority_expiry"
			maximum := int64(math.Ceil(mission.Constraints.MaximumDurationMinutes * 60))
			if maximum < int64(revision.DurationS) {
				maximum = int64(revision.DurationS)
			}
			program.AuthorizationExpiryTick = revision.ActivationTick + maximum
		}
	}
	program.ContentHash = fullProgramHash(*program)
	program.Signature = sign(program.ContentHash, key)
}

// ProgramInstallRevision identifies the newest complete revision carried by
// an installation payload, including a future-tick revision not yet active.
func ProgramInstallRevision(program domain.TrajectoryProgramV2) int {
	if program.PendingRevision > program.ActiveRevision {
		return program.PendingRevision
	}
	return program.ActiveRevision
}

func AdvanceFullProgram(program *domain.TrajectoryProgramV2, elapsedMS int64) (activated bool) {
	program.MissionTickMS += elapsedMS
	tick := program.MissionTickMS / 1000
	if program.PendingRevision > 0 && tick >= program.ActivationTick {
		program.ActiveRevision = program.PendingRevision
		program.PendingRevision = 0
		program.ActivationTick = 0
		activated = true
	}
	UpdateFullProgramCursors(program)
	return activated
}

func FullProgramCurrentSegment(program domain.TrajectoryProgramV2, vesselID string) (domain.TrajectorySegmentV2, bool) {
	tick := program.MissionTickMS / 1000
	if tick >= program.AuthorizationExpiryTick {
		return domain.TrajectorySegmentV2{}, false
	}
	revision, ok := program.Revisions[program.ActiveRevision]
	if !ok {
		return domain.TrajectorySegmentV2{}, false
	}
	for _, segment := range revision.Segments[vesselID] {
		if tick >= segment.ActivationTick && tick < segment.ExpiryTick {
			return segment, true
		}
	}
	return domain.TrajectorySegmentV2{}, false
}

func UpdateFullProgramCursors(program *domain.TrajectoryProgramV2) {
	revision, ok := program.Revisions[program.ActiveRevision]
	if !ok {
		return
	}
	tick := program.MissionTickMS / 1000
	for vesselID, segments := range revision.Segments {
		sequence := len(segments)
		lifecycle := "completed"
		for i, segment := range segments {
			if tick < segment.ExpiryTick {
				sequence = i
				lifecycle = "executing"
				break
			}
		}
		if tick >= program.AuthorizationExpiryTick {
			lifecycle = "expired"
		}
		remaining := max(0, revision.DurationS-int(tick-revision.ActivationTick))
		program.Cursors[vesselID] = domain.ExecutionCursorV2{
			VesselID: vesselID, Revision: revision.Revision, Sequence: sequence, MissionTick: tick,
			ProgramRemainingSeconds:        remaining,
			AuthorizedTimeRemainingSeconds: max64(0, program.AuthorizationExpiryTick-tick), Lifecycle: lifecycle,
		}
	}
}

func ValidateFullProgram(program domain.TrajectoryProgramV2, key []byte) error {
	if err := ValidateFullProgramContent(program); err != nil {
		return err
	}
	if !hmac.Equal([]byte(program.Signature), []byte(sign(program.ContentHash, key))) {
		return fmt.Errorf("PROGRAM_HASH_MISMATCH: full program signature does not match")
	}
	for number, revision := range program.Revisions {
		if !ValidateRevision(revision, key) {
			return fmt.Errorf("PROGRAM_HASH_MISMATCH: trajectory revision %d signature does not match", number)
		}
	}
	return nil
}

// ValidateFullProgramContent is the node-side structural verifier. The
// program's authority is established separately by the mTLS transport and
// quorum proof; this check guarantees that the installed bytes still match
// the immutable program hash.
func ValidateFullProgramContent(program domain.TrajectoryProgramV2) error {
	if program.SchemaVersion != 2 || program.ProgramID == "" || program.MissionID == "" || program.AuthorizationExpiryTick <= program.IssuedTick {
		return fmt.Errorf("PROGRAM_NOT_INSTALLED: full program identity or authorization boundary is invalid")
	}
	if program.ContentHash != fullProgramHash(program) {
		return fmt.Errorf("PROGRAM_HASH_MISMATCH: full program content does not match")
	}
	if _, ok := program.Revisions[program.ActiveRevision]; !ok {
		return fmt.Errorf("PROGRAM_HASH_MISMATCH: active trajectory revision is invalid")
	}
	if program.PendingRevision > 0 {
		if _, ok := program.Revisions[program.PendingRevision]; !ok {
			return fmt.Errorf("PROGRAM_HASH_MISMATCH: pending trajectory revision is invalid")
		}
	}
	return nil
}

func SignFullProgram(program *domain.TrajectoryProgramV2, key []byte) {
	program.ContentHash = fullProgramHash(*program)
	program.Signature = sign(program.ContentHash, key)
}

func FullProgramManifest(program domain.TrajectoryProgramV2) domain.ProgramManifestV1 {
	installRevision := ProgramInstallRevision(program)
	revision := program.Revisions[installRevision]
	vessels := make([]string, 0, len(revision.Segments))
	segments := 0
	for vesselID, list := range revision.Segments {
		vessels = append(vessels, vesselID)
		segments += len(list)
	}
	sort.Strings(vessels)
	return domain.ProgramManifestV1{SchemaVersion: 1, ProgramID: program.ProgramID, MissionID: program.MissionID, PlanHash: program.PlanHash, Revision: installRevision, VesselIDs: vessels, SegmentCount: segments, AuthorizationExpiryTick: program.AuthorizationExpiryTick, ContentHash: program.ContentHash}
}

func FullProgramSummary(program domain.TrajectoryProgramV2) domain.TrajectoryProgramSummaryV2 {
	revision := program.Revisions[program.ActiveRevision]
	manifest := FullProgramManifest(program)
	tick := program.MissionTickMS / 1000
	return domain.TrajectoryProgramSummaryV2{
		SchemaVersion: 2, ProgramID: program.ProgramID, MissionID: program.MissionID, PlanID: program.PlanID,
		ActiveRevision: program.ActiveRevision, PendingRevision: program.PendingRevision, ActivationTick: program.ActivationTick,
		MissionTick: tick, DurationSeconds: revision.DurationS, TotalSegments: manifest.SegmentCount,
		CompleteProgramOnboard: len(program.InstallReceipts) > 0, InstalledNodeCount: len(program.InstallReceipts), InstallationState: installationState(program), AuthorizationExpiryTick: program.AuthorizationExpiryTick,
		AuthorizedTimeRemainingSeconds: max64(0, program.AuthorizationExpiryTick-tick), CompletionPolicy: program.CompletionPolicy,
		TerminalContingency: program.TerminalContingency, Execution: program.Cursors, LastAdaptations: program.LastAdaptations,
		ContentHash: program.ContentHash,
	}
}

func installationState(program domain.TrajectoryProgramV2) string {
	if len(program.InstallReceipts) == 0 {
		return "not_installed"
	}
	return "installed"
}

func FullProgramView(program domain.TrajectoryProgramV2) domain.TrajectoryProgramViewV2 {
	return domain.TrajectoryProgramViewV2{Summary: FullProgramSummary(program), Manifest: FullProgramManifest(program), Program: program}
}

// LegacyView keeps v1/v2 clients wire-compatible for one release. Tape fields
// are deliberately zero and no executable segment window is exposed.
func LegacyView(program domain.TrajectoryProgramV2) domain.TrajectoryProgramViewV1 {
	summary := FullProgramSummary(program)
	execution := make(map[string]domain.ExecutionCursorV1, len(program.Cursors))
	for vesselID, cursor := range program.Cursors {
		execution[vesselID] = domain.ExecutionCursorV1{VesselID: vesselID, Revision: cursor.Revision, Sequence: cursor.Sequence, MissionTick: cursor.MissionTick, HotTapeDepthS: 0, ProgramRemainingS: cursor.ProgramRemainingSeconds, Lifecycle: cursor.Lifecycle}
	}
	return domain.TrajectoryProgramViewV1{Summary: domain.TrajectoryProgramSummaryV1{
		MissionID: summary.MissionID, ActiveRevision: summary.ActiveRevision, PendingRevision: summary.PendingRevision,
		ActivationTick: summary.ActivationTick, MissionTick: summary.MissionTick, DurationS: summary.DurationSeconds,
		TotalSegments: summary.TotalSegments, HotTapeHorizonS: 0, Execution: execution,
		ContentHash: summary.ContentHash,
	}, HotTape: map[string][]domain.TrajectorySegmentV2{}}
}

func FullProgramAuthority(program domain.TrajectoryProgramV2, vesselID, decisionNode, decisionScope string, decisionEpoch int64) domain.ExecutionAuthorityV1 {
	tick := program.MissionTickMS / 1000
	status := "active"
	if tick >= program.AuthorizationExpiryTick {
		status = "expired"
	} else if _, ok := program.Cursors[vesselID]; !ok {
		status = "not_assigned"
	}
	return domain.ExecutionAuthorityV1{
		SchemaVersion: 1, ProgramID: program.ProgramID, MissionID: program.MissionID, PlanID: program.PlanID,
		Revision: program.ActiveRevision, Status: status, CompleteProgramOnboard: len(program.InstallReceipts) > 0, IssuedTick: program.IssuedTick,
		AuthorizationExpiryTick: program.AuthorizationExpiryTick, AuthorizedTimeRemainingSeconds: max64(0, program.AuthorizationExpiryTick-tick),
		DecisionNodeID: decisionNode, DecisionScope: decisionScope, DecisionEpoch: decisionEpoch,
		TerminalContingency: program.TerminalContingency, ProgramHash: program.ContentHash,
	}
}

func fullProgramID(missionID string, revision int, revisionHash string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%s", missionID, revision, revisionHash)))
	return "program-" + hex.EncodeToString(sum[:8])
}

func fullProgramHash(value domain.TrajectoryProgramV2) string {
	// Execution cursors, leadership/adaptation receipts, and active/pending
	// lifecycle pointers are mutable runtime evidence. The signed authority hash
	// covers only the immutable program identity, limits, and signed revisions.
	return hashJSON(struct {
		SchemaVersion           int
		ProgramID               string
		MissionID               string
		PlanID                  string
		PlanHash                string
		LeaseID                 string
		IssuedAt                time.Time
		IssuedTick              int64
		AuthorizationExpiryTick int64
		CompletionPolicy        string
		TerminalContingency     string
		Revisions               map[int]domain.TrajectoryRevisionV1
	}{value.SchemaVersion, value.ProgramID, value.MissionID, value.PlanID, value.PlanHash, value.LeaseID, value.IssuedAt, value.IssuedTick, value.AuthorizationExpiryTick, value.CompletionPolicy, value.TerminalContingency, value.Revisions})
}

func hashJSON(value any) string {
	encoded, _ := json.Marshal(value)
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
