package service

import (
	"context"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/identity"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/repository"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/requestmeta"
)

type CatalogService struct{ dependencies }

func (s *CatalogService) CreateFishingPermit(ctx context.Context, program domain.FishingPermit) (domain.FishingPermit, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleVoyageCoordinator); err != nil {
		return domain.FishingPermit{}, err
	}
	now := s.clock.Now()
	program.ID = identity.New("program")
	program.Code = domain.NormalizeCode(program.Code)
	if err := domain.ValidateBusinessCode("code", program.Code); err != nil {
		return domain.FishingPermit{}, err
	}
	program.Status = domain.FishingPermitDraft
	program.Version = 1
	program.CreatedAt, program.UpdatedAt = now, now
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		if err := tx.InsertFishingPermit(ctx, program); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, "fishing_permit_created", "fishing_permit", program.ID, "success", nil)
	})
	return program, err
}

func (s *CatalogService) ActivateFishingPermit(ctx context.Context, fishingPermitID string) (domain.FishingPermit, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleVoyageCoordinator); err != nil {
		return domain.FishingPermit{}, err
	}
	var result domain.FishingPermit
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		program, err := tx.GetFishingPermit(ctx, fishingPermitID)
		if err != nil {
			return err
		}
		if program.Status != domain.FishingPermitDraft {
			return domain.TransitionError{Entity: "program", From: string(program.Status), To: string(domain.FishingPermitActive)}
		}
		before := program.Version
		program.Status = domain.FishingPermitActive
		program.UpdatedAt = s.clock.Now()
		if err := tx.UpdateFishingPermit(ctx, program, before); err != nil {
			return err
		}
		result = program
		return s.audit.Record(ctx, tx, "fishing_permit_activated", "fishing_permit", program.ID, "success", nil)
	})
	return result, err
}

func (s *CatalogService) CreatePortFacility(ctx context.Context, port_facility domain.PortFacility) (domain.PortFacility, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleVoyageCoordinator); err != nil {
		return domain.PortFacility{}, err
	}
	now := s.clock.Now()
	port_facility.ID = identity.New("port_facility")
	port_facility.Code = domain.NormalizeCode(port_facility.Code)
	if err := domain.ValidateBusinessCode("code", port_facility.Code); err != nil {
		return domain.PortFacility{}, err
	}
	port_facility.Status = domain.PortFacilityActive
	port_facility.Version = 1
	port_facility.CreatedAt, port_facility.UpdatedAt = now, now
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		if err := tx.InsertPortFacility(ctx, port_facility); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, "port_facility_created", "port_facility", port_facility.ID, "success", nil)
	})
	return port_facility, err
}

func (s *CatalogService) CreateSupportFleet(ctx context.Context, support_fleet domain.SupportFleet) (domain.SupportFleet, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleVoyageCoordinator); err != nil {
		return domain.SupportFleet{}, err
	}
	now := s.clock.Now()
	support_fleet.ID = identity.New("fleet")
	support_fleet.FleetCode = domain.NormalizeCode(support_fleet.FleetCode)
	if err := domain.ValidateBusinessCode("fleet_code", support_fleet.FleetCode); err != nil {
		return domain.SupportFleet{}, err
	}
	support_fleet.State = domain.SupportFleetStandby
	support_fleet.Version = 1
	support_fleet.CreatedAt, support_fleet.UpdatedAt = now, now
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		if err := tx.InsertSupportFleet(ctx, support_fleet); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, "support_fleet_created", "support_fleet", support_fleet.ID, "success", nil)
	})
	return support_fleet, err
}

func (s *CatalogService) RegisterFishingVessel(ctx context.Context, batch domain.FishingVessel) (domain.FishingVessel, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleVoyageCoordinator); err != nil {
		return domain.FishingVessel{}, err
	}
	now := s.clock.Now()
	batch.ID = identity.New("stage")
	batch.State = domain.FishingVesselBerthed
	batch.Version = 1
	batch.CreatedAt, batch.UpdatedAt = now, now
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		if err := tx.InsertFishingVessel(ctx, batch); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, "fishing_vessel_registered", "fishing_vessel", batch.ID, "success", nil)
	})
	return batch, err
}

func (s *CatalogService) VerifyFishingVessel(ctx context.Context, fishingVesselID string) (domain.FishingVessel, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleVoyageCoordinator); err != nil {
		return domain.FishingVessel{}, err
	}
	var result domain.FishingVessel
	err := s.store.WithTx(ctx, func(tx repository.Tx) error {
		batch, err := tx.GetFishingVessel(ctx, fishingVesselID)
		if err != nil {
			return err
		}
		if err := batch.Transition(domain.FishingVesselSeaReady, s.clock.Now()); err != nil {
			return err
		}
		if err := tx.UpdateFishingVessel(ctx, batch, batch.Version); err != nil {
			return err
		}
		result = batch
		return s.audit.Record(ctx, tx, "fishing_vessel_verified", "fishing_vessel", batch.ID, "success", nil)
	})
	return result, err
}

func principalOrEmpty(ctx context.Context) (domain.Principal, bool) {
	principal, ok := requestmeta.Principal(ctx)
	return principal, ok
}
