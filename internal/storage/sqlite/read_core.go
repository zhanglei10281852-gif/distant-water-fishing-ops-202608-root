package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
)

type scanner interface {
	Scan(dest ...any) error
}

func (q *queries) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	row := q.q.QueryRowContext(ctx, userSelect+` WHERE email = ? COLLATE NOCASE`, strings.TrimSpace(email))
	user, err := scanUser(row)
	return user, translateError("get user by email", err)
}

func (q *queries) GetUser(ctx context.Context, id string) (domain.User, error) {
	user, err := scanUser(q.q.QueryRowContext(ctx, userSelect+` WHERE id = ?`, id))
	return user, translateError("get user", err)
}

const userSelect = `SELECT id, email, display_name, password_hash, role, status, version, created_at, updated_at FROM users`

func scanUser(row scanner) (domain.User, error) {
	var user domain.User
	var role, status, createdAt, updatedAt string
	if err := row.Scan(&user.ID, &user.Email, &user.DisplayName, &user.PasswordHash, &role, &status, &user.Version, &createdAt, &updatedAt); err != nil {
		return domain.User{}, err
	}
	var err error
	if user.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.User{}, err
	}
	if user.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.User{}, err
	}
	user.Role = domain.Role(role)
	user.Status = domain.UserStatus(status)
	return user, nil
}

func (q *queries) GetSessionByTokenHash(ctx context.Context, tokenHash string) (domain.Session, error) {
	row := q.q.QueryRowContext(ctx, `SELECT id, user_id, token_hash, expires_at, created_at, revoked_at FROM sessions WHERE token_hash = ?`, tokenHash)
	var session domain.Session
	var expiresAt, createdAt string
	var revokedAt sql.NullString
	if err := row.Scan(&session.ID, &session.UserID, &session.TokenHash, &expiresAt, &createdAt, &revokedAt); err != nil {
		return domain.Session{}, translateError("get session", err)
	}
	var err error
	if session.ExpiresAt, err = parseTime(expiresAt); err != nil {
		return domain.Session{}, err
	}
	if session.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.Session{}, err
	}
	if session.RevokedAt, err = parseNullableTime(revokedAt); err != nil {
		return domain.Session{}, err
	}
	return session, nil
}

func (q *queries) GetFishingPermit(ctx context.Context, id string) (domain.FishingPermit, error) {
	row := q.q.QueryRowContext(ctx, `SELECT id, code, name, status, minimum_catch_variance_grams, maximum_catch_variance_grams,
        max_voyage_duration_seconds, compliance_review_deadline_seconds, business_timezone, version, created_at, updated_at
        FROM fishing_permits WHERE id = ?`, id)
	var program domain.FishingPermit
	var status, createdAt, updatedAt string
	var maxVoyageDurationSeconds, complianceReviewDeadlineSeconds int64
	if err := row.Scan(&program.ID, &program.Code, &program.Name, &status, &program.CatchVariance.Minimum,
		&program.CatchVariance.Maximum, &maxVoyageDurationSeconds, &complianceReviewDeadlineSeconds, &program.BusinessTimezone,
		&program.Version, &createdAt, &updatedAt); err != nil {
		return domain.FishingPermit{}, translateError("get program", err)
	}
	program.Status = domain.FishingPermitStatus(status)
	program.MaxVoyageDuration = durationSeconds(maxVoyageDurationSeconds)
	program.ComplianceReviewDeadline = durationSeconds(complianceReviewDeadlineSeconds)
	var err error
	if program.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.FishingPermit{}, err
	}
	if program.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.FishingPermit{}, err
	}
	return program, nil
}

func (q *queries) GetPortFacility(ctx context.Context, id string) (domain.PortFacility, error) {
	row := q.q.QueryRowContext(ctx, `SELECT id, code, name, timezone, status, daily_limit, cutoff_hour, version, created_at, updated_at FROM port_facilities WHERE id = ?`, id)
	var port_facility domain.PortFacility
	var status, createdAt, updatedAt string
	if err := row.Scan(&port_facility.ID, &port_facility.Code, &port_facility.Name, &port_facility.Timezone, &status, &port_facility.DailyLimit, &port_facility.CutoffHour, &port_facility.Version, &createdAt, &updatedAt); err != nil {
		return domain.PortFacility{}, translateError("get port_facility", err)
	}
	port_facility.Status = domain.PortFacilityStatus(status)
	var err error
	if port_facility.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.PortFacility{}, err
	}
	if port_facility.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.PortFacility{}, err
	}
	return port_facility, nil
}

func (q *queries) GetFishingVessel(ctx context.Context, id string) (domain.FishingVessel, error) {
	batch, err := scanFishingVessel(q.q.QueryRowContext(ctx, fishingVesselSelect+` WHERE id = ?`, id))
	return batch, translateError("get fishing vessel", err)
}

const fishingVesselSelect = `SELECT id, fishing_permit_id, departure_port_id, registry_number, vessel_class, voyage_count, hold_capacity_kg,
    state, expires_at, COALESCE(request_id, ''), quarantine_note, version, created_at, updated_at FROM fishing_vessels`

func scanFishingVessel(row scanner) (domain.FishingVessel, error) {
	var batch domain.FishingVessel
	var state, expiresAt, createdAt, updatedAt string
	if err := row.Scan(&batch.ID, &batch.FishingPermitID, &batch.DeparturePortID, &batch.RegistryNumber, &batch.VesselClass,
		&batch.VoyageCount, &batch.HoldCapacityKg, &state, &expiresAt, &batch.FishingVoyageID, &batch.QuarantineNote,
		&batch.Version, &createdAt, &updatedAt); err != nil {
		return domain.FishingVessel{}, err
	}
	batch.State = domain.FishingVesselState(state)
	var err error
	if batch.ExpiresAt, err = parseTime(expiresAt); err != nil {
		return domain.FishingVessel{}, err
	}
	if batch.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.FishingVessel{}, err
	}
	if batch.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.FishingVessel{}, err
	}
	return batch, nil
}

func (q *queries) GetSupportFleet(ctx context.Context, id string) (domain.SupportFleet, error) {
	row := q.q.QueryRowContext(ctx, `SELECT id, fleet_code, state, cargo_capacity_kg, certification_due_at, last_inspected_at,
        COALESCE(assigned_voyage_id, ''), version, created_at, updated_at FROM support_fleets WHERE id = ?`, id)
	support_fleet, err := scanSupportFleet(row)
	return support_fleet, translateError("get support_fleet", err)
}

func scanSupportFleet(row scanner) (domain.SupportFleet, error) {
	var support_fleet domain.SupportFleet
	var state, certificationDueAt, lastInspectionAt, createdAt, updatedAt string
	if err := row.Scan(&support_fleet.ID, &support_fleet.FleetCode, &state, &support_fleet.CargoCapacityKg,
		&certificationDueAt, &lastInspectionAt, &support_fleet.AssignedVoyageID, &support_fleet.Version, &createdAt, &updatedAt); err != nil {
		return domain.SupportFleet{}, err
	}
	support_fleet.State = domain.SupportFleetState(state)
	var err error
	if support_fleet.CertificationDueAt, err = parseTime(certificationDueAt); err != nil {
		return domain.SupportFleet{}, err
	}
	if support_fleet.LastInspectionAt, err = parseTime(lastInspectionAt); err != nil {
		return domain.SupportFleet{}, err
	}
	if support_fleet.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.SupportFleet{}, err
	}
	if support_fleet.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.SupportFleet{}, err
	}
	return support_fleet, nil
}

func (q *queries) GetFishingVoyage(ctx context.Context, id string) (domain.FishingVoyage, error) {
	run, err := scanFishingVoyage(q.q.QueryRowContext(ctx, runSelect+` WHERE id = ?`, id))
	return run, translateError("get fishing voyage", err)
}

const runSelect = `SELECT id, fishing_permit_id, departure_port_id, landing_port_id, support_fleet_id, voyage_code, state,
    departure_window_opens_at, landing_deadline_at, departed_at, landed_at, closed_at, total_hold_capacity_kg, version, created_at, updated_at FROM fishing_voyages`

func scanFishingVoyage(row scanner) (domain.FishingVoyage, error) {
	var run domain.FishingVoyage
	var state, departureWindowOpensAt, landingDeadlineAt, createdAt, updatedAt string
	var departedAt, landedAt, closedAt sql.NullString
	if err := row.Scan(&run.ID, &run.FishingPermitID, &run.DeparturePortID, &run.LandingPortID,
		&run.SupportFleetID, &run.VoyageCode, &state, &departureWindowOpensAt, &landingDeadlineAt,
		&departedAt, &landedAt, &closedAt, &run.TotalHoldCapacityKg, &run.Version,
		&createdAt, &updatedAt); err != nil {
		return domain.FishingVoyage{}, err
	}
	run.State = domain.FishingVoyageState(state)
	var err error
	if run.DepartureWindowOpensAt, err = parseTime(departureWindowOpensAt); err != nil {
		return domain.FishingVoyage{}, err
	}
	if run.LandingDeadlineAt, err = parseTime(landingDeadlineAt); err != nil {
		return domain.FishingVoyage{}, err
	}
	if run.DepartedAt, err = parseNullableTime(departedAt); err != nil {
		return domain.FishingVoyage{}, err
	}
	if run.LandedAt, err = parseNullableTime(landedAt); err != nil {
		return domain.FishingVoyage{}, err
	}
	if run.ClosedAt, err = parseNullableTime(closedAt); err != nil {
		return domain.FishingVoyage{}, err
	}
	if run.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.FishingVoyage{}, err
	}
	if run.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.FishingVoyage{}, err
	}
	return run, nil
}

func (q *queries) ListFishingVoyageVessels(ctx context.Context, requestID string) ([]domain.FishingVessel, error) {
	rows, err := q.q.QueryContext(ctx, `SELECT fishing_vessels.id, fishing_vessels.fishing_permit_id, fishing_vessels.departure_port_id,
        fishing_vessels.registry_number, fishing_vessels.vessel_class, fishing_vessels.voyage_count, fishing_vessels.hold_capacity_kg,
        fishing_vessels.state, fishing_vessels.expires_at, COALESCE(fishing_vessels.request_id, ''), fishing_vessels.quarantine_note,
        fishing_vessels.version, fishing_vessels.created_at, fishing_vessels.updated_at
		FROM fishing_vessels JOIN fishing_voyage_vessels ri ON ri.fishing_vessel_id = fishing_vessels.id
		WHERE ri.request_id = ? ORDER BY ri.added_at, fishing_vessels.id`, requestID)
	if err != nil {
		return nil, translateError("list fishing voyage vessels", err)
	}
	defer rows.Close()
	items := make([]domain.FishingVessel, 0)
	for rows.Next() {
		batch, err := scanFishingVessel(rows)
		if err != nil {
			return nil, fmt.Errorf("scan fishing voyage vessel: %w", err)
		}
		items = append(items, batch.Clone())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate fishing voyage vessels: %w", err)
	}
	return items, nil
}

func decodeMetadata(raw string) (map[string]string, error) {
	metadata := make(map[string]string)
	if raw == "" || raw == "{}" {
		return metadata, nil
	}
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		return nil, fmt.Errorf("decode audit metadata: %w", err)
	}
	return metadata, nil
}
