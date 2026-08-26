package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/repository"
)

func testStore(t *testing.T) (*Store, context.Context, time.Time) {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "fishingops.db")
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	return store, ctx, now
}

func seedCatalog(t *testing.T, store *Store, ctx context.Context, now time.Time) (domain.FishingPermit, domain.PortFacility, domain.PortFacility, domain.SupportFleet, domain.FishingVessel) {
	t.Helper()
	minimum, _ := domain.CatchVarianceFromKilograms(0.8)
	maximum, _ := domain.CatchVarianceFromKilograms(0.99)
	rangeValue, _ := domain.NewCatchVarianceEnvelope(minimum, maximum)
	program := domain.FishingPermit{ID: "program_1", Code: "RLV-1", Name: "Western Pacific tuna program", Status: domain.FishingPermitActive, CatchVariance: rangeValue, MaxVoyageDuration: 24 * time.Hour, ComplianceReviewDeadline: 4 * time.Hour, BusinessTimezone: "Asia/Shanghai", Version: 1, CreatedAt: now, UpdatedAt: now}
	origin := domain.PortFacility{ID: "port_facility_1", Code: "ZONE-1", Name: "Feature source", Timezone: "Asia/Shanghai", Status: domain.PortFacilityActive, DailyLimit: 10, CutoffHour: 6, Version: 1, CreatedAt: now, UpdatedAt: now}
	destination := domain.PortFacility{ID: "port_facility_2", Code: "ZONE-2", Name: "Mission target", Timezone: "Asia/Shanghai", Status: domain.PortFacilityActive, DailyLimit: 10, CutoffHour: 6, Version: 1, CreatedAt: now, UpdatedAt: now}
	support_fleet := domain.SupportFleet{ID: "fleet_1", FleetCode: "SEA-GUARDIAN-1", State: domain.SupportFleetStandby, CargoCapacityKg: 1000, CertificationDueAt: now.Add(48 * time.Hour), LastInspectionAt: now, Version: 1, CreatedAt: now, UpdatedAt: now}
	batch := domain.FishingVessel{ID: "stage_1", FishingPermitID: program.ID, DeparturePortID: origin.ID, RegistryNumber: "BST-1", VesselClass: "tianque-r1", VoyageCount: 2, HoldCapacityKg: 100, State: domain.FishingVesselSeaReady, ExpiresAt: now.Add(48 * time.Hour), Version: 1, CreatedAt: now, UpdatedAt: now}
	err := store.WithTx(ctx, func(tx repository.Tx) error {
		if err := tx.InsertFishingPermit(ctx, program); err != nil {
			return err
		}
		if err := tx.InsertPortFacility(ctx, origin); err != nil {
			return err
		}
		if err := tx.InsertPortFacility(ctx, destination); err != nil {
			return err
		}
		if err := tx.InsertSupportFleet(ctx, support_fleet); err != nil {
			return err
		}
		return tx.InsertFishingVessel(ctx, batch)
	})
	if err != nil {
		t.Fatalf("seed catalog: %v", err)
	}
	return program, origin, destination, support_fleet, batch
}

func TestOpenAppliesMigrationsAndEnablesForeignKeys(t *testing.T) {
	store, ctx, _ := testStore(t)
	if err := store.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	var tableCount int
	if err := store.Read(ctx, func(reader repository.Reader) error {
		_, err := reader.GetFishingPermit(ctx, "missing")
		if !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`).Scan(&tableCount); err != nil {
		t.Fatal(err)
	}
	if tableCount < 15 {
		t.Fatalf("table count = %d, want at least 15", tableCount)
	}
	var foreignKeys int
	if err := store.db.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d", foreignKeys)
	}
}

func TestTransactionRollsBackAllEntities(t *testing.T) {
	store, ctx, now := testStore(t)
	minimum, _ := domain.CatchVarianceFromKilograms(2)
	maximum, _ := domain.CatchVarianceFromKilograms(8)
	rangeValue, _ := domain.NewCatchVarianceEnvelope(minimum, maximum)
	program := domain.FishingPermit{ID: "workspace_roll", Code: "ROLL", Name: "Rollback", Status: domain.FishingPermitActive, CatchVariance: rangeValue, MaxVoyageDuration: time.Hour, ComplianceReviewDeadline: time.Hour, BusinessTimezone: "UTC", Version: 1, CreatedAt: now, UpdatedAt: now}
	err := store.WithTx(ctx, func(tx repository.Tx) error {
		if err := tx.InsertFishingPermit(ctx, program); err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	if err == nil {
		t.Fatal("rollback transaction returned nil")
	}
	if err := store.Read(ctx, func(reader repository.Reader) error {
		_, err := reader.GetFishingPermit(ctx, program.ID)
		return err
	}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("program after rollback error = %v", err)
	}
}

func TestRepositoryReadsAndDeepCopiesFishingVessel(t *testing.T) {
	store, ctx, now := testStore(t)
	_, origin, _, _, batch := seedCatalog(t, store, ctx, now)
	got, err := storeReadFishingVessel(store, ctx, batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.DeparturePortID != origin.ID || got.State != domain.FishingVesselSeaReady {
		t.Fatalf("fishing vessel = %+v", got)
	}
	got.QuarantineNote = "local mutation"
	again, err := storeReadFishingVessel(store, ctx, batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if again.QuarantineNote != "" {
		t.Fatalf("stored fishing vessel was mutated: %+v", again)
	}
}

func storeReadFishingVessel(store *Store, ctx context.Context, id string) (domain.FishingVessel, error) {
	var result domain.FishingVessel
	err := store.Read(ctx, func(reader repository.Reader) error {
		var err error
		result, err = reader.GetFishingVessel(ctx, id)
		return err
	})
	return result, err
}

func TestOptimisticVersionRejectsStaleUpdate(t *testing.T) {
	store, ctx, now := testStore(t)
	_, _, _, _, batch := seedCatalog(t, store, ctx, now)
	first, err := storeReadFishingVessel(store, ctx, batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	second := first
	first.State = domain.FishingVesselAssigned
	first.UpdatedAt = now.Add(time.Minute)
	if err := store.WithTx(ctx, func(tx repository.Tx) error { return tx.UpdateFishingVessel(ctx, first, first.Version) }); err != nil {
		t.Fatal(err)
	}
	second.State = domain.FishingVesselAssigned
	second.UpdatedAt = now.Add(2 * time.Minute)
	err = store.WithTx(ctx, func(tx repository.Tx) error { return tx.UpdateFishingVessel(ctx, second, second.Version) })
	if !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("stale update error = %v", err)
	}
}

func TestFishingVoyageFilterPaginationAndOrdering(t *testing.T) {
	store, ctx, now := testStore(t)
	program, origin, destination, support_fleet, batch := seedCatalog(t, store, ctx, now)
	secondFishingVessel := batch
	secondFishingVessel.ID = "stage_2"
	secondFishingVessel.RegistryNumber = "EXT-2"
	if err := store.WithTx(ctx, func(tx repository.Tx) error { return tx.InsertFishingVessel(ctx, secondFishingVessel) }); err != nil {
		t.Fatal(err)
	}
	fishing_voyages := []domain.FishingVoyage{
		{ID: "ship_1", FishingPermitID: program.ID, DeparturePortID: origin.ID, LandingPortID: destination.ID, SupportFleetID: support_fleet.ID, VoyageCode: "REF-1", State: domain.FishingVoyagePlanned, DepartureWindowOpensAt: now.Add(time.Hour), LandingDeadlineAt: now.Add(2 * time.Hour), TotalHoldCapacityKg: 100, Version: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "ship_2", FishingPermitID: program.ID, DeparturePortID: origin.ID, LandingPortID: destination.ID, SupportFleetID: support_fleet.ID, VoyageCode: "REF-2", State: domain.FishingVoyageCleared, DepartureWindowOpensAt: now.Add(2 * time.Hour), LandingDeadlineAt: now.Add(3 * time.Hour), TotalHoldCapacityKg: 100, Version: 1, CreatedAt: now, UpdatedAt: now},
	}
	if err := store.WithTx(ctx, func(tx repository.Tx) error {
		for _, run := range fishing_voyages {
			if err := tx.InsertFishingVoyage(ctx, run); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var page repository.FishingVoyagePage
	err := store.Read(ctx, func(reader repository.Reader) error {
		var err error
		page, err = reader.ListFishingVoyages(ctx, repository.FishingVoyageFilter{Page: repository.PageRequest{Limit: 1, Sort: "departure_window_opens_at"}, FishingPermitID: program.ID})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || len(page.Items) != 1 || page.Items[0].ID != "ship_1" {
		t.Fatalf("page = %+v", page)
	}
}

func TestIdempotencyRecordCopiesResponse(t *testing.T) {
	store, ctx, now := testStore(t)
	record := repository.IdempotencyRecord{Scope: "scope", Key: "key", RequestHash: "hash", ResponseCode: 201, ResponseBody: []byte("body"), ExpiresAt: now.Add(time.Hour), CreatedAt: now}
	if err := store.WithTx(ctx, func(tx repository.Tx) error { return tx.PutIdempotency(ctx, record) }); err != nil {
		t.Fatal(err)
	}
	var got repository.IdempotencyRecord
	if err := store.Read(ctx, func(reader repository.Reader) error {
		var err error
		got, err = reader.GetIdempotency(ctx, record.Scope, record.Key)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	got.ResponseBody[0] = 'B'
	var again repository.IdempotencyRecord
	if err := store.Read(ctx, func(reader repository.Reader) error {
		var err error
		again, err = reader.GetIdempotency(ctx, record.Scope, record.Key)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if string(again.ResponseBody) != "body" {
		t.Fatalf("response body = %q", again.ResponseBody)
	}
}

func TestExpiredIdempotencyRecordCanBeReplaced(t *testing.T) {
	store, ctx, now := testStore(t)
	expired := repository.IdempotencyRecord{Scope: "voyage", Key: "reuse", RequestHash: "old", ResponseCode: 201, ResponseBody: []byte("old response"), ExpiresAt: now, CreatedAt: now.Add(-time.Hour)}
	replacement := repository.IdempotencyRecord{Scope: "voyage", Key: "reuse", RequestHash: "new", ResponseCode: 201, ResponseBody: []byte("new response"), ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now.Add(time.Minute)}
	if err := store.WithTx(ctx, func(tx repository.Tx) error {
		if err := tx.PutIdempotency(ctx, expired); err != nil {
			return err
		}
		return tx.PutIdempotency(ctx, replacement)
	}); err != nil {
		t.Fatal(err)
	}
	var got repository.IdempotencyRecord
	if err := store.Read(ctx, func(reader repository.Reader) error {
		var err error
		got, err = reader.GetIdempotency(ctx, replacement.Scope, replacement.Key)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if got.RequestHash != replacement.RequestHash || string(got.ResponseBody) != string(replacement.ResponseBody) || !got.ExpiresAt.Equal(replacement.ExpiresAt) {
		t.Fatalf("replacement record = %+v", got)
	}
}

func TestOutboxClaimRetryAndCompletion(t *testing.T) {
	store, ctx, now := testStore(t)
	job := domain.OutboxJob{ID: "job_1", Kind: "fishing_voyage_planned", AggregateID: "ship_1", Payload: []byte("{}"), Status: domain.JobPending, MaxAttempts: 2, AvailableAt: now, CreatedAt: now, UpdatedAt: now}
	if err := store.WithTx(ctx, func(tx repository.Tx) error { return tx.InsertJob(ctx, job) }); err != nil {
		t.Fatal(err)
	}
	var claimed []domain.OutboxJob
	if err := store.WithTx(ctx, func(tx repository.Tx) error {
		var err error
		claimed, err = tx.ClaimJobs(ctx, now, 10)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 1 || claimed[0].Attempts != 1 || claimed[0].Status != domain.JobRunning {
		t.Fatalf("claimed = %+v", claimed)
	}
	if err := store.WithTx(ctx, func(tx repository.Tx) error {
		return tx.RetryJob(ctx, job.ID, now.Add(time.Minute), "temporary", false)
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.WithTx(ctx, func(tx repository.Tx) error {
		jobs, err := tx.ClaimJobs(ctx, now.Add(2*time.Minute), 10)
		if err != nil || len(jobs) != 1 {
			return errors.New("job was not re-claimed")
		}
		return tx.CompleteJob(ctx, jobs[0].ID, now.Add(2*time.Minute))
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRestartRecoversPersistedRows(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "restart.db")
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, _, batch := seedCatalog(t, store, ctx, now)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	got, err := storeReadFishingVessel(reopened, ctx, batch.ID)
	if err != nil || got.ID != batch.ID {
		t.Fatalf("recovered fishing vessel = %+v, error = %v", got, err)
	}
}

func TestForeignKeyRejectsUnknownFishingPermit(t *testing.T) {
	store, ctx, now := testStore(t)
	batch := domain.FishingVessel{ID: "orphan", FishingPermitID: "missing", DeparturePortID: "missing", RegistryNumber: "BST", VesselClass: "tianque-r1", VoyageCount: 1, HoldCapacityKg: 1, State: domain.FishingVesselBerthed, ExpiresAt: now.Add(time.Hour), Version: 1, CreatedAt: now, UpdatedAt: now}
	err := store.WithTx(ctx, func(tx repository.Tx) error { return tx.InsertFishingVessel(ctx, batch) })
	if err == nil {
		t.Fatal("orphan insert succeeded")
	}
}

func TestReadinessCountsRelatedRows(t *testing.T) {
	store, ctx, now := testStore(t)
	program, origin, destination, support_fleet, batch := seedCatalog(t, store, ctx, now)
	run := domain.FishingVoyage{ID: "ship_report", FishingPermitID: program.ID, DeparturePortID: origin.ID, LandingPortID: destination.ID, SupportFleetID: support_fleet.ID, VoyageCode: "REPORT-1", State: domain.FishingVoyageLanded, DepartureWindowOpensAt: now, LandingDeadlineAt: now.Add(time.Hour), TotalHoldCapacityKg: batch.HoldCapacityKg, Version: 1, CreatedAt: now, UpdatedAt: now}
	if err := store.WithTx(ctx, func(tx repository.Tx) error {
		if err := tx.InsertFishingVoyage(ctx, run); err != nil {
			return err
		}
		batch.State = domain.FishingVesselLanded
		batch.FishingVoyageID = run.ID
		if err := tx.UpdateFishingVessel(ctx, batch, batch.Version); err != nil {
			return err
		}
		return tx.InsertFishingVoyageVesselLink(ctx, domain.FishingVoyageVesselLink{FishingVoyageID: run.ID, FishingVesselID: batch.ID, AddedAt: now})
	}); err != nil {
		t.Fatal(err)
	}
	var report domain.VoyageReadiness
	if err := store.Read(ctx, func(reader repository.Reader) error {
		var err error
		report, err = reader.GetVoyageReadiness(ctx, run.ID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if report.ExpectedFishingVesselCount != 1 || report.LandedFishingVesselCount != 1 || !report.Complete {
		t.Fatalf("report = %+v", report)
	}
}
