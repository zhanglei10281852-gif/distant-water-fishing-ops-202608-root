package service

import (
	"context"
	"errors"

	"github.com/zhanglei10281852-gif/distant-water-fishing-ops-202608-root/internal/domain"
)

type BulkFishingVesselItem struct {
	Index         int                   `json:"index"`
	FishingVessel *domain.FishingVessel `json:"fishing_vessel,omitempty"`
	Error         string                `json:"error,omitempty"`
	Code          string                `json:"code"`
}

type BulkFishingVesselResult struct {
	Items     []BulkFishingVesselItem `json:"items"`
	Succeeded int                     `json:"succeeded"`
	Failed    int                     `json:"failed"`
}

func (s *CatalogService) BulkRegisterFishingVessels(ctx context.Context, stages []domain.FishingVessel) (BulkFishingVesselResult, error) {
	principal, _ := principalOrEmpty(ctx)
	if err := requireRole(principal, domain.RoleVoyageCoordinator); err != nil {
		return BulkFishingVesselResult{}, err
	}
	if len(stages) == 0 {
		return BulkFishingVesselResult{}, domain.FieldError{Field: "fishing_vessels", Message: "at least one fishing vessel is required"}
	}
	if len(stages) > 100 {
		return BulkFishingVesselResult{}, domain.FieldError{Field: "fishing_vessels", Message: "cannot contain more than 100 items"}
	}
	result := BulkFishingVesselResult{Items: make([]BulkFishingVesselItem, 0, len(stages))}
	for index, input := range stages {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		created, err := s.RegisterFishingVessel(ctx, input.Clone())
		if err != nil {
			result.Failed++
			result.Items = append(result.Items, BulkFishingVesselItem{Index: index, Error: err.Error(), Code: classifyBulkError(err)})
			continue
		}
		result.Succeeded++
		createdCopy := created.Clone()
		result.Items = append(result.Items, BulkFishingVesselItem{Index: index, FishingVessel: &createdCopy, Code: "created"})
	}
	return result, nil
}

func classifyBulkError(err error) string {
	switch {
	case errors.Is(err, domain.ErrValidation):
		return "invalid"
	case errors.Is(err, domain.ErrConflict):
		return "conflict"
	default:
		return "failed"
	}
}
