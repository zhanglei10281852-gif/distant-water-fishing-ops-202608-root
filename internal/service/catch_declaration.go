package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/identity"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/repository"
)

type CatchDeclarationService struct{ dependencies }

type SubmitCatchDeclarationInput struct {
	FishingVoyageID string
	SpeciesCode     string
	Sequence        int64
	CatchVariance   domain.GramCatchVariance
	RecordedAt      time.Time
}

func (s *CatchDeclarationService) SubmitCatchDeclaration(ctx context.Context, input SubmitCatchDeclarationInput) (domain.CatchDeclaration, *domain.CatchAnomaly, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleVesselCaptain, domain.RoleVoyageCoordinator); err != nil {
		return domain.CatchDeclaration{}, nil, err
	}
	now := s.clock.Now()
	declaration := domain.CatchDeclaration{ID: identity.New("obs"), FishingVoyageID: input.FishingVoyageID, SpeciesCode: input.SpeciesCode, Sequence: input.Sequence, CatchVariance: input.CatchVariance, RecordedAt: input.RecordedAt.UTC(), ReceivedAt: now}
	if err := declaration.Validate(); err != nil {
		return domain.CatchDeclaration{}, nil, err
	}
	var catch_anomaly *domain.CatchAnomaly
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		run, err := tx.GetFishingVoyage(ctx, input.FishingVoyageID)
		if err != nil {
			return err
		}
		program, err := tx.GetFishingPermit(ctx, run.FishingPermitID)
		if err != nil {
			return err
		}
		if run.State != domain.FishingVoyageDeparted && run.State != domain.FishingVoyageLanded {
			return domain.ConflictError{Resource: "fishing_voyage", Reason: "catch declaration requires a departed or landed voyage"}
		}
		if err := tx.InsertCatchDeclaration(ctx, declaration); err != nil {
			return err
		}
		if program.CatchVariance.Contains(declaration.CatchVariance) {
			return s.audit.Record(ctx, tx, "catch_declaration_recorded", "fishing_voyage", run.ID, "success", map[string]string{"in_range": "true"})
		}
		active, err := tx.GetActiveCatchAnomaly(ctx, run.ID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		if errors.Is(err, domain.ErrNotFound) {
			active = domain.CatchAnomaly{ID: identity.New("drift"), FishingVoyageID: run.ID, Status: domain.CatchAnomalyOpen, ReviewDueAt: now.Add(program.ComplianceReviewDeadline), Version: 1, CreatedAt: now, UpdatedAt: now}
			active.Include(declaration, now)
			if err := tx.InsertCatchAnomaly(ctx, active); err != nil {
				return err
			}
		} else {
			before := active.Version
			active.Include(declaration, now)
			if err := tx.UpdateCatchAnomaly(ctx, active, before); err != nil {
				return err
			}
		}
		items, err := tx.ListFishingVoyageVessels(ctx, run.ID)
		if err != nil {
			return err
		}
		for _, batch := range items {
			if batch.State == domain.FishingVesselAtSea || batch.State == domain.FishingVesselLanded {
				batch.State = domain.FishingVesselQuarantined
				batch.QuarantineNote = fmt.Sprintf("catch anomaly %s", active.ID)
				batch.UpdatedAt = now
				if err := tx.UpdateFishingVessel(ctx, batch, batch.Version); err != nil {
					return err
				}
			}
		}
		payload := []byte(active.ID)
		if err := tx.InsertJob(ctx, domain.OutboxJob{ID: identity.New("job"), Kind: "catch_anomaly_review", AggregateID: active.ID, Payload: payload, Status: domain.JobPending, MaxAttempts: 5, AvailableAt: now, CreatedAt: now, UpdatedAt: now}); err != nil {
			return err
		}
		catch_anomaly = &active
		return s.audit.Record(ctx, tx, "catch_anomaly_opened", "catch_anomaly", active.ID, "success", map[string]string{"fishing_voyage_id": run.ID})
	})
	return declaration, catch_anomaly, err
}
