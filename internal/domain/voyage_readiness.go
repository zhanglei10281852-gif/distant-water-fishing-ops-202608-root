package domain

import "time"

type VoyageReadiness struct {
	FishingVoyageID               string             `json:"fishing_voyage_id"`
	FishingVoyageState            FishingVoyageState `json:"fishing_voyage_state"`
	ExpectedFishingVesselCount    int                `json:"expected_fishing_vessel_count"`
	LandedFishingVesselCount      int                `json:"landed_fishing_vessel_count"`
	ReinspectedFishingVesselCount int                `json:"reinspected_fishing_vessel_count"`
	RetiredFishingVesselCount     int                `json:"retired_fishing_vessel_count"`
	QuarantinedCount              int                `json:"quarantined_count"`
	PendingCatchLanding           bool               `json:"pending_catch_landing"`
	OpenCatchAnomaly              bool               `json:"open_catch_anomaly"`
	LastDeclarationAt             *time.Time         `json:"last_declaration_at,omitempty"`
	Complete                      bool               `json:"complete"`
	Blockers                      []string           `json:"blockers"`
}

func (r VoyageReadiness) Clone() VoyageReadiness {
	clone := r
	clone.Blockers = append([]string(nil), r.Blockers...)
	if r.LastDeclarationAt != nil {
		value := *r.LastDeclarationAt
		clone.LastDeclarationAt = &value
	}
	return clone
}

func (r *VoyageReadiness) Evaluate() {
	r.Blockers = r.Blockers[:0]
	if r.PendingCatchLanding {
		r.Blockers = append(r.Blockers, "pending handoff task")
	}
	if r.OpenCatchAnomaly {
		r.Blockers = append(r.Blockers, "open catch anomaly")
	}
	if r.QuarantinedCount > 0 {
		r.Blockers = append(r.Blockers, "quarantined fishing vessels require review")
	}
	if r.ExpectedFishingVesselCount == 0 {
		r.Blockers = append(r.Blockers, "voyage has no fishing vessels")
	}
	if r.LandedFishingVesselCount < r.ExpectedFishingVesselCount && r.FishingVoyageState == FishingVoyageLanded {
		r.Blockers = append(r.Blockers, "not all fishing vessels reached a post-landing state")
	}
	r.Complete = len(r.Blockers) == 0 && (r.FishingVoyageState == FishingVoyageClosed || r.FishingVoyageState == FishingVoyageLanded)
}
