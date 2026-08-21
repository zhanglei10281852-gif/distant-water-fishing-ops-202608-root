package domain

import (
	"strings"
	"time"
)

type FishingVesselState string

const (
	FishingVesselBerthed     FishingVesselState = "berthed"
	FishingVesselSeaReady    FishingVesselState = "sea_ready"
	FishingVesselAssigned    FishingVesselState = "assigned"
	FishingVesselAtSea       FishingVesselState = "at_sea"
	FishingVesselLanded      FishingVesselState = "landed"
	FishingVesselQuarantined FishingVesselState = "quarantined"
	FishingVesselReinspected FishingVesselState = "reinspected"
	FishingVesselRetired     FishingVesselState = "retired"
)

type FishingVessel struct {
	ID              string             `json:"id"`
	FishingPermitID string             `json:"fishing_permit_id"`
	DeparturePortID string             `json:"departure_port_id"`
	RegistryNumber  string             `json:"registry_number"`
	VesselClass     string             `json:"vessel_class"`
	VoyageCount     int                `json:"voyage_count"`
	HoldCapacityKg  int                `json:"hold_capacity_kg"`
	State           FishingVesselState `json:"state"`
	ExpiresAt       time.Time          `json:"expires_at"`
	FishingVoyageID string             `json:"fishing_voyage_id,omitempty"`
	QuarantineNote  string             `json:"quarantine_note,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
	Version         int64              `json:"version"`
}

func (b FishingVessel) Validate() error {
	if strings.TrimSpace(b.FishingPermitID) == "" || strings.TrimSpace(b.DeparturePortID) == "" {
		return FieldError{Field: "fishing_vessel", Message: "permit and departure port are required"}
	}
	if strings.TrimSpace(b.RegistryNumber) == "" || strings.TrimSpace(b.VesselClass) == "" {
		return FieldError{Field: "fishing_vessel", Message: "registry number and vessel class are required"}
	}
	if b.VoyageCount < 1 || b.HoldCapacityKg < 1 {
		return FieldError{Field: "fishing_vessel", Message: "voyage count and hold capacity must be positive"}
	}
	if b.ExpiresAt.IsZero() {
		return FieldError{Field: "expires_at", Message: "is required"}
	}
	return validateFishingVesselState(b.State)
}

func validateFishingVesselState(state FishingVesselState) error {
	switch state {
	case FishingVesselBerthed, FishingVesselSeaReady, FishingVesselAssigned, FishingVesselAtSea, FishingVesselLanded, FishingVesselQuarantined, FishingVesselReinspected, FishingVesselRetired:
		return nil
	default:
		return FieldError{Field: "fishing_vessel_state", Message: "is invalid"}
	}
}

func (s FishingVesselState) IsTerminal() bool {
	return s == FishingVesselReinspected || s == FishingVesselRetired
}

func (b *FishingVessel) Transition(to FishingVesselState, now time.Time) error {
	allowed := map[FishingVesselState]map[FishingVesselState]bool{
		FishingVesselBerthed:     {FishingVesselSeaReady: true, FishingVesselRetired: true},
		FishingVesselSeaReady:    {FishingVesselAssigned: true, FishingVesselRetired: true},
		FishingVesselAssigned:    {FishingVesselSeaReady: true, FishingVesselAtSea: true},
		FishingVesselAtSea:       {FishingVesselLanded: true, FishingVesselQuarantined: true},
		FishingVesselLanded:      {FishingVesselReinspected: true, FishingVesselQuarantined: true},
		FishingVesselQuarantined: {FishingVesselReinspected: true, FishingVesselRetired: true},
	}
	if !allowed[b.State][to] {
		return TransitionError{Entity: "fishing_vessel", From: string(b.State), To: string(to)}
	}
	if !b.IsUsableAt(now) && to != FishingVesselRetired && to != FishingVesselQuarantined {
		return ConflictError{Resource: "fishing_vessel", Reason: "vessel certificate expired before transition"}
	}
	b.State = to
	b.UpdatedAt = now.UTC()
	return nil
}

func (b FishingVessel) Clone() FishingVessel { return b }

func (b FishingVessel) IsUsableAt(at time.Time) bool {
	return b.ExpiresAt.After(at) && b.State != FishingVesselRetired
}
