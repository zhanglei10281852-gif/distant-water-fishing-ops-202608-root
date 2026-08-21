package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/repository"
)

func (q *queries) GetPendingCatchLanding(ctx context.Context, requestID string) (domain.CatchLandingTask, error) {
	catch_landing_task, err := scanCatchLandingTask(q.q.QueryRowContext(ctx, catch_landing_taskSelect+` WHERE request_id = ? AND status = 'pending'`, requestID))
	return catch_landing_task, translateError("get pending catch_landing_task", err)
}

func (q *queries) GetCatchLandingTask(ctx context.Context, id string) (domain.CatchLandingTask, error) {
	catch_landing_task, err := scanCatchLandingTask(q.q.QueryRowContext(ctx, catch_landing_taskSelect+` WHERE id = ?`, id))
	return catch_landing_task, translateError("get catch_landing_task", err)
}

const catch_landing_taskSelect = `SELECT id, request_id, coordinator_id, fisheries_officer_id, landing_station, status, expires_at,
    resolved_at, resolution_note, version, created_at, updated_at FROM catch_landing_tasks`

func scanCatchLandingTask(row scanner) (domain.CatchLandingTask, error) {
	var catch_landing_task domain.CatchLandingTask
	var status, expiresAt, createdAt, updatedAt string
	var resolvedAt sql.NullString
	if err := row.Scan(&catch_landing_task.ID, &catch_landing_task.FishingVoyageID, &catch_landing_task.CoordinatorID, &catch_landing_task.FisheriesOfficerID,
		&catch_landing_task.LandingStation, &status, &expiresAt, &resolvedAt, &catch_landing_task.ResolutionNote,
		&catch_landing_task.Version, &createdAt, &updatedAt); err != nil {
		return domain.CatchLandingTask{}, err
	}
	catch_landing_task.Status = domain.CatchLandingTaskStatus(status)
	var err error
	if catch_landing_task.ExpiresAt, err = parseTime(expiresAt); err != nil {
		return domain.CatchLandingTask{}, err
	}
	if catch_landing_task.ResolvedAt, err = parseNullableTime(resolvedAt); err != nil {
		return domain.CatchLandingTask{}, err
	}
	if catch_landing_task.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.CatchLandingTask{}, err
	}
	if catch_landing_task.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.CatchLandingTask{}, err
	}
	return catch_landing_task, nil
}

func (q *queries) GetActiveCatchAnomaly(ctx context.Context, requestID string) (domain.CatchAnomaly, error) {
	catch_anomaly, err := scanCatchAnomaly(q.q.QueryRowContext(ctx, catch_anomalySelect+` WHERE request_id = ? AND status IN ('open', 'reviewing')`, requestID))
	return catch_anomaly, translateError("get active catch_anomaly", err)
}

func (q *queries) GetCatchAnomaly(ctx context.Context, id string) (domain.CatchAnomaly, error) {
	catch_anomaly, err := scanCatchAnomaly(q.q.QueryRowContext(ctx, catch_anomalySelect+` WHERE id = ?`, id))
	return catch_anomaly, translateError("get catch_anomaly", err)
}

const catch_anomalySelect = `SELECT id, request_id, status, first_declaration_at, last_declaration_at,
    minimum_catch_variance_grams, maximum_catch_variance_grams, declaration_count, review_due_at, version, created_at, updated_at FROM catch_anomalies`

func scanCatchAnomaly(row scanner) (domain.CatchAnomaly, error) {
	var catch_anomaly domain.CatchAnomaly
	var status, firstSampleAt, lastSampleAt, reviewDueAt, createdAt, updatedAt string
	if err := row.Scan(&catch_anomaly.ID, &catch_anomaly.FishingVoyageID, &status, &firstSampleAt, &lastSampleAt,
		&catch_anomaly.Minimum, &catch_anomaly.Maximum, &catch_anomaly.DeclarationCount, &reviewDueAt,
		&catch_anomaly.Version, &createdAt, &updatedAt); err != nil {
		return domain.CatchAnomaly{}, err
	}
	catch_anomaly.Status = domain.CatchAnomalyStatus(status)
	var err error
	if catch_anomaly.FirstDeclarationAt, err = parseTime(firstSampleAt); err != nil {
		return domain.CatchAnomaly{}, err
	}
	if catch_anomaly.LastDeclarationAt, err = parseTime(lastSampleAt); err != nil {
		return domain.CatchAnomaly{}, err
	}
	if catch_anomaly.ReviewDueAt, err = parseTime(reviewDueAt); err != nil {
		return domain.CatchAnomaly{}, err
	}
	if catch_anomaly.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.CatchAnomaly{}, err
	}
	if catch_anomaly.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.CatchAnomaly{}, err
	}
	return catch_anomaly, nil
}

func (q *queries) GetIdempotency(ctx context.Context, scope, key string) (repository.IdempotencyRecord, error) {
	row := q.q.QueryRowContext(ctx, `SELECT scope, idempotency_key, request_hash, response_code, response_body, expires_at, created_at
        FROM idempotency_records WHERE scope = ? AND idempotency_key = ?`, scope, key)
	var record repository.IdempotencyRecord
	var expiresAt, createdAt string
	if err := row.Scan(&record.Scope, &record.Key, &record.RequestHash, &record.ResponseCode, &record.ResponseBody, &expiresAt, &createdAt); err != nil {
		return repository.IdempotencyRecord{}, translateError("get idempotency record", err)
	}
	var err error
	if record.ExpiresAt, err = parseTime(expiresAt); err != nil {
		return repository.IdempotencyRecord{}, err
	}
	if record.CreatedAt, err = parseTime(createdAt); err != nil {
		return repository.IdempotencyRecord{}, err
	}
	record.ResponseBody = append([]byte(nil), record.ResponseBody...)
	return record, nil
}

func (q *queries) CountPortFacilityFishingVoyagesForWindow(ctx context.Context, portFacilityID string, startsAt, endsAt time.Time) (int, error) {
	var count int
	err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM fishing_voyages
		WHERE departure_port_id = ? AND departure_window_opens_at >= ? AND departure_window_opens_at < ?
		AND state != 'cancelled'`, portFacilityID, formatTime(startsAt), formatTime(endsAt)).Scan(&count)
	if err != nil {
		return 0, translateError("count port_facility fishing_voyages", err)
	}
	return count, nil
}

func (q *queries) ListFishingVoyages(ctx context.Context, filter repository.FishingVoyageFilter) (repository.FishingVoyagePage, error) {
	page := filter.Page.Normalize(200)
	where, args := buildFishingVoyageWhere(filter)
	var total int
	if err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM fishing_voyages`+where, args...).Scan(&total); err != nil {
		return repository.FishingVoyagePage{}, translateError("count fishing_voyages", err)
	}
	sortColumn := runSortColumn(page.Sort)
	direction := " ASC"
	if page.Desc {
		direction = " DESC"
	}
	query := runSelect + where + ` ORDER BY ` + sortColumn + direction + `, id ASC LIMIT ? OFFSET ?`
	rows, err := q.q.QueryContext(ctx, query, append(args, page.Limit, page.Offset)...)
	if err != nil {
		return repository.FishingVoyagePage{}, translateError("list fishing_voyages", err)
	}
	defer rows.Close()
	items := make([]domain.FishingVoyage, 0, page.Limit)
	for rows.Next() {
		run, err := scanFishingVoyage(rows)
		if err != nil {
			return repository.FishingVoyagePage{}, fmt.Errorf("scan fishing voyage: %w", err)
		}
		items = append(items, run)
	}
	if err := rows.Err(); err != nil {
		return repository.FishingVoyagePage{}, fmt.Errorf("iterate fishing_voyages: %w", err)
	}
	return repository.FishingVoyagePage{Items: items, Total: total}, nil
}

func buildFishingVoyageWhere(filter repository.FishingVoyageFilter) (string, []any) {
	clauses := make([]string, 0, 6)
	args := make([]any, 0, 6)
	appendStringFilter := func(column, value string) {
		if value != "" {
			clauses = append(clauses, column+" = ?")
			args = append(args, value)
		}
	}
	appendStringFilter("fishing_permit_id", filter.FishingPermitID)
	appendStringFilter("departure_port_id", filter.DeparturePortID)
	appendStringFilter("landing_port_id", filter.LandingPortID)
	appendStringFilter("state", string(filter.State))
	if filter.From != nil {
		clauses = append(clauses, "departure_window_opens_at >= ?")
		args = append(args, formatTime(*filter.From))
	}
	if filter.To != nil {
		clauses = append(clauses, "departure_window_opens_at < ?")
		args = append(args, formatTime(*filter.To))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func runSortColumn(value string) string {
	switch value {
	case "landing_deadline_at":
		return "landing_deadline_at"
	case "updated_at":
		return "updated_at"
	case "voyage_code":
		return "voyage_code"
	default:
		return "departure_window_opens_at"
	}
}

func (q *queries) ListFishingVessels(ctx context.Context, filter repository.FishingVesselFilter) (repository.FishingVesselPage, error) {
	page := filter.Page.Normalize(200)
	where, args := buildFishingVesselWhere(filter)
	var total int
	if err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM fishing_vessels`+where, args...).Scan(&total); err != nil {
		return repository.FishingVesselPage{}, translateError("count fishing_vessels", err)
	}
	query := fishingVesselSelect + where + ` ORDER BY expires_at ASC, id ASC LIMIT ? OFFSET ?`
	rows, err := q.q.QueryContext(ctx, query, append(args, page.Limit, page.Offset)...)
	if err != nil {
		return repository.FishingVesselPage{}, translateError("list fishing_vessels", err)
	}
	defer rows.Close()
	items := make([]domain.FishingVessel, 0, page.Limit)
	for rows.Next() {
		batch, err := scanFishingVessel(rows)
		if err != nil {
			return repository.FishingVesselPage{}, fmt.Errorf("scan fishing vessel: %w", err)
		}
		items = append(items, batch.Clone())
	}
	if err := rows.Err(); err != nil {
		return repository.FishingVesselPage{}, fmt.Errorf("iterate fishing_vessels: %w", err)
	}
	return repository.FishingVesselPage{Items: items, Total: total}, nil
}

func buildFishingVesselWhere(filter repository.FishingVesselFilter) (string, []any) {
	clauses := make([]string, 0, 5)
	args := make([]any, 0, 5)
	values := []struct{ column, value string }{
		{"fishing_permit_id", filter.FishingPermitID}, {"departure_port_id", filter.PortFacilityID}, {"request_id", filter.FishingVoyageID}, {"state", string(filter.State)},
	}
	for _, item := range values {
		if item.value != "" {
			clauses = append(clauses, item.column+" = ?")
			args = append(args, item.value)
		}
	}
	if filter.ExpiresBy != nil {
		clauses = append(clauses, "expires_at <= ?")
		args = append(args, formatTime(*filter.ExpiresBy))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func (q *queries) ListLandingAnomalies(ctx context.Context, filter repository.CatchAnomalyFilter) (repository.CatchAnomalyPage, error) {
	page := filter.Page.Normalize(200)
	clauses := make([]string, 0, 3)
	args := make([]any, 0, 3)
	if filter.FishingVoyageID != "" {
		clauses = append(clauses, "request_id = ?")
		args = append(args, filter.FishingVoyageID)
	}
	if filter.Status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.DueBefore != nil {
		clauses = append(clauses, "review_due_at <= ?")
		args = append(args, formatTime(*filter.DueBefore))
	}
	where := ""
	if len(clauses) > 0 {
		where = " WHERE " + strings.Join(clauses, " AND ")
	}
	var total int
	if err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM catch_anomalies`+where, args...).Scan(&total); err != nil {
		return repository.CatchAnomalyPage{}, translateError("count catch_anomalies", err)
	}
	rows, err := q.q.QueryContext(ctx, catch_anomalySelect+where+` ORDER BY review_due_at ASC, id ASC LIMIT ? OFFSET ?`, append(args, page.Limit, page.Offset)...)
	if err != nil {
		return repository.CatchAnomalyPage{}, translateError("list catch_anomalies", err)
	}
	defer rows.Close()
	items := make([]domain.CatchAnomaly, 0, page.Limit)
	for rows.Next() {
		catch_anomaly, err := scanCatchAnomaly(rows)
		if err != nil {
			return repository.CatchAnomalyPage{}, fmt.Errorf("scan catch_anomaly: %w", err)
		}
		items = append(items, catch_anomaly)
	}
	if err := rows.Err(); err != nil {
		return repository.CatchAnomalyPage{}, fmt.Errorf("iterate catch_anomalies: %w", err)
	}
	return repository.CatchAnomalyPage{Items: items, Total: total}, nil
}

func (q *queries) ListAuditEvents(ctx context.Context, filter repository.AuditFilter) (repository.AuditPage, error) {
	page := filter.Page.Normalize(500)
	clauses := make([]string, 0, 4)
	args := make([]any, 0, 4)
	values := []struct{ column, value string }{
		{"entity_type", filter.EntityType}, {"entity_id", filter.EntityID}, {"actor", filter.Actor}, {"request_id", filter.RequestID},
	}
	for _, item := range values {
		if item.value != "" {
			clauses = append(clauses, item.column+" = ?")
			args = append(args, item.value)
		}
	}
	where := ""
	if len(clauses) > 0 {
		where = " WHERE " + strings.Join(clauses, " AND ")
	}
	var total int
	if err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events`+where, args...).Scan(&total); err != nil {
		return repository.AuditPage{}, translateError("count audit events", err)
	}
	rows, err := q.q.QueryContext(ctx, `SELECT id, request_id, actor, action, entity_type, entity_id, outcome, metadata_json, created_at
        FROM audit_events`+where+` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`, append(args, page.Limit, page.Offset)...)
	if err != nil {
		return repository.AuditPage{}, translateError("list audit events", err)
	}
	defer rows.Close()
	items := make([]domain.AuditEvent, 0, page.Limit)
	for rows.Next() {
		var event domain.AuditEvent
		var metadataJSON, createdAt string
		if err := rows.Scan(&event.ID, &event.RequestID, &event.Actor, &event.Action, &event.EntityType,
			&event.EntityID, &event.Outcome, &metadataJSON, &createdAt); err != nil {
			return repository.AuditPage{}, fmt.Errorf("scan audit event: %w", err)
		}
		metadata, err := decodeMetadata(metadataJSON)
		if err != nil {
			return repository.AuditPage{}, err
		}
		event.Metadata = metadata
		if event.CreatedAt, err = parseTime(createdAt); err != nil {
			return repository.AuditPage{}, err
		}
		items = append(items, event.Clone())
	}
	if err := rows.Err(); err != nil {
		return repository.AuditPage{}, fmt.Errorf("iterate audit events: %w", err)
	}
	return repository.AuditPage{Items: items, Total: total}, nil
}

func beginningOfUTCDate(day string) (time.Time, error) {
	return time.Parse("2006-01-02", day)
}
