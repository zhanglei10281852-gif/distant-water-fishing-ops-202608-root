package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/idempotency"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/identity"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/repository"
)

type VoyageService struct{ dependencies }

type PlanFishingVoyageInput struct {
	FishingPermitID        string
	DeparturePortID        string
	LandingPortID          string
	SupportFleetID         string
	VoyageCode             string
	FishingVesselIDs       []string
	DepartureWindowOpensAt time.Time
	LandingDeadlineAt      time.Time
	IdempotencyKey         string
}

func (s *VoyageService) PlanFishingVoyage(ctx context.Context, input PlanFishingVoyageInput) (domain.FishingVoyage, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleVoyageCoordinator); err != nil {
		return domain.FishingVoyage{}, err
	}
	if len(input.FishingVesselIDs) == 0 {
		return domain.FishingVoyage{}, domain.FieldError{Field: "fishing_vessel_ids", Message: "at least one fishing vessel is required"}
	}
	if err := domain.ValidateIdempotencyKey(input.IdempotencyKey); err != nil {
		return domain.FishingVoyage{}, err
	}
	hash, err := idempotency.Hash(input)
	if err != nil {
		return domain.FishingVoyage{}, err
	}
	var run domain.FishingVoyage
	err = s.store.WithTx(ctx, func(tx repository.Tx) error {
		if existing, err := tx.GetIdempotency(ctx, "plan-mission", input.IdempotencyKey); err == nil {
			if existing.RequestHash != hash {
				return domain.ConflictError{Resource: "idempotency_key", Reason: "request payload differs"}
			}
			return json.Unmarshal(existing.ResponseBody, &run)
		} else if !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		program, err := tx.GetFishingPermit(ctx, input.FishingPermitID)
		if err != nil {
			return err
		}
		if !program.CanAcceptFishingVoyages() {
			return domain.ConflictError{Resource: "program", Reason: "program is not active"}
		}
		source, err := tx.GetPortFacility(ctx, input.DeparturePortID)
		if err != nil {
			return err
		}
		target, err := tx.GetPortFacility(ctx, input.LandingPortID)
		if err != nil {
			return err
		}
		if source.Status != domain.PortFacilityActive || target.Status != domain.PortFacilityActive {
			return domain.ConflictError{Resource: "port_facility", Reason: "departure and landing ports must be active"}
		}
		if err := domain.ValidateRoute(source, target); err != nil {
			return err
		}
		businessDay, err := source.BusinessDay(input.DepartureWindowOpensAt)
		if err != nil {
			return err
		}
		count, err := tx.CountPortFacilityFishingVoyagesForBusinessDay(ctx, source.ID, businessDay)
		if err != nil {
			return err
		}
		if count >= source.DailyLimit {
			return domain.ConflictError{Resource: "port_facility", Reason: "daily departure limit reached"}
		}
		support_fleet, err := tx.GetSupportFleet(ctx, input.SupportFleetID)
		if err != nil {
			return err
		}
		if err := support_fleet.EligibleFor(input.DepartureWindowOpensAt, 1); err != nil {
			return err
		}
		now := s.clock.Now()
		run = domain.FishingVoyage{ID: identity.New("mission"), FishingPermitID: input.FishingPermitID, DeparturePortID: input.DeparturePortID,
			LandingPortID: input.LandingPortID, SupportFleetID: input.SupportFleetID, VoyageCode: strings.TrimSpace(input.VoyageCode),
			State: domain.FishingVoyagePlanned, DepartureWindowOpensAt: input.DepartureWindowOpensAt.UTC(), LandingDeadlineAt: input.LandingDeadlineAt.UTC(), Version: 1, CreatedAt: now, UpdatedAt: now}
		volume := 0
		batches := make([]domain.FishingVessel, 0, len(input.FishingVesselIDs))
		seen := make(map[string]struct{}, len(input.FishingVesselIDs))
		for _, batchID := range input.FishingVesselIDs {
			if _, exists := seen[batchID]; exists {
				return domain.ConflictError{Resource: "fishing_vessel", Reason: "duplicate fishing vessel in voyage"}
			}
			seen[batchID] = struct{}{}
			batch, err := tx.GetFishingVessel(ctx, batchID)
			if err != nil {
				return err
			}
			if batch.FishingPermitID != program.ID || batch.DeparturePortID != source.ID {
				return domain.ConflictError{Resource: "fishing_vessel", Reason: "vessel belongs to another program or departure site"}
			}
			if err := batch.Transition(domain.FishingVesselAssigned, now); err != nil {
				return err
			}
			volume += batch.HoldCapacityKg
			batches = append(batches, batch)
		}
		if err := (domain.VoyageWindow{StartAt: input.DepartureWindowOpensAt.UTC(), FinishAt: input.LandingDeadlineAt.UTC()}).Validate(program, batches, now); err != nil {
			return err
		}
		run.TotalHoldCapacityKg = volume
		if err := support_fleet.EligibleFor(input.DepartureWindowOpensAt, volume); err != nil {
			return err
		}
		if err := run.Validate(); err != nil {
			return err
		}
		if err := tx.InsertFishingVoyage(ctx, run); err != nil {
			return err
		}
		for _, batch := range batches {
			batch.FishingVoyageID = run.ID
			if err := tx.UpdateFishingVessel(ctx, batch, batch.Version); err != nil {
				return err
			}
			if err := tx.InsertFishingVoyageVesselLink(ctx, domain.FishingVoyageVesselLink{FishingVoyageID: run.ID, FishingVesselID: batch.ID, AddedAt: now}); err != nil {
				return err
			}
		}
		support_fleet.State = domain.SupportFleetAssigned
		support_fleet.AssignedVoyageID = run.ID
		support_fleet.UpdatedAt = now
		if err := tx.UpdateSupportFleet(ctx, support_fleet, support_fleet.Version); err != nil {
			return err
		}
		body, err := idempotency.Encode(run)
		if err != nil {
			return err
		}
		if err := tx.PutIdempotency(ctx, repository.IdempotencyRecord{Scope: "plan-mission", Key: input.IdempotencyKey, RequestHash: hash, ResponseCode: 201, ResponseBody: body, ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now}); err != nil {
			return err
		}
		if err := tx.InsertJob(ctx, domain.OutboxJob{ID: identity.New("job"), Kind: "fishing_voyage_planned", AggregateID: run.ID, Payload: body, Status: domain.JobPending, MaxAttempts: 5, AvailableAt: now, CreatedAt: now, UpdatedAt: now}); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, "fishing_voyage_planned", "fishing_voyage", run.ID, "success", map[string]string{"fishing_vessel_count": fmt.Sprint(len(batches))})
	})
	return run, err
}

func (s *VoyageService) ClearFishingVoyage(ctx context.Context, requestID string) (domain.FishingVoyage, error) {
	return s.transition(ctx, requestID, domain.FishingVoyageCleared, domain.RoleVoyageCoordinator, "fishing_voyage_cleared")
}

func (s *VoyageService) DepartureFishingVoyage(ctx context.Context, requestID string) (domain.FishingVoyage, error) {
	return s.transitionAny(ctx, requestID, domain.FishingVoyageDeparted, []domain.Role{domain.RoleVesselCaptain, domain.RoleVoyageCoordinator}, "fishing_voyage_started")
}

func (s *VoyageService) ConfirmFishingVoyageLanding(ctx context.Context, requestID string) (domain.FishingVoyage, error) {
	return s.transitionAny(ctx, requestID, domain.FishingVoyageLanded, []domain.Role{domain.RoleVesselCaptain, domain.RoleVoyageCoordinator}, "fishing_voyage_recovered")
}

func (s *VoyageService) CloseFishingVoyage(ctx context.Context, requestID string) (domain.FishingVoyage, error) {
	return s.transitionAny(ctx, requestID, domain.FishingVoyageClosed, []domain.Role{domain.RoleVoyageCoordinator}, "fishing_voyage_closed")
}

func (s *VoyageService) CancelFishingVoyage(ctx context.Context, requestID string, note string) (domain.FishingVoyage, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleVoyageCoordinator); err != nil {
		return domain.FishingVoyage{}, err
	}
	var result domain.FishingVoyage
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		run, err := tx.GetFishingVoyage(ctx, requestID)
		if err != nil {
			return err
		}
		if run.State != domain.FishingVoyagePlanned && run.State != domain.FishingVoyageCleared {
			return domain.TransitionError{Entity: "fishing_voyage", From: string(run.State), To: string(domain.FishingVoyageCancelled)}
		}
		now := s.clock.Now()
		items, err := tx.ListFishingVoyageVessels(ctx, run.ID)
		if err != nil {
			return err
		}
		if err := run.Transition(domain.FishingVoyageCancelled, now); err != nil {
			return err
		}
		for _, batch := range items {
			if err := batch.Transition(domain.FishingVesselSeaReady, now); err != nil {
				return err
			}
			batch.FishingVoyageID = ""
			if err := tx.UpdateFishingVessel(ctx, batch, batch.Version); err != nil {
				return err
			}
		}
		support_fleet, err := tx.GetSupportFleet(ctx, run.SupportFleetID)
		if err != nil {
			return err
		}
		support_fleet.State = domain.SupportFleetStandby
		support_fleet.AssignedVoyageID = ""
		support_fleet.UpdatedAt = now
		if err := tx.UpdateSupportFleet(ctx, support_fleet, support_fleet.Version); err != nil {
			return err
		}
		if err := tx.UpdateFishingVoyage(ctx, run, run.Version); err != nil {
			return err
		}
		result = run
		return s.audit.Record(ctx, tx, "fishing_voyage_cancelled", "fishing_voyage", run.ID, "success", map[string]string{"note": strings.TrimSpace(note)})
	})
	return result, err
}

func (s *VoyageService) transition(ctx context.Context, requestID string, target domain.FishingVoyageState, role domain.Role, action string) (domain.FishingVoyage, error) {
	return s.transitionAny(ctx, requestID, target, []domain.Role{role}, action)
}

func (s *VoyageService) transitionAny(ctx context.Context, requestID string, target domain.FishingVoyageState, roles []domain.Role, action string) (domain.FishingVoyage, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, roles...); err != nil {
		return domain.FishingVoyage{}, err
	}
	var result domain.FishingVoyage
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		run, err := tx.GetFishingVoyage(ctx, requestID)
		if err != nil {
			return err
		}
		if target == domain.FishingVoyageClosed {
			if _, err := tx.GetPendingCatchLanding(ctx, run.ID); err == nil {
				return domain.ConflictError{Resource: "catch_landing_task", Reason: "pending handoff review blocks voyage closure"}
			} else if !errors.Is(err, domain.ErrNotFound) {
				return err
			}
			if _, err := tx.GetActiveCatchAnomaly(ctx, run.ID); err == nil {
				return domain.ConflictError{Resource: "catch_anomaly", Reason: "active catch anomaly blocks voyage closure"}
			} else if !errors.Is(err, domain.ErrNotFound) {
				return err
			}
		}
		if err := run.Transition(target, s.clock.Now()); err != nil {
			return err
		}
		now := s.clock.Now()
		items, err := tx.ListFishingVoyageVessels(ctx, run.ID)
		if err != nil {
			return err
		}
		for _, batch := range items {
			switch target {
			case domain.FishingVoyageDeparted:
				if err := batch.Transition(domain.FishingVesselAtSea, now); err != nil {
					return err
				}
			case domain.FishingVoyageLanded:
				if batch.State != domain.FishingVesselQuarantined && batch.State != domain.FishingVesselRetired && batch.State != domain.FishingVesselReinspected {
					if err := batch.Transition(domain.FishingVesselLanded, now); err != nil {
						return err
					}
				}
			case domain.FishingVoyageClosed:
				if batch.State != domain.FishingVesselReinspected && batch.State != domain.FishingVesselRetired && batch.State != domain.FishingVesselLanded {
					return domain.ConflictError{Resource: "fishing_vessel", Reason: "all vessels must be landed, reinspected, retired, or quarantined before voyage closure"}
				}
			}
			if target == domain.FishingVoyageDeparted || target == domain.FishingVoyageLanded {
				if err := tx.UpdateFishingVessel(ctx, batch, batch.Version); err != nil {
					return err
				}
			}
		}
		support_fleet, err := tx.GetSupportFleet(ctx, run.SupportFleetID)
		if err != nil {
			return err
		}
		switch target {
		case domain.FishingVoyageDeparted:
			support_fleet.State = domain.SupportFleetDeployed
		case domain.FishingVoyageClosed, domain.FishingVoyageCancelled:
			support_fleet.State = domain.SupportFleetStandby
			support_fleet.AssignedVoyageID = ""
		}
		support_fleet.UpdatedAt = now
		if err := tx.UpdateSupportFleet(ctx, support_fleet, support_fleet.Version); err != nil {
			return err
		}
		if err := tx.UpdateFishingVoyage(ctx, run, run.Version); err != nil {
			return err
		}
		result = run
		return s.audit.Record(ctx, tx, action, "fishing_voyage", run.ID, "success", nil)
	})
	return result, err
}
