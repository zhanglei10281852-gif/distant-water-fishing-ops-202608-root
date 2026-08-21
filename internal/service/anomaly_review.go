package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/identity"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/repository"
)

type ReviewService struct{ dependencies }

func (s *ReviewService) StartReview(ctx context.Context, catch_anomalyID string) (domain.CatchAnomaly, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleFisheriesOfficer); err != nil {
		return domain.CatchAnomaly{}, err
	}
	var result domain.CatchAnomaly
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		catch_anomaly, err := tx.GetCatchAnomaly(ctx, catch_anomalyID)
		if err != nil {
			return err
		}
		before := catch_anomaly.Version
		if err := catch_anomaly.StartReview(s.clock.Now()); err != nil {
			return err
		}
		if err := tx.UpdateCatchAnomaly(ctx, catch_anomaly, before); err != nil {
			return err
		}
		result = catch_anomaly
		return s.audit.Record(ctx, tx, "catch_anomaly_review_started", "catch_anomaly", catch_anomaly.ID, "success", nil)
	})
	return result, err
}

type DecideInput struct {
	CatchAnomalyID string
	Decision       domain.CatchAnomalyStatus
	Rationale      string
}

func (s *ReviewService) Decide(ctx context.Context, input DecideInput) (domain.CatchAnomaly, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleFisheriesOfficer); err != nil {
		return domain.CatchAnomaly{}, err
	}
	if strings.TrimSpace(input.Rationale) == "" {
		return domain.CatchAnomaly{}, domain.FieldError{Field: "rationale", Message: "is required"}
	}
	var result domain.CatchAnomaly
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		catch_anomaly, err := tx.GetCatchAnomaly(ctx, input.CatchAnomalyID)
		if err != nil {
			return err
		}
		before := catch_anomaly.Version
		if err := catch_anomaly.Decide(input.Decision, s.clock.Now()); err != nil {
			return err
		}
		if err := tx.UpdateCatchAnomaly(ctx, catch_anomaly, before); err != nil {
			return err
		}
		run, err := tx.GetFishingVoyage(ctx, catch_anomaly.FishingVoyageID)
		if err != nil {
			return err
		}
		items, err := tx.ListFishingVoyageVessels(ctx, run.ID)
		if err != nil {
			return err
		}
		now := s.clock.Now()
		for _, batch := range items {
			switch input.Decision {
			case domain.CatchAnomalyCleared:
				if batch.State != domain.FishingVesselQuarantined {
					continue
				}
				batch.State = domain.FishingVesselReinspected
				batch.QuarantineNote = ""
			case domain.CatchAnomalyRejected:
				if batch.State != domain.FishingVesselQuarantined {
					continue
				}
				batch.State = domain.FishingVesselRetired
				batch.QuarantineNote = strings.TrimSpace(input.Rationale)
			default:
				return fmt.Errorf("unsupported review decision: %w", domain.ErrValidation)
			}
			batch.UpdatedAt = now
			if err := tx.UpdateFishingVessel(ctx, batch, batch.Version); err != nil {
				return err
			}
		}
		decision := domain.AnomalyDisposition{ID: identity.New("decision"), CatchAnomalyID: catch_anomaly.ID, Reviewer: principal.UserID, Decision: input.Decision, Rationale: strings.TrimSpace(input.Rationale), CreatedAt: now}
		if err := tx.InsertAnomalyDisposition(ctx, decision); err != nil {
			return err
		}
		result = catch_anomaly
		return s.audit.Record(ctx, tx, "catch_anomaly_decided", "catch_anomaly", catch_anomaly.ID, "success", map[string]string{"decision": string(input.Decision)})
	})
	return result, err
}

func (s *ReviewService) EnsureReviewable(ctx context.Context, catch_anomalyID string) error {
	return s.store.Read(ctx, func(reader repository.Reader) error {
		catch_anomaly, err := reader.GetCatchAnomaly(ctx, catch_anomalyID)
		if err != nil {
			return err
		}
		if catch_anomaly.Status != domain.CatchAnomalyOpen && catch_anomaly.Status != domain.CatchAnomalyReviewing {
			return errors.New("catch_anomaly is already decided")
		}
		return nil
	})
}
