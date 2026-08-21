package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
)

func (q *queries) GetVoyageReadiness(ctx context.Context, requestID string) (domain.VoyageReadiness, error) {
	run, err := q.GetFishingVoyage(ctx, requestID)
	if err != nil {
		return domain.VoyageReadiness{}, err
	}
	var report domain.VoyageReadiness
	report.FishingVoyageID = run.ID
	report.FishingVoyageState = run.State
	if err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM fishing_voyage_vessels WHERE request_id = ?`, requestID).Scan(&report.ExpectedFishingVesselCount); err != nil {
		return domain.VoyageReadiness{}, translateError("count mission fishing vessels", err)
	}
	if err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM fishing_voyage_vessels ri JOIN fishing_vessels s ON s.id = ri.fishing_vessel_id
        WHERE ri.request_id = ? AND s.state IN ('landed', 'reinspected', 'retired', 'quarantined')`, requestID).Scan(&report.LandedFishingVesselCount); err != nil {
		return domain.VoyageReadiness{}, translateError("count recovered fishing vessels", err)
	}
	if err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM fishing_voyage_vessels ri JOIN fishing_vessels s ON s.id = ri.fishing_vessel_id
		WHERE ri.request_id = ? AND s.state = 'reinspected'`, requestID).Scan(&report.ReinspectedFishingVesselCount); err != nil {
		return domain.VoyageReadiness{}, translateError("count reinspected fishing vessels", err)
	}
	if err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM fishing_voyage_vessels ri JOIN fishing_vessels s ON s.id = ri.fishing_vessel_id
		WHERE ri.request_id = ? AND s.state = 'retired'`, requestID).Scan(&report.RetiredFishingVesselCount); err != nil {
		return domain.VoyageReadiness{}, translateError("count retired fishing vessels", err)
	}
	if err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM fishing_voyage_vessels ri JOIN fishing_vessels s ON s.id = ri.fishing_vessel_id
		WHERE ri.request_id = ? AND s.state = 'quarantined'`, requestID).Scan(&report.QuarantinedCount); err != nil {
		return domain.VoyageReadiness{}, translateError("count quarantined fishing vessels", err)
	}
	var pending int
	if err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM catch_landing_tasks WHERE request_id = ? AND status = 'pending'`, requestID).Scan(&pending); err != nil {
		return domain.VoyageReadiness{}, translateError("count pending catch_landing_tasks", err)
	}
	report.PendingCatchLanding = pending > 0
	var open int
	if err := q.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM catch_anomalies WHERE request_id = ? AND status IN ('open', 'reviewing')`, requestID).Scan(&open); err != nil {
		return domain.VoyageReadiness{}, translateError("count open catch_anomalies", err)
	}
	report.OpenCatchAnomaly = open > 0
	var lastSample sql.NullString
	if err := q.q.QueryRowContext(ctx, `SELECT MAX(recorded_at) FROM catch_declarations WHERE request_id = ?`, requestID).Scan(&lastSample); err != nil {
		return domain.VoyageReadiness{}, translateError("get last declaration", err)
	}
	if lastSample.Valid {
		parsed, err := parseTime(lastSample.String)
		if err != nil {
			return domain.VoyageReadiness{}, err
		}
		report.LastDeclarationAt = &parsed
	}
	report.Evaluate()
	return report.Clone(), nil
}

func (q *queries) latestSampleAt(ctx context.Context, requestID string) (time.Time, error) {
	var raw string
	if err := q.q.QueryRowContext(ctx, `SELECT recorded_at FROM catch_declarations WHERE request_id = ? ORDER BY recorded_at DESC LIMIT 1`, requestID).Scan(&raw); err != nil {
		return time.Time{}, translateError("get latest declaration", err)
	}
	parsed, err := parseTime(raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse latest declaration: %w", err)
	}
	return parsed, nil
}
