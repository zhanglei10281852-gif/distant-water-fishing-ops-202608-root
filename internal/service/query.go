package service

import (
	"context"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/repository"
)

type QueryService struct{ dependencies }

func (s *QueryService) PlatformSummary(ctx context.Context) (repository.PlatformSummary, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireAction(principal, domain.ActionReadOperations); err != nil {
		return repository.PlatformSummary{}, err
	}
	var summary repository.PlatformSummary
	err := s.store.Read(ctx, func(reader repository.Reader) error {
		var err error
		summary, err = reader.GetPlatformSummary(ctx)
		return err
	})
	return summary, err
}

func (s *QueryService) FishingVoyage(ctx context.Context, id string) (domain.FishingVoyage, []domain.FishingVessel, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleVoyageCoordinator, domain.RoleVesselCaptain, domain.RoleFisheriesOfficer, domain.RoleComplianceAuditor); err != nil {
		return domain.FishingVoyage{}, nil, err
	}
	var run domain.FishingVoyage
	var items []domain.FishingVessel
	err := s.store.Read(ctx, func(reader repository.Reader) error {
		var err error
		run, err = reader.GetFishingVoyage(ctx, id)
		if err != nil {
			return err
		}
		items, err = reader.ListFishingVoyageVessels(ctx, id)
		return err
	})
	return run, items, err
}

func (s *QueryService) FishingVoyages(ctx context.Context, filter repository.FishingVoyageFilter) (repository.FishingVoyagePage, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleVoyageCoordinator, domain.RoleVesselCaptain, domain.RoleFisheriesOfficer, domain.RoleComplianceAuditor); err != nil {
		return repository.FishingVoyagePage{}, err
	}
	var page repository.FishingVoyagePage
	err := s.store.Read(ctx, func(reader repository.Reader) error {
		var err error
		page, err = reader.ListFishingVoyages(ctx, filter)
		return err
	})
	return page, err
}

func (s *QueryService) FishingVessels(ctx context.Context, filter repository.FishingVesselFilter) (repository.FishingVesselPage, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleVoyageCoordinator, domain.RoleVesselCaptain, domain.RoleFisheriesOfficer, domain.RoleComplianceAuditor); err != nil {
		return repository.FishingVesselPage{}, err
	}
	var page repository.FishingVesselPage
	err := s.store.Read(ctx, func(reader repository.Reader) error {
		var err error
		page, err = reader.ListFishingVessels(ctx, filter)
		return err
	})
	return page, err
}

func (s *QueryService) LandingAnomalies(ctx context.Context, filter repository.CatchAnomalyFilter) (repository.CatchAnomalyPage, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleVoyageCoordinator, domain.RoleFisheriesOfficer, domain.RoleComplianceAuditor); err != nil {
		return repository.CatchAnomalyPage{}, err
	}
	var page repository.CatchAnomalyPage
	err := s.store.Read(ctx, func(reader repository.Reader) error {
		var err error
		page, err = reader.ListLandingAnomalies(ctx, filter)
		return err
	})
	return page, err
}

func (s *QueryService) Audit(ctx context.Context, filter repository.AuditFilter) (repository.AuditPage, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleComplianceAuditor, domain.RoleVoyageCoordinator); err != nil {
		return repository.AuditPage{}, err
	}
	var page repository.AuditPage
	err := s.store.Read(ctx, func(reader repository.Reader) error {
		var err error
		page, err = reader.ListAuditEvents(ctx, filter)
		return err
	})
	return page, err
}
