package worker

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/clock"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/repository"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/storage/sqlite"
)

func workerFixture(t *testing.T) (*Worker, *sqlite.Store, context.Context, time.Time) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	fixed := clock.NewFixed(now)
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "worker.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	logger := slog.New(slog.NewTextHandler(testWriter{t}, nil))
	return New(store, fixed, time.Second, 20, logger), store, ctx, now
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) { return len(p), nil }

func TestRunOnceExpiresCatchLandingTasksAndCompletesJobs(t *testing.T) {
	worker, store, ctx, now := workerFixture(t)
	minimum, _ := domain.CatchVarianceFromKilograms(2)
	maximum, _ := domain.CatchVarianceFromKilograms(8)
	rangeValue, _ := domain.NewCatchVarianceEnvelope(minimum, maximum)
	program := domain.FishingPermit{ID: "workspace_worker", Code: "WORKER", Name: "Worker", Status: domain.FishingPermitActive, CatchVariance: rangeValue, MaxVoyageDuration: time.Hour, ComplianceReviewDeadline: time.Hour, BusinessTimezone: "UTC", Version: 1, CreatedAt: now, UpdatedAt: now}
	origin := domain.PortFacility{ID: "origin_worker", Code: "ORIGIN", Name: "Origin", Timezone: "UTC", Status: domain.PortFacilityActive, DailyLimit: 10, CutoffHour: 0, Version: 1, CreatedAt: now, UpdatedAt: now}
	destination := domain.PortFacility{ID: "dest_worker", Code: "DEST", Name: "Destination", Timezone: "UTC", Status: domain.PortFacilityActive, DailyLimit: 10, CutoffHour: 0, Version: 1, CreatedAt: now, UpdatedAt: now}
	support_fleet := domain.SupportFleet{ID: "box_worker", FleetCode: "BOX-W", State: domain.SupportFleetDeployed, CargoCapacityKg: 1000, CertificationDueAt: now.Add(time.Hour), LastInspectionAt: now, AssignedVoyageID: "ship_worker", Version: 1, CreatedAt: now, UpdatedAt: now}
	run := domain.FishingVoyage{ID: "ship_worker", FishingPermitID: program.ID, DeparturePortID: origin.ID, LandingPortID: destination.ID, SupportFleetID: support_fleet.ID, VoyageCode: "SHIP-W", State: domain.FishingVoyageDeparted, DepartureWindowOpensAt: now, LandingDeadlineAt: now.Add(time.Hour), TotalHoldCapacityKg: 1, Version: 1, CreatedAt: now, UpdatedAt: now}
	catch_landing_task := domain.CatchLandingTask{ID: "catch_landing_task_worker", FishingVoyageID: run.ID, CoordinatorID: "from", FisheriesOfficerID: "to", LandingStation: "dock", Status: domain.CatchLandingTaskPending, ExpiresAt: now.Add(-time.Minute), Version: 1, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)}
	job := domain.OutboxJob{ID: "job_worker", Kind: "fishing_voyage_planned", AggregateID: run.ID, Payload: []byte(`{"id":"ship_worker"}`), Status: domain.JobPending, MaxAttempts: 3, AvailableAt: now.Add(-time.Minute), CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)}
	if err := store.WithTx(ctx, func(tx repository.Tx) error {
		for _, entity := range []any{program, origin, destination, support_fleet, run, catch_landing_task, job} {
			switch value := entity.(type) {
			case domain.FishingPermit:
				if err := tx.InsertFishingPermit(ctx, value); err != nil {
					return err
				}
			case domain.PortFacility:
				if err := tx.InsertPortFacility(ctx, value); err != nil {
					return err
				}
			case domain.SupportFleet:
				if err := tx.InsertSupportFleet(ctx, value); err != nil {
					return err
				}
			case domain.FishingVoyage:
				if err := tx.InsertFishingVoyage(ctx, value); err != nil {
					return err
				}
			case domain.CatchLandingTask:
				if err := tx.InsertCatchLandingTask(ctx, value); err != nil {
					return err
				}
			case domain.OutboxJob:
				if err := tx.InsertJob(ctx, value); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := worker.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.Read(ctx, func(reader repository.Reader) error {
		catch_landing_task, err := reader.GetCatchLandingTask(ctx, catch_landing_task.ID)
		if err != nil {
			return err
		}
		if catch_landing_task.Status != domain.CatchLandingTaskExpired {
			t.Fatalf("catch_landing_task = %+v", catch_landing_task)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRunOnceHonorsCancellation(t *testing.T) {
	worker, _, _, _ := workerFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := worker.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v", err)
	}
}
