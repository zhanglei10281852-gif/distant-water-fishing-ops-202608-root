package service

import (
	"context"
	"strings"
	"time"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/identity"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/repository"
)

type HandoffService struct {
	dependencies
	catch_landing_taskTTL time.Duration
}

type CreateCatchLandingTaskInput struct {
	FishingVoyageID    string
	CoordinatorID      string
	FisheriesOfficerID string
	LandingStation     string
}

func (s *HandoffService) CreateCatchLandingTask(ctx context.Context, input CreateCatchLandingTaskInput) (domain.CatchLandingTask, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleVesselCaptain, domain.RoleVoyageCoordinator); err != nil {
		return domain.CatchLandingTask{}, err
	}
	if strings.TrimSpace(input.CoordinatorID) == "" {
		input.CoordinatorID = principal.UserID
	}
	var result domain.CatchLandingTask
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		run, err := tx.GetFishingVoyage(ctx, input.FishingVoyageID)
		if err != nil {
			return err
		}
		if run.State != domain.FishingVoyageDeparted && run.State != domain.FishingVoyageLanded {
			return domain.ConflictError{Resource: "fishing_voyage", Reason: "handoff review requires a departed or landed voyage"}
		}
		if _, err := tx.GetPendingCatchLanding(ctx, run.ID); err == nil {
			return domain.ConflictError{Resource: "catch_landing_task", Reason: "mission already has a pending handoff review"}
		} else if !isNotFound(err) {
			return err
		}
		now := s.clock.Now()
		catch_landing_task := domain.CatchLandingTask{ID: identity.New("catch_landing_task"), FishingVoyageID: run.ID, CoordinatorID: strings.TrimSpace(input.CoordinatorID), FisheriesOfficerID: strings.TrimSpace(input.FisheriesOfficerID), LandingStation: strings.TrimSpace(input.LandingStation), Status: domain.CatchLandingTaskPending, ExpiresAt: now.Add(s.catch_landing_taskTTL), Version: 1, CreatedAt: now, UpdatedAt: now}
		if err := catch_landing_task.Validate(); err != nil {
			return err
		}
		if err := tx.InsertCatchLandingTask(ctx, catch_landing_task); err != nil {
			return err
		}
		result = catch_landing_task
		return s.audit.Record(ctx, tx, "catch_landing_task_created", "catch_landing_task", catch_landing_task.ID, "success", nil)
	})
	return result, err
}

func (s *HandoffService) ResolveCatchLandingTask(ctx context.Context, catch_landing_taskID string, accepted bool, note string) (domain.CatchLandingTask, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleVesselCaptain, domain.RoleVoyageCoordinator); err != nil {
		return domain.CatchLandingTask{}, err
	}
	var result domain.CatchLandingTask
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		catch_landing_task, err := tx.GetCatchLandingTask(ctx, catch_landing_taskID)
		if err != nil {
			return err
		}
		if catch_landing_task.FisheriesOfficerID != principal.UserID && !principal.Can(domain.RoleVoyageCoordinator) {
			return domain.ConflictError{Resource: "catch_landing_task", Reason: "only the assigned fisheries officer may resolve it"}
		}
		status := domain.CatchLandingTaskDenied
		if accepted {
			status = domain.CatchLandingTaskApproved
		}
		if err := catch_landing_task.Resolve(status, note, s.clock.Now()); err != nil {
			return err
		}
		if err := tx.UpdateCatchLandingTask(ctx, catch_landing_task, catch_landing_task.Version); err != nil {
			return err
		}
		result = catch_landing_task
		return s.audit.Record(ctx, tx, "catch_landing_task_resolved", "catch_landing_task", catch_landing_task.ID, "success", map[string]string{"status": string(status)})
	})
	return result, err
}
