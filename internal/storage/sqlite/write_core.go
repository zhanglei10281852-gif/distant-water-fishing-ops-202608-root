package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/repository"
)

func (q *queries) InsertUser(ctx context.Context, user domain.User) error {
	if err := user.Validate(); err != nil {
		return err
	}
	_, err := q.q.ExecContext(ctx, `INSERT INTO users(id, email, display_name, password_hash, role, status, version, created_at, updated_at)
        VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`, user.ID, user.Email, user.DisplayName, user.PasswordHash, user.Role,
		user.Status, user.Version, formatTime(user.CreatedAt), formatTime(user.UpdatedAt))
	return translateError("insert user", err)
}

func (q *queries) InsertSession(ctx context.Context, session domain.Session) error {
	_, err := q.q.ExecContext(ctx, `INSERT INTO sessions(id, user_id, token_hash, expires_at, created_at, revoked_at)
        VALUES(?, ?, ?, ?, ?, NULL)`, session.ID, session.UserID, session.TokenHash, formatTime(session.ExpiresAt), formatTime(session.CreatedAt))
	return translateError("insert session", err)
}

func (q *queries) RevokeSession(ctx context.Context, sessionID string, revokedAt time.Time) error {
	result, err := q.q.ExecContext(ctx, `UPDATE sessions SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`, formatTime(revokedAt), sessionID)
	if err != nil {
		return translateError("revoke session", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke session rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("revoke session: %w", domain.ErrNotFound)
	}
	return nil
}

func (q *queries) InsertFishingPermit(ctx context.Context, program domain.FishingPermit) error {
	if err := program.Validate(); err != nil {
		return err
	}
	_, err := q.q.ExecContext(ctx, `INSERT INTO fishing_permits(id, code, name, status, minimum_catch_variance_grams, maximum_catch_variance_grams,
        max_voyage_duration_seconds, compliance_review_deadline_seconds, business_timezone, version, created_at, updated_at)
        VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, program.ID, program.Code, program.Name, program.Status,
		program.CatchVariance.Minimum, program.CatchVariance.Maximum, int64(program.MaxVoyageDuration/time.Second),
		int64(program.ComplianceReviewDeadline/time.Second), program.BusinessTimezone, program.Version,
		formatTime(program.CreatedAt), formatTime(program.UpdatedAt))
	return translateError("insert program", err)
}

func (q *queries) UpdateFishingPermit(ctx context.Context, program domain.FishingPermit, expectedVersion int64) error {
	if err := program.Validate(); err != nil {
		return err
	}
	result, err := q.q.ExecContext(ctx, `UPDATE fishing_permits SET status = ?, version = version + 1, updated_at = ? WHERE id = ? AND version = ?`, program.Status, formatTime(program.UpdatedAt), program.ID, expectedVersion)
	if err != nil {
		return translateError("update program", err)
	}
	return expectVersion(result, "update program")
}

func (q *queries) InsertPortFacility(ctx context.Context, port_facility domain.PortFacility) error {
	if err := port_facility.Validate(); err != nil {
		return err
	}
	_, err := q.q.ExecContext(ctx, `INSERT INTO port_facilities(id, code, name, timezone, status, daily_limit, cutoff_hour, version, created_at, updated_at)
        VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, port_facility.ID, port_facility.Code, port_facility.Name, port_facility.Timezone, port_facility.Status,
		port_facility.DailyLimit, port_facility.CutoffHour, port_facility.Version, formatTime(port_facility.CreatedAt), formatTime(port_facility.UpdatedAt))
	return translateError("insert port_facility", err)
}

func (q *queries) InsertFishingVessel(ctx context.Context, batch domain.FishingVessel) error {
	if err := batch.Validate(); err != nil {
		return err
	}
	requestID := nullableString(batch.FishingVoyageID)
	_, err := q.q.ExecContext(ctx, `INSERT INTO fishing_vessels(id, fishing_permit_id, departure_port_id, registry_number, vessel_class,
        voyage_count, hold_capacity_kg, state, expires_at, request_id, quarantine_note, version, created_at, updated_at)
        VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, batch.ID, batch.FishingPermitID, batch.DeparturePortID,
		batch.RegistryNumber, batch.VesselClass, batch.VoyageCount, batch.HoldCapacityKg, batch.State,
		formatTime(batch.ExpiresAt), requestID, batch.QuarantineNote, batch.Version,
		formatTime(batch.CreatedAt), formatTime(batch.UpdatedAt))
	return translateError("insert fishing vessel", err)
}

func (q *queries) UpdateFishingVessel(ctx context.Context, batch domain.FishingVessel, expectedVersion int64) error {
	if err := batch.Validate(); err != nil {
		return err
	}
	result, err := q.q.ExecContext(ctx, `UPDATE fishing_vessels SET state = ?, request_id = ?, quarantine_note = ?,
        expires_at = ?, version = version + 1, updated_at = ? WHERE id = ? AND version = ?`, batch.State,
		nullableString(batch.FishingVoyageID), batch.QuarantineNote, formatTime(batch.ExpiresAt), formatTime(batch.UpdatedAt),
		batch.ID, expectedVersion)
	if err != nil {
		return translateError("update fishing vessel", err)
	}
	return expectVersion(result, "update fishing vessel")
}

func (q *queries) InsertSupportFleet(ctx context.Context, support_fleet domain.SupportFleet) error {
	if err := support_fleet.Validate(); err != nil {
		return err
	}
	_, err := q.q.ExecContext(ctx, `INSERT INTO support_fleets(id, fleet_code, state, cargo_capacity_kg, certification_due_at,
        last_inspected_at, assigned_voyage_id, version, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		support_fleet.ID, support_fleet.FleetCode, support_fleet.State, support_fleet.CargoCapacityKg,
		formatTime(support_fleet.CertificationDueAt), formatTime(support_fleet.LastInspectionAt), nullableString(support_fleet.AssignedVoyageID),
		support_fleet.Version, formatTime(support_fleet.CreatedAt), formatTime(support_fleet.UpdatedAt))
	return translateError("insert support_fleet", err)
}

func (q *queries) UpdateSupportFleet(ctx context.Context, support_fleet domain.SupportFleet, expectedVersion int64) error {
	if err := support_fleet.Validate(); err != nil {
		return err
	}
	result, err := q.q.ExecContext(ctx, `UPDATE support_fleets SET state = ?, cargo_capacity_kg = ?, certification_due_at = ?,
        last_inspected_at = ?, assigned_voyage_id = ?, version = version + 1, updated_at = ? WHERE id = ? AND version = ?`,
		support_fleet.State, support_fleet.CargoCapacityKg, formatTime(support_fleet.CertificationDueAt), formatTime(support_fleet.LastInspectionAt),
		nullableString(support_fleet.AssignedVoyageID), formatTime(support_fleet.UpdatedAt), support_fleet.ID, expectedVersion)
	if err != nil {
		return translateError("update support_fleet", err)
	}
	return expectVersion(result, "update support_fleet")
}

func (q *queries) InsertFishingVoyage(ctx context.Context, run domain.FishingVoyage) error {
	if err := run.Validate(); err != nil {
		return err
	}
	_, err := q.q.ExecContext(ctx, `INSERT INTO fishing_voyages(id, fishing_permit_id, departure_port_id, landing_port_id, support_fleet_id,
        voyage_code, state, departure_window_opens_at, landing_deadline_at, departed_at, landed_at, closed_at,
        total_hold_capacity_kg, version, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.FishingPermitID, run.DeparturePortID, run.LandingPortID, run.SupportFleetID,
		run.VoyageCode, run.State, formatTime(run.DepartureWindowOpensAt), formatTime(run.LandingDeadlineAt),
		nullableTime(run.DepartedAt), nullableTime(run.LandedAt), nullableTime(run.ClosedAt),
		run.TotalHoldCapacityKg, run.Version, formatTime(run.CreatedAt), formatTime(run.UpdatedAt))
	return translateError("insert fishing voyage", err)
}

func (q *queries) UpdateFishingVoyage(ctx context.Context, run domain.FishingVoyage, expectedVersion int64) error {
	if err := run.Validate(); err != nil {
		return err
	}
	result, err := q.q.ExecContext(ctx, `UPDATE fishing_voyages SET state = ?, departed_at = ?, landed_at = ?, closed_at = ?,
        version = version + 1, updated_at = ? WHERE id = ? AND version = ?`, run.State,
		nullableTime(run.DepartedAt), nullableTime(run.LandedAt), nullableTime(run.ClosedAt),
		formatTime(run.UpdatedAt), run.ID, expectedVersion)
	if err != nil {
		return translateError("update fishing voyage", err)
	}
	return expectVersion(result, "update fishing voyage")
}

func (q *queries) InsertFishingVoyageVesselLink(ctx context.Context, item domain.FishingVoyageVesselLink) error {
	_, err := q.q.ExecContext(ctx, `INSERT INTO fishing_voyage_vessels(request_id, fishing_vessel_id, added_at) VALUES(?, ?, ?)`,
		item.FishingVoyageID, item.FishingVesselID, formatTime(item.AddedAt))
	return translateError("insert fishing voyage vessel", err)
}

func (q *queries) InsertCatchLandingTask(ctx context.Context, catch_landing_task domain.CatchLandingTask) error {
	if err := catch_landing_task.Validate(); err != nil {
		return err
	}
	_, err := q.q.ExecContext(ctx, `INSERT INTO catch_landing_tasks(id, request_id, coordinator_id, fisheries_officer_id,
        landing_station, status, expires_at, resolved_at, resolution_note, version, created_at, updated_at)
        VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, catch_landing_task.ID, catch_landing_task.FishingVoyageID, catch_landing_task.CoordinatorID,
		catch_landing_task.FisheriesOfficerID, catch_landing_task.LandingStation, catch_landing_task.Status, formatTime(catch_landing_task.ExpiresAt), nullableTime(catch_landing_task.ResolvedAt),
		catch_landing_task.ResolutionNote, catch_landing_task.Version, formatTime(catch_landing_task.CreatedAt), formatTime(catch_landing_task.UpdatedAt))
	return translateError("insert catch_landing_task", err)
}

func (q *queries) UpdateCatchLandingTask(ctx context.Context, catch_landing_task domain.CatchLandingTask, expectedVersion int64) error {
	if err := catch_landing_task.Validate(); err != nil {
		return err
	}
	result, err := q.q.ExecContext(ctx, `UPDATE catch_landing_tasks SET status = ?, resolved_at = ?, resolution_note = ?,
        version = version + 1, updated_at = ? WHERE id = ? AND version = ?`, catch_landing_task.Status,
		nullableTime(catch_landing_task.ResolvedAt), catch_landing_task.ResolutionNote, formatTime(catch_landing_task.UpdatedAt), catch_landing_task.ID, expectedVersion)
	if err != nil {
		return translateError("update catch_landing_task", err)
	}
	return expectVersion(result, "update catch_landing_task")
}

func (q *queries) InsertCatchDeclaration(ctx context.Context, declaration domain.CatchDeclaration) error {
	if err := declaration.Validate(); err != nil {
		return err
	}
	_, err := q.q.ExecContext(ctx, `INSERT INTO catch_declarations(id, request_id, species_code, sequence,
        catch_variance_grams, recorded_at, received_at) VALUES(?, ?, ?, ?, ?, ?, ?)`, declaration.ID,
		declaration.FishingVoyageID, declaration.SpeciesCode, declaration.Sequence, declaration.CatchVariance,
		formatTime(declaration.RecordedAt), formatTime(declaration.ReceivedAt))
	return translateError("insert catch_variance declaration", err)
}

func (q *queries) InsertCatchAnomaly(ctx context.Context, catch_anomaly domain.CatchAnomaly) error {
	_, err := q.q.ExecContext(ctx, `INSERT INTO catch_anomalies(id, request_id, status, first_declaration_at, last_declaration_at,
        minimum_catch_variance_grams, maximum_catch_variance_grams, declaration_count, review_due_at, version, created_at, updated_at)
        VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, catch_anomaly.ID, catch_anomaly.FishingVoyageID, catch_anomaly.Status,
		formatTime(catch_anomaly.FirstDeclarationAt), formatTime(catch_anomaly.LastDeclarationAt), catch_anomaly.Minimum, catch_anomaly.Maximum,
		catch_anomaly.DeclarationCount, formatTime(catch_anomaly.ReviewDueAt), catch_anomaly.Version,
		formatTime(catch_anomaly.CreatedAt), formatTime(catch_anomaly.UpdatedAt))
	return translateError("insert catch_anomaly", err)
}

func (q *queries) UpdateCatchAnomaly(ctx context.Context, catch_anomaly domain.CatchAnomaly, expectedVersion int64) error {
	result, err := q.q.ExecContext(ctx, `UPDATE catch_anomalies SET status = ?, first_declaration_at = ?, last_declaration_at = ?,
        minimum_catch_variance_grams = ?, maximum_catch_variance_grams = ?, declaration_count = ?, review_due_at = ?,
        version = version + 1, updated_at = ? WHERE id = ? AND version = ?`, catch_anomaly.Status,
		formatTime(catch_anomaly.FirstDeclarationAt), formatTime(catch_anomaly.LastDeclarationAt), catch_anomaly.Minimum, catch_anomaly.Maximum,
		catch_anomaly.DeclarationCount, formatTime(catch_anomaly.ReviewDueAt), formatTime(catch_anomaly.UpdatedAt), catch_anomaly.ID, expectedVersion)
	if err != nil {
		return translateError("update catch_anomaly", err)
	}
	return expectVersion(result, "update catch_anomaly")
}

func (q *queries) InsertAnomalyDisposition(ctx context.Context, decision domain.AnomalyDisposition) error {
	_, err := q.q.ExecContext(ctx, `INSERT INTO anomaly_dispositions(id, catch_anomaly_id, reviewer_id, decision, rationale, created_at)
        VALUES(?, ?, ?, ?, ?, ?)`, decision.ID, decision.CatchAnomalyID, decision.Reviewer, decision.Decision,
		decision.Rationale, formatTime(decision.CreatedAt))
	return translateError("insert review decision", err)
}

func (q *queries) InsertAuditEvent(ctx context.Context, event domain.AuditEvent) error {
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("encode audit metadata: %w", err)
	}
	_, err = q.q.ExecContext(ctx, `INSERT INTO audit_events(id, request_id, actor, action, entity_type, entity_id,
        outcome, metadata_json, created_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`, event.ID, event.RequestID,
		event.Actor, event.Action, event.EntityType, event.EntityID, event.Outcome, string(metadata), formatTime(event.CreatedAt))
	return translateError("insert audit event", err)
}

func (q *queries) PutIdempotency(ctx context.Context, record repository.IdempotencyRecord) error {
	_, err := q.q.ExecContext(ctx, `INSERT INTO idempotency_records(scope, idempotency_key, request_hash,
        response_code, response_body, expires_at, created_at) VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(scope, idempotency_key) DO UPDATE SET request_hash = excluded.request_hash,
		response_code = excluded.response_code, response_body = excluded.response_body,
		expires_at = excluded.expires_at, created_at = excluded.created_at
		WHERE idempotency_records.expires_at <= excluded.created_at`, record.Scope,
		record.Key, record.RequestHash, record.ResponseCode, append([]byte(nil), record.ResponseBody...),
		formatTime(record.ExpiresAt), formatTime(record.CreatedAt))
	return translateError("put idempotency record", err)
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return formatTime(*value)
}
