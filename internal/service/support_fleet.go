package service

import (
	"context"
	"strings"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/repository"
)

type SupportFleetService struct{ dependencies }

func (s *SupportFleetService) StartMaintenance(ctx context.Context, support_fleetID string) (domain.SupportFleet, error) {
	return s.change(ctx, support_fleetID, "support_fleet_maintenance_started", func(support_fleet *domain.SupportFleet) error {
		return support_fleet.StartMaintenance(s.clock.Now())
	})
}

func (s *SupportFleetService) CompleteMaintenance(ctx context.Context, support_fleetID string) (domain.SupportFleet, error) {
	return s.change(ctx, support_fleetID, "support_fleet_maintenance_recovered", func(support_fleet *domain.SupportFleet) error {
		return support_fleet.CompleteMaintenance(s.clock.Now())
	})
}

func (s *SupportFleetService) Retire(ctx context.Context, support_fleetID, reason string) (domain.SupportFleet, error) {
	if strings.TrimSpace(reason) == "" {
		return domain.SupportFleet{}, domain.FieldError{Field: "reason", Message: "is required"}
	}
	return s.changeWithMetadata(ctx, support_fleetID, "support_fleet_retired", map[string]string{"reason": strings.TrimSpace(reason)}, func(support_fleet *domain.SupportFleet) error {
		return support_fleet.Retire(s.clock.Now())
	})
}

func (s *SupportFleetService) change(ctx context.Context, support_fleetID, action string, mutate func(*domain.SupportFleet) error) (domain.SupportFleet, error) {
	return s.changeWithMetadata(ctx, support_fleetID, action, nil, mutate)
}

func (s *SupportFleetService) changeWithMetadata(ctx context.Context, support_fleetID, action string, metadata map[string]string, mutate func(*domain.SupportFleet) error) (domain.SupportFleet, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleVoyageCoordinator); err != nil {
		return domain.SupportFleet{}, err
	}
	var result domain.SupportFleet
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		support_fleet, err := tx.GetSupportFleet(ctx, support_fleetID)
		if err != nil {
			return err
		}
		before := support_fleet.Version
		if err := mutate(&support_fleet); err != nil {
			return err
		}
		if err := tx.UpdateSupportFleet(ctx, support_fleet, before); err != nil {
			return err
		}
		result = support_fleet
		return s.audit.Record(ctx, tx, action, "support_fleet", support_fleet.ID, "success", metadata)
	})
	return result, err
}
