package domain

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

func TestCatchVarianceEnvelopeBoundaries(t *testing.T) {
	min := GramCatchVariance(2000)
	max := GramCatchVariance(8000)
	rangeValue, err := NewCatchVarianceEnvelope(min, max)
	if err != nil {
		t.Fatalf("create range: %v", err)
	}
	tests := []struct {
		name  string
		value GramCatchVariance
		want  bool
	}{
		{name: "minimum included", value: min, want: true},
		{name: "middle included", value: 5000, want: true},
		{name: "maximum included", value: max, want: true},
		{name: "below minimum", value: 1999, want: false},
		{name: "above maximum", value: 8001, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := rangeValue.Contains(test.value); got != test.want {
				t.Fatalf("Contains(%d) = %v, want %v", test.value, got, test.want)
			}
		})
	}
}

func TestCatchVarianceParsingRejectsInvalidValues(t *testing.T) {
	for _, value := range []float64{-197, 101, math.NaN()} {
		_, err := CatchVarianceFromKilograms(value)
		if err == nil {
			t.Fatalf("CatchVarianceFromKilograms(%v) succeeded", value)
		}
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("error %v does not wrap validation", err)
		}
	}
}

func TestFishingVesselTransitionTable(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	base := FishingVessel{State: FishingVesselBerthed, ExpiresAt: now.Add(24 * time.Hour)}
	cases := []struct {
		name string
		from FishingVesselState
		to   FishingVesselState
		want bool
	}{
		{"berthed to flight ready", FishingVesselBerthed, FishingVesselSeaReady, true},
		{"flight ready to assigned", FishingVesselSeaReady, FishingVesselAssigned, true},
		{"assigned to in flight", FishingVesselAssigned, FishingVesselAtSea, true},
		{"in flight to landed", FishingVesselAtSea, FishingVesselLanded, true},
		{"landed to reinspected", FishingVesselLanded, FishingVesselReinspected, true},
		{"landed to quarantine", FishingVesselLanded, FishingVesselQuarantined, true},
		{"quarantine to retired", FishingVesselQuarantined, FishingVesselRetired, true},
		{"berthed to reinspected", FishingVesselBerthed, FishingVesselReinspected, false},
		{"reinspected to flight ready", FishingVesselReinspected, FishingVesselSeaReady, false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			batch := base
			batch.State = test.from
			err := batch.Transition(test.to, now)
			if (err == nil) != test.want {
				t.Fatalf("transition %s -> %s error = %v, want allowed=%v", test.from, test.to, err, test.want)
			}
			if test.want && batch.State != test.to {
				t.Fatalf("state = %s, want %s", batch.State, test.to)
			}
		})
	}
}

func TestExpiredFishingVesselCanOnlyBeDestroyedOrQuarantined(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	batch := FishingVessel{State: FishingVesselLanded, ExpiresAt: now.Add(-time.Minute)}
	if err := batch.Transition(FishingVesselReinspected, now); !errors.Is(err, ErrConflict) {
		t.Fatalf("expired release error = %v, want conflict", err)
	}
	batch.State = FishingVesselLanded
	if err := batch.Transition(FishingVesselQuarantined, now); err != nil {
		t.Fatalf("expired quarantine failed: %v", err)
	}
}

func TestFishingVesselQuarantineBelongsToCatchAnomaly(t *testing.T) {
	vessel := FishingVessel{State: FishingVesselQuarantined, QuarantineNote: "catch anomaly anomaly-17"}
	if !vessel.IsQuarantinedForCatchAnomaly("anomaly-17") {
		t.Fatal("matching catch anomaly quarantine was not recognized")
	}
	if vessel.IsQuarantinedForCatchAnomaly("anomaly-18") {
		t.Fatal("quarantine leaked to another catch anomaly")
	}
	vessel.QuarantineNote = "certificate inspection hold"
	if vessel.IsQuarantinedForCatchAnomaly("anomaly-17") {
		t.Fatal("non-catch quarantine was treated as owned by anomaly")
	}
}

func TestFishingVoyageTransitionSetsTimestamps(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	run := FishingVoyage{State: FishingVoyagePlanned, DepartureWindowOpensAt: now, LandingDeadlineAt: now.Add(2 * time.Hour)}
	if err := run.Transition(FishingVoyageCleared, now); err != nil {
		t.Fatalf("pack: %v", err)
	}
	if err := run.Transition(FishingVoyageDeparted, now.Add(time.Minute)); err != nil {
		t.Fatalf("start: %v", err)
	}
	if run.DepartedAt == nil || !run.DepartedAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("departed_at = %v", run.DepartedAt)
	}
	if err := run.Transition(FishingVoyageLanded, now.Add(time.Hour)); err != nil {
		t.Fatalf("arrive: %v", err)
	}
	if err := run.Transition(FishingVoyageClosed, now.Add(2*time.Hour)); err != nil {
		t.Fatalf("close: %v", err)
	}
	if run.ClosedAt == nil {
		t.Fatal("closed_at is nil")
	}
}

func TestFishingVoyageRejectsSkippedState(t *testing.T) {
	run := FishingVoyage{State: FishingVoyagePlanned}
	err := run.Transition(FishingVoyageLanded, time.Now())
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("error = %v, want invalid transition", err)
	}
}

func TestCatchLandingTaskResolutionAndExpiry(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	catch_landing_task := CatchLandingTask{Status: CatchLandingTaskPending, ExpiresAt: now.Add(time.Hour)}
	if err := catch_landing_task.Resolve(CatchLandingTaskApproved, "seal intact", now); err != nil {
		t.Fatalf("accept: %v", err)
	}
	if catch_landing_task.Status != CatchLandingTaskApproved || catch_landing_task.ResolvedAt == nil {
		t.Fatalf("catch_landing_task after accept = %+v", catch_landing_task)
	}
	catch_landing_task = CatchLandingTask{Status: CatchLandingTaskPending, ExpiresAt: now.Add(-time.Minute)}
	if err := catch_landing_task.Resolve(CatchLandingTaskApproved, "", now); !errors.Is(err, ErrExpired) {
		t.Fatalf("expired accept error = %v", err)
	}
	if err := catch_landing_task.Resolve(CatchLandingTaskExpired, "expired", now); err != nil {
		t.Fatalf("expire: %v", err)
	}
}

func TestCatchAnomalyAggregatesSamples(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	catch_anomaly := CatchAnomaly{Status: CatchAnomalyOpen}
	declarations := []CatchDeclaration{
		{CatchVariance: 9200, RecordedAt: now.Add(10 * time.Minute)},
		{CatchVariance: 8500, RecordedAt: now.Add(5 * time.Minute)},
		{CatchVariance: 11000, RecordedAt: now.Add(20 * time.Minute)},
	}
	for _, declaration := range declarations {
		catch_anomaly.Include(declaration, now)
	}
	if catch_anomaly.DeclarationCount != 3 || catch_anomaly.Minimum != 8500 || catch_anomaly.Maximum != 11000 {
		t.Fatalf("aggregate = %+v", catch_anomaly)
	}
	if !catch_anomaly.FirstDeclarationAt.Equal(now.Add(5*time.Minute)) || !catch_anomaly.LastDeclarationAt.Equal(now.Add(20*time.Minute)) {
		t.Fatalf("declaration window = %v..%v", catch_anomaly.FirstDeclarationAt, catch_anomaly.LastDeclarationAt)
	}
}

func TestCatchAnomalyDecisionTable(t *testing.T) {
	now := time.Now().UTC()
	for _, decision := range []CatchAnomalyStatus{CatchAnomalyCleared, CatchAnomalyRejected} {
		catch_anomaly := CatchAnomaly{Status: CatchAnomalyReviewing}
		if err := catch_anomaly.Decide(decision, now); err != nil {
			t.Fatalf("decision %s: %v", decision, err)
		}
		if catch_anomaly.Status != decision {
			t.Fatalf("status = %s, want %s", catch_anomaly.Status, decision)
		}
	}
	catch_anomaly := CatchAnomaly{Status: CatchAnomalyCleared}
	if err := catch_anomaly.Decide(CatchAnomalyRejected, now); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("decide closed catch_anomaly = %v", err)
	}
}

func TestPortFacilityBusinessDayUsesCutoffAndTimezone(t *testing.T) {
	port_facility := PortFacility{Timezone: "Asia/Shanghai", CutoffHour: 6}
	before, err := port_facility.BusinessDay(time.Date(2026, 8, 18, 21, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if before != "2026-08-18" {
		t.Fatalf("business day = %s", before)
	}
	after, err := port_facility.BusinessDay(time.Date(2026, 8, 18, 22, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if after != "2026-08-19" {
		t.Fatalf("business day after cutoff = %s", after)
	}
}

func TestSupportFleetEligibility(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	base := SupportFleet{State: SupportFleetStandby, CargoCapacityKg: 1000, CertificationDueAt: now.Add(time.Hour)}
	if err := base.EligibleFor(now, 1000); err != nil {
		t.Fatalf("capacity boundary: %v", err)
	}
	if err := base.EligibleFor(now, 1001); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("capacity overflow = %v", err)
	}
	base.State = SupportFleetAssigned
	if err := base.EligibleFor(now, 1); !errors.Is(err, ErrConflict) {
		t.Fatalf("reserved support_fleet = %v", err)
	}
}

func TestReadinessEvaluate(t *testing.T) {
	report := VoyageReadiness{FishingVoyageState: FishingVoyageLanded, ExpectedFishingVesselCount: 2, LandedFishingVesselCount: 2, PendingCatchLanding: true}
	report.Evaluate()
	if report.Complete || len(report.Blockers) != 1 || report.Blockers[0] != "pending handoff task" {
		t.Fatalf("report = %+v", report)
	}
	report.PendingCatchLanding = false
	report.Evaluate()
	if !report.Complete {
		t.Fatalf("resolved report = %+v", report)
	}
}

func TestAuditAndJobCloneIsolation(t *testing.T) {
	event := AuditEvent{Metadata: map[string]string{"one": "1"}}
	clone := event.Clone()
	clone.Metadata["one"] = "2"
	if event.Metadata["one"] != "1" {
		t.Fatal("audit metadata was shared")
	}
	job := OutboxJob{Payload: []byte("payload")}
	jobClone := job.Clone()
	jobClone.Payload[0] = 'P'
	if string(job.Payload) != "payload" {
		t.Fatal("job payload was shared")
	}
}

func TestVoyageWindowChecksFishingPermitLimitAndExpiry(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	program := FishingPermit{MaxVoyageDuration: 2 * time.Hour}
	batch := FishingVessel{ExpiresAt: now.Add(4 * time.Hour)}
	valid := VoyageWindow{StartAt: now.Add(time.Hour), FinishAt: now.Add(2 * time.Hour)}
	if err := valid.Validate(program, []FishingVessel{batch}, now); err != nil {
		t.Fatalf("valid window: %v", err)
	}
	tooLong := VoyageWindow{StartAt: now.Add(time.Hour), FinishAt: now.Add(4 * time.Hour)}
	if err := tooLong.Validate(program, []FishingVessel{batch}, now); !errors.Is(err, ErrConflict) {
		t.Fatalf("long window = %v", err)
	}
	batch.ExpiresAt = now.Add(90 * time.Minute)
	if err := valid.Validate(program, []FishingVessel{batch}, now); !errors.Is(err, ErrConflict) {
		t.Fatalf("expired batch window = %v", err)
	}
}

func TestPrincipalActionMatrix(t *testing.T) {
	cases := []struct {
		role   Role
		action Action
		want   bool
	}{
		{RoleVoyageCoordinator, ActionPlanVoyage, true},
		{RoleVoyageCoordinator, ActionReviewCatchAnomaly, false},
		{RoleVesselCaptain, ActionSubmitCatchDeclaration, true},
		{RoleVesselCaptain, ActionFleetCatalogWrite, false},
		{RoleFisheriesOfficer, ActionReviewCatchAnomaly, true},
		{RoleComplianceAuditor, ActionReadAudit, true},
		{RoleComplianceAuditor, ActionManageVoyage, false},
	}
	for _, test := range cases {
		principal := Principal{Role: test.role}
		if got := principal.CanAction(test.action); got != test.want {
			t.Fatalf("%s %s = %v, want %v", test.role, test.action, got, test.want)
		}
	}
}

func TestIdentifierNormalizationAndValidation(t *testing.T) {
	if got := NormalizeCode("  data-zone-sh-01 "); got != "DATA-ZONE-SH-01" {
		t.Fatalf("normalized code = %q", got)
	}
	for _, value := range []string{"A", "with spaces", "ümlaut", "", strings.Repeat("X", 65)} {
		if err := ValidateBusinessCode("code", value); err == nil {
			t.Fatalf("invalid code %q passed", value)
		}
	}
	for _, value := range []string{"valid-key", "request-1234", strings.Repeat("x", 128)} {
		if err := ValidateIdempotencyKey(value); err != nil {
			t.Fatalf("valid idempotency key %q: %v", value, err)
		}
	}
	for _, value := range []string{"short", "line\nbreak", strings.Repeat("x", 129)} {
		if err := ValidateIdempotencyKey(value); err == nil {
			t.Fatalf("invalid idempotency key %q passed", value)
		}
	}
}

func TestTerminalStateHelpers(t *testing.T) {
	if !FishingVoyageClosed.IsTerminal() || !FishingVoyageCancelled.IsTerminal() || FishingVoyageLanded.IsTerminal() {
		t.Fatal("run terminal states are incorrect")
	}
	if !FishingVesselReinspected.IsTerminal() || !FishingVesselRetired.IsTerminal() || FishingVesselQuarantined.IsTerminal() {
		t.Fatal("fishing vessel terminal states are incorrect")
	}
	if !CatchAnomalyCleared.IsResolved() || !CatchAnomalyRejected.IsResolved() || CatchAnomalyOpen.IsResolved() {
		t.Fatal("catch_anomaly resolved states are incorrect")
	}
}
