package sqlite

import (
	"context"
	"fmt"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/repository"
)

func (q *queries) GetPlatformSummary(ctx context.Context) (repository.PlatformSummary, error) {
	var summary repository.PlatformSummary
	queries := []struct {
		name   string
		target *int
		sql    string
	}{
		{"active fishing_permits", &summary.FishingPermitsActive, `SELECT COUNT(*) FROM fishing_permits WHERE status = 'active'`},
		{"flight-ready fishing vessels", &summary.FishingVesselsSeaReady, `SELECT COUNT(*) FROM fishing_vessels WHERE state = 'sea_ready'`},
		{"at_sea fishing vessels", &summary.FishingVesselsMaterializing, `SELECT COUNT(*) FROM fishing_vessels WHERE state = 'at_sea'`},
		{"quarantined fishing vessels", &summary.FishingVesselsQuarantined, `SELECT COUNT(*) FROM fishing_vessels WHERE state = 'quarantined'`},
		{"standby support fleets", &summary.SupportFleetsStandby, `SELECT COUNT(*) FROM support_fleets WHERE state = 'standby'`},
		{"active voyage requests", &summary.FishingVoyagesActive, `SELECT COUNT(*) FROM fishing_voyages WHERE state IN ('planned', 'cleared', 'at_sea', 'landed')`},
		{"open catch_anomalies", &summary.OpenLandingAnomalies, `SELECT COUNT(*) FROM catch_anomalies WHERE status IN ('open', 'reviewing')`},
		{"pending catch_landing_tasks", &summary.PendingCatchLandings, `SELECT COUNT(*) FROM catch_landing_tasks WHERE status = 'pending'`},
		{"failed jobs", &summary.FailedJobs, `SELECT COUNT(*) FROM outbox_jobs WHERE status IN ('failed', 'dead')`},
	}
	for _, item := range queries {
		if err := q.q.QueryRowContext(ctx, item.sql).Scan(item.target); err != nil {
			return repository.PlatformSummary{}, fmt.Errorf("count %s: %w", item.name, err)
		}
	}
	return summary, nil
}
