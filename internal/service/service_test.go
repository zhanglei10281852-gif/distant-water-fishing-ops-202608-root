package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/clock"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/repository"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/requestmeta"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/storage/sqlite"
)

type serviceFixture struct {
	t                  *testing.T
	ctx                context.Context
	store              *sqlite.Store
	services           *Services
	clock              *clock.Fixed
	voyage_coordinator domain.Principal
	vessel_captain     domain.Principal
	fisheries_officer  domain.Principal
	program            domain.FishingPermit
	origin             domain.PortFacility
	destination        domain.PortFacility
	support_fleet      domain.SupportFleet
	batch              domain.FishingVessel
}

func newServiceFixture(t *testing.T) *serviceFixture {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	fixed := clock.NewFixed(now)
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "service.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	services := New(store, fixed, 4*time.Hour, 30*time.Minute)
	users := []struct {
		email string
		name  string
		role  domain.Role
	}{
		{"ops@example.test", "Ops", domain.RoleVoyageCoordinator},
		{"vessel_captain@example.test", "Recovery Controller", domain.RoleVesselCaptain},
		{"fisheries_officer@example.test", "Reviewer", domain.RoleFisheriesOfficer},
	}
	principals := make([]domain.Principal, 0, len(users))
	for _, user := range users {
		created, err := services.Auth.CreateUser(ctx, user.email, user.name, "very-secure-password", user.role)
		if err != nil {
			t.Fatalf("create user %s: %v", user.email, err)
		}
		login, err := services.Auth.Login(ctx, LoginInput{Email: user.email, Password: "very-secure-password"})
		if err != nil {
			t.Fatalf("login %s: %v", user.email, err)
		}
		if login.Principal.UserID != created.ID {
			t.Fatalf("principal user = %s, created = %s", login.Principal.UserID, created.ID)
		}
		principals = append(principals, login.Principal)
	}
	minimum, _ := domain.CatchVarianceFromKilograms(2)
	maximum, _ := domain.CatchVarianceFromKilograms(8)
	rangeValue, _ := domain.NewCatchVarianceEnvelope(minimum, maximum)
	opsCtx := requestmeta.WithPrincipal(ctx, principals[0])
	program, err := services.Catalog.CreateFishingPermit(opsCtx, domain.FishingPermit{Code: "RLV-1", Name: "Western Pacific tuna permit", CatchVariance: rangeValue, MaxVoyageDuration: 24 * time.Hour, ComplianceReviewDeadline: 4 * time.Hour, BusinessTimezone: "Asia/Shanghai"})
	if err != nil {
		t.Fatal(err)
	}
	program, err = services.Catalog.ActivateFishingPermit(opsCtx, program.ID)
	if err != nil {
		t.Fatal(err)
	}
	origin, err := services.Catalog.CreatePortFacility(opsCtx, domain.PortFacility{Code: "SITE-1", Name: "Origin", Timezone: "Asia/Shanghai", DailyLimit: 10, CutoffHour: 6})
	if err != nil {
		t.Fatal(err)
	}
	destination, err := services.Catalog.CreatePortFacility(opsCtx, domain.PortFacility{Code: "SITE-2", Name: "Destination", Timezone: "Asia/Shanghai", DailyLimit: 10, CutoffHour: 6})
	if err != nil {
		t.Fatal(err)
	}
	now = fixed.Now()
	support_fleet, err := services.Catalog.CreateSupportFleet(opsCtx, domain.SupportFleet{FleetCode: "BOX-1", CargoCapacityKg: 1000, CertificationDueAt: now.Add(48 * time.Hour), LastInspectionAt: now})
	if err != nil {
		t.Fatal(err)
	}
	batch, err := services.Catalog.RegisterFishingVessel(opsCtx, domain.FishingVessel{FishingPermitID: program.ID, DeparturePortID: origin.ID, RegistryNumber: "BST-1", VesselClass: "tianque-r1", VoyageCount: 2, HoldCapacityKg: 100, ExpiresAt: now.Add(48 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	batch, err = services.Catalog.VerifyFishingVessel(opsCtx, batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	return &serviceFixture{t: t, ctx: ctx, store: store, services: services, clock: fixed, voyage_coordinator: principals[0], vessel_captain: principals[1], fisheries_officer: principals[2], program: program, origin: origin, destination: destination, support_fleet: support_fleet, batch: batch}
}

func (f *serviceFixture) as(principal domain.Principal) context.Context {
	return requestmeta.WithPrincipal(requestmeta.WithRequestID(f.ctx, "req-test"), principal)
}

func TestAuthRejectsWrongPasswordAndHonorsLogout(t *testing.T) {
	f := newServiceFixture(t)
	if _, err := f.services.Auth.Login(f.ctx, LoginInput{Email: f.voyage_coordinator.Email, Password: "wrong-password"}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("wrong password error = %v", err)
	}
	if err := f.services.Auth.Logout(f.as(f.voyage_coordinator), f.voyage_coordinator); err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.Auth.Authenticate(f.ctx, "missing-token"); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("missing token error = %v, want unauthenticated", err)
	}
}

func TestMissionIsIdempotentAndReservesRelatedEntities(t *testing.T) {
	f := newServiceFixture(t)
	input := PlanFishingVoyageInput{FishingPermitID: f.program.ID, DeparturePortID: f.origin.ID, LandingPortID: f.destination.ID, SupportFleetID: f.support_fleet.ID, VoyageCode: "SHIP-1", FishingVesselIDs: []string{f.batch.ID}, DepartureWindowOpensAt: f.clock.Now().Add(time.Hour), LandingDeadlineAt: f.clock.Now().Add(2 * time.Hour), IdempotencyKey: "plan-key"}
	ctx := f.as(f.voyage_coordinator)
	first, err := f.services.FishingVoyages.PlanFishingVoyage(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.services.FishingVoyages.PlanFishingVoyage(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || first.VoyageCode != "SHIP-1" {
		t.Fatalf("idempotent responses differ: %+v / %+v", first, second)
	}
	if err := f.store.Read(ctx, func(reader repository.Reader) error {
		batch, err := reader.GetFishingVessel(ctx, f.batch.ID)
		if err != nil {
			return err
		}
		if batch.State != domain.FishingVesselAssigned || batch.FishingVoyageID != first.ID {
			t.Fatalf("reserved batch = %+v", batch)
		}
		support_fleet, err := reader.GetSupportFleet(ctx, f.support_fleet.ID)
		if err != nil {
			return err
		}
		if support_fleet.State != domain.SupportFleetAssigned || support_fleet.AssignedVoyageID != first.ID {
			t.Fatalf("reserved support_fleet = %+v", support_fleet)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestMissionRejectsDifferentIdempotencyPayload(t *testing.T) {
	f := newServiceFixture(t)
	input := PlanFishingVoyageInput{FishingPermitID: f.program.ID, DeparturePortID: f.origin.ID, LandingPortID: f.destination.ID, SupportFleetID: f.support_fleet.ID, VoyageCode: "SHIP-1", FishingVesselIDs: []string{f.batch.ID}, DepartureWindowOpensAt: f.clock.Now().Add(time.Hour), LandingDeadlineAt: f.clock.Now().Add(2 * time.Hour), IdempotencyKey: "plan-key"}
	ctx := f.as(f.voyage_coordinator)
	if _, err := f.services.FishingVoyages.PlanFishingVoyage(ctx, input); err != nil {
		t.Fatal(err)
	}
	input.VoyageCode = "SHIP-OTHER"
	if _, err := f.services.FishingVoyages.PlanFishingVoyage(ctx, input); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("different payload error = %v", err)
	}
}

func TestMissionLifecycleMovesFishingVesselsAndSupportFleet(t *testing.T) {
	f := newServiceFixture(t)
	ctx := f.as(f.voyage_coordinator)
	run, err := f.services.FishingVoyages.PlanFishingVoyage(ctx, PlanFishingVoyageInput{FishingPermitID: f.program.ID, DeparturePortID: f.origin.ID, LandingPortID: f.destination.ID, SupportFleetID: f.support_fleet.ID, VoyageCode: "SHIP-LIFE", FishingVesselIDs: []string{f.batch.ID}, DepartureWindowOpensAt: f.clock.Now().Add(time.Hour), LandingDeadlineAt: f.clock.Now().Add(2 * time.Hour), IdempotencyKey: "life-key"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.FishingVoyages.ClearFishingVoyage(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.FishingVoyages.DepartureFishingVoyage(f.as(f.vessel_captain), run.ID); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Read(ctx, func(reader repository.Reader) error {
		batch, err := reader.GetFishingVessel(ctx, f.batch.ID)
		if err != nil {
			return err
		}
		if batch.State != domain.FishingVesselAtSea {
			t.Fatalf("in voyage batch = %+v", batch)
		}
		support_fleet, err := reader.GetSupportFleet(ctx, f.support_fleet.ID)
		if err != nil {
			return err
		}
		if support_fleet.State != domain.SupportFleetDeployed {
			t.Fatalf("in voyage support_fleet = %+v", support_fleet)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.FishingVoyages.ConfirmFishingVoyageLanding(f.as(f.vessel_captain), run.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.FishingVoyages.CloseFishingVoyage(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
}

func TestFishingVoyageClosureRequiresResolvedOperationalReviews(t *testing.T) {
	f := newServiceFixture(t)
	run := f.planAndStart(t, "RUN-CLOSE-GUARDS")
	if _, err := f.services.FishingVoyages.ConfirmFishingVoyageLanding(f.as(f.vessel_captain), run.ID); err != nil {
		t.Fatalf("confirm landing: %v", err)
	}
	handoff, err := f.services.Handoffs.CreateCatchLandingTask(f.as(f.vessel_captain), CreateCatchLandingTaskInput{
		FishingVoyageID:    run.ID,
		CoordinatorID:      f.voyage_coordinator.UserID,
		FisheriesOfficerID: f.vessel_captain.UserID,
		LandingStation:     "Recovery deck 2",
	})
	if err != nil {
		t.Fatalf("create handoff: %v", err)
	}
	if _, err := f.services.FishingVoyages.CloseFishingVoyage(f.as(f.voyage_coordinator), run.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("close with pending handoff error = %v, want conflict", err)
	}
	if _, err := f.services.Handoffs.ResolveCatchLandingTask(f.as(f.vessel_captain), handoff.ID, true, "deck transfer complete"); err != nil {
		t.Fatalf("resolve handoff: %v", err)
	}
	if _, err := f.services.FishingVoyages.CloseFishingVoyage(f.as(f.voyage_coordinator), run.ID); err != nil {
		t.Fatalf("close after reviews resolved: %v", err)
	}
}

func TestHandoffsOnlyReceiverCanResolve(t *testing.T) {
	f := newServiceFixture(t)
	ctx := f.as(f.voyage_coordinator)
	run, err := f.services.FishingVoyages.PlanFishingVoyage(ctx, PlanFishingVoyageInput{FishingPermitID: f.program.ID, DeparturePortID: f.origin.ID, LandingPortID: f.destination.ID, SupportFleetID: f.support_fleet.ID, VoyageCode: "SHIP-HAND", FishingVesselIDs: []string{f.batch.ID}, DepartureWindowOpensAt: f.clock.Now().Add(time.Hour), LandingDeadlineAt: f.clock.Now().Add(2 * time.Hour), IdempotencyKey: "hand-key"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.FishingVoyages.ClearFishingVoyage(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.FishingVoyages.DepartureFishingVoyage(f.as(f.vessel_captain), run.ID); err != nil {
		t.Fatal(err)
	}
	catch_landing_task, err := f.services.Handoffs.CreateCatchLandingTask(f.as(f.vessel_captain), CreateCatchLandingTaskInput{FishingVoyageID: run.ID, CoordinatorID: f.voyage_coordinator.UserID, FisheriesOfficerID: f.vessel_captain.UserID, LandingStation: "Dock 2"})
	if err != nil {
		t.Fatal(err)
	}
	other := domain.Principal{UserID: "compliance_auditor-user", Role: domain.RoleComplianceAuditor}
	if _, err := f.services.Handoffs.ResolveCatchLandingTask(f.as(other), catch_landing_task.ID, true, "wrong actor"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("wrong actor error = %v", err)
	}
	if _, err := f.services.Handoffs.ResolveCatchLandingTask(f.as(f.vessel_captain), catch_landing_task.ID, true, "seal intact"); err != nil {
		t.Fatal(err)
	}
}

func (f *serviceFixture) planAndStart(t *testing.T, ref string) domain.FishingVoyage {
	t.Helper()
	run, err := f.services.FishingVoyages.PlanFishingVoyage(f.as(f.voyage_coordinator), PlanFishingVoyageInput{FishingPermitID: f.program.ID, DeparturePortID: f.origin.ID, LandingPortID: f.destination.ID, SupportFleetID: f.support_fleet.ID, VoyageCode: ref, FishingVesselIDs: []string{f.batch.ID}, DepartureWindowOpensAt: f.clock.Now().Add(time.Hour), LandingDeadlineAt: f.clock.Now().Add(2 * time.Hour), IdempotencyKey: ref + "-key"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.FishingVoyages.ClearFishingVoyage(f.as(f.voyage_coordinator), run.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.FishingVoyages.DepartureFishingVoyage(f.as(f.vessel_captain), run.ID); err != nil {
		t.Fatal(err)
	}
	return run
}

func TestCatchVarianceCatchAnomalyQuarantinesAndReviewerClears(t *testing.T) {
	f := newServiceFixture(t)
	run := f.planAndStart(t, "RUN-DRIFT")
	declaration, catch_anomaly, err := f.services.CatchDeclaration.SubmitCatchDeclaration(f.as(f.vessel_captain), SubmitCatchDeclarationInput{FishingVoyageID: run.ID, SpeciesCode: "sensor-1", Sequence: 1, CatchVariance: 12000, RecordedAt: f.clock.Now().Add(10 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if declaration.ID == "" || catch_anomaly == nil || catch_anomaly.DeclarationCount != 1 {
		t.Fatalf("declaration=%+v catch_anomaly=%+v", declaration, catch_anomaly)
	}
	if _, err := f.services.FishingVoyages.ConfirmFishingVoyageLanding(f.as(f.vessel_captain), run.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.Review.StartReview(f.as(f.fisheries_officer), catch_anomaly.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.Review.Decide(f.as(f.fisheries_officer), DecideInput{CatchAnomalyID: catch_anomaly.ID, Decision: domain.CatchAnomalyCleared, Rationale: "verified logger trace"}); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Read(f.ctx, func(reader repository.Reader) error {
		batch, err := reader.GetFishingVessel(f.ctx, f.batch.ID)
		if err != nil {
			return err
		}
		if batch.State != domain.FishingVesselReinspected {
			t.Fatalf("batch after clear = %+v", batch)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestInRangeSampleDoesNotOpenCatchAnomaly(t *testing.T) {
	f := newServiceFixture(t)
	run := f.planAndStart(t, "RUN-IN-RANGE")
	_, catch_anomaly, err := f.services.CatchDeclaration.SubmitCatchDeclaration(f.as(f.vessel_captain), SubmitCatchDeclarationInput{FishingVoyageID: run.ID, SpeciesCode: "sensor-1", Sequence: 1, CatchVariance: 5000, RecordedAt: f.clock.Now()})
	if err != nil || catch_anomaly != nil {
		t.Fatalf("in range result catch_anomaly=%+v error=%v", catch_anomaly, err)
	}
}

func TestQueryReadinessReportsBlockers(t *testing.T) {
	f := newServiceFixture(t)
	run := f.planAndStart(t, "RUN-REPORT")
	report, err := f.services.Query.GetFishingVoyageReadiness(f.as(f.voyage_coordinator), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if report.ExpectedFishingVesselCount != 1 || report.Complete {
		t.Fatalf("report = %+v", report)
	}
}

func TestContextCancellationReachesTransaction(t *testing.T) {
	f := newServiceFixture(t)
	cancelled, cancel := context.WithCancel(f.as(f.voyage_coordinator))
	cancel()
	_, err := f.services.Catalog.VerifyFishingVessel(cancelled, f.batch.ID)
	if err == nil {
		t.Fatal("cancelled command succeeded")
	}
}

func TestSupportFleetMaintenanceAndRetirementLifecycle(t *testing.T) {
	f := newServiceFixture(t)
	ctx := f.as(f.voyage_coordinator)
	maintenance, err := f.services.SupportFleets.StartMaintenance(ctx, f.support_fleet.ID)
	if err != nil || maintenance.State != domain.SupportFleetMaintenance {
		t.Fatalf("start maintenance = %+v, error=%v", maintenance, err)
	}
	f.clock.Advance(time.Hour)
	available, err := f.services.SupportFleets.CompleteMaintenance(ctx, f.support_fleet.ID)
	if err != nil || available.State != domain.SupportFleetStandby || !available.LastInspectionAt.Equal(f.clock.Now()) {
		t.Fatalf("complete maintenance = %+v, error=%v", available, err)
	}
	retired, err := f.services.SupportFleets.Retire(ctx, f.support_fleet.ID, "certification program ended")
	if err != nil || retired.State != domain.SupportFleetRetired {
		t.Fatalf("retire = %+v, error=%v", retired, err)
	}
	if _, err := f.services.SupportFleets.StartMaintenance(ctx, f.support_fleet.ID); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("clean retired error = %v", err)
	}
}

func TestBulkRegistrationReturnsPartialFailures(t *testing.T) {
	f := newServiceFixture(t)
	now := f.clock.Now()
	inputs := []domain.FishingVessel{
		{FishingPermitID: f.program.ID, DeparturePortID: f.origin.ID, RegistryNumber: "BULK-OK", VesselClass: "tianque-r1", VoyageCount: 1, HoldCapacityKg: 20, ExpiresAt: now.Add(time.Hour)},
		{FishingPermitID: f.program.ID, DeparturePortID: f.origin.ID, RegistryNumber: "", VesselClass: "tianque-r1", VoyageCount: 1, HoldCapacityKg: 20, ExpiresAt: now.Add(time.Hour)},
		{FishingPermitID: f.program.ID, DeparturePortID: f.origin.ID, RegistryNumber: "BULK-OK", VesselClass: "tianque-r1", VoyageCount: 1, HoldCapacityKg: 20, ExpiresAt: now.Add(time.Hour)},
	}
	result, err := f.services.Catalog.BulkRegisterFishingVessels(f.as(f.voyage_coordinator), inputs)
	if err != nil {
		t.Fatal(err)
	}
	if result.Succeeded != 1 || result.Failed != 2 || len(result.Items) != 3 {
		t.Fatalf("bulk result = %+v", result)
	}
	if result.Items[0].Code != "created" || result.Items[1].Code != "invalid" || result.Items[2].Code != "conflict" {
		t.Fatalf("bulk item codes = %+v", result.Items)
	}
}

func TestPlatformSummaryRequiresReadPermissionAndCountsRows(t *testing.T) {
	f := newServiceFixture(t)
	if _, err := f.services.Query.PlatformSummary(f.as(f.fisheries_officer)); err != nil {
		t.Fatalf("fisheries_officer summary: %v", err)
	}
	summary, err := f.services.Query.PlatformSummary(f.as(f.voyage_coordinator))
	if err != nil {
		t.Fatal(err)
	}
	if summary.FishingPermitsActive != 1 || summary.FishingVesselsSeaReady != 1 || summary.SupportFleetsStandby != 1 {
		t.Fatalf("summary = %+v", summary)
	}
	if _, err := f.services.Query.PlatformSummary(f.as(domain.Principal{UserID: "vessel_captain", Role: domain.RoleVesselCaptain})); err != nil {
		t.Fatalf("vessel_captain read summary: %v", err)
	}
}
