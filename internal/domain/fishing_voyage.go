package domain

import (
	"strings"
	"time"
)

type FishingVoyageState string

const (
	FishingVoyagePlanned   FishingVoyageState = "planned"
	FishingVoyageCleared   FishingVoyageState = "cleared"
	FishingVoyageDeparted  FishingVoyageState = "at_sea"
	FishingVoyageLanded    FishingVoyageState = "landed"
	FishingVoyageClosed    FishingVoyageState = "closed"
	FishingVoyageCancelled FishingVoyageState = "cancelled"
)

type FishingVoyage struct {
	ID                     string             `json:"id"`
	FishingPermitID        string             `json:"fishing_permit_id"`
	DeparturePortID        string             `json:"departure_port_id"`
	LandingPortID          string             `json:"landing_port_id"`
	SupportFleetID         string             `json:"support_fleet_id"`
	VoyageCode             string             `json:"voyage_code"`
	State                  FishingVoyageState `json:"state"`
	DepartureWindowOpensAt time.Time          `json:"departure_window_opens_at"`
	LandingDeadlineAt      time.Time          `json:"landing_deadline_at"`
	DepartedAt             *time.Time         `json:"departed_at,omitempty"`
	LandedAt               *time.Time         `json:"landed_at,omitempty"`
	ClosedAt               *time.Time         `json:"closed_at,omitempty"`
	TotalHoldCapacityKg    int                `json:"total_hold_capacity_kg"`
	CreatedAt              time.Time          `json:"created_at"`
	UpdatedAt              time.Time          `json:"updated_at"`
	Version                int64              `json:"version"`
}

type FishingVoyageVesselLink struct {
	FishingVoyageID string
	FishingVesselID string
	AddedAt         time.Time
}

func (s FishingVoyage) Validate() error {
	if strings.TrimSpace(s.FishingPermitID) == "" || strings.TrimSpace(s.DeparturePortID) == "" || strings.TrimSpace(s.LandingPortID) == "" {
		return FieldError{Field: "fishing_voyage", Message: "permit, departure port and landing port are required"}
	}
	if s.DeparturePortID == s.LandingPortID {
		return FieldError{Field: "landing_port_id", Message: "must differ from departure port"}
	}
	if strings.TrimSpace(s.VoyageCode) == "" || strings.TrimSpace(s.SupportFleetID) == "" {
		return FieldError{Field: "fishing_voyage", Message: "voyage code and support fleet are required"}
	}
	if !s.LandingDeadlineAt.After(s.DepartureWindowOpensAt) {
		return FieldError{Field: "landing_deadline_at", Message: "must be after departure window opening"}
	}
	if s.TotalHoldCapacityKg < 1 {
		return FieldError{Field: "total_hold_capacity_kg", Message: "must be positive"}
	}
	return validateFishingVoyageState(s.State)
}

func validateFishingVoyageState(state FishingVoyageState) error {
	switch state {
	case FishingVoyagePlanned, FishingVoyageCleared, FishingVoyageDeparted, FishingVoyageLanded, FishingVoyageClosed, FishingVoyageCancelled:
		return nil
	default:
		return FieldError{Field: "fishing_voyage_state", Message: "is invalid"}
	}
}

func (s FishingVoyageState) IsTerminal() bool {
	return s == FishingVoyageClosed || s == FishingVoyageCancelled
}

func (s *FishingVoyage) Transition(to FishingVoyageState, now time.Time) error {
	allowed := map[FishingVoyageState]map[FishingVoyageState]bool{
		FishingVoyagePlanned:  {FishingVoyageCleared: true, FishingVoyageCancelled: true},
		FishingVoyageCleared:  {FishingVoyageDeparted: true, FishingVoyageCancelled: true},
		FishingVoyageDeparted: {FishingVoyageLanded: true},
		FishingVoyageLanded:   {FishingVoyageClosed: true},
	}
	if !allowed[s.State][to] {
		return TransitionError{Entity: "fishing_voyage", From: string(s.State), To: string(to)}
	}
	now = now.UTC()
	switch to {
	case FishingVoyageDeparted:
		s.DepartedAt = &now
	case FishingVoyageLanded:
		s.LandedAt = &now
	case FishingVoyageClosed:
		s.ClosedAt = &now
	}
	s.State = to
	s.UpdatedAt = now
	return nil
}
