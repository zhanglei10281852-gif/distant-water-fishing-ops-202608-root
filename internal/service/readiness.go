package service

import (
	"context"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/repository"
)

func (s *QueryService) GetFishingVoyageReadiness(ctx context.Context, requestID string) (domain.VoyageReadiness, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleVoyageCoordinator, domain.RoleVesselCaptain, domain.RoleFisheriesOfficer, domain.RoleComplianceAuditor); err != nil {
		return domain.VoyageReadiness{}, err
	}
	var report domain.VoyageReadiness
	err := s.store.Read(ctx, func(reader repository.Reader) error {
		var err error
		report, err = reader.GetVoyageReadiness(ctx, requestID)
		return err
	})
	return report, err
}
