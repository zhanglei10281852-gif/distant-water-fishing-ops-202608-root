package domain

import (
	"strings"
	"time"
)

type VoyageWindow struct {
	StartAt  time.Time
	FinishAt time.Time
}

func (w VoyageWindow) Duration() time.Duration {
	return w.FinishAt.Sub(w.StartAt)
}

func (w VoyageWindow) Validate(program FishingPermit, stages []FishingVessel, now time.Time) error {
	if w.StartAt.IsZero() || w.FinishAt.IsZero() {
		return FieldError{Field: "mission_window", Message: "start and finish are required"}
	}
	if !w.FinishAt.After(w.StartAt) {
		return FieldError{Field: "finish_at", Message: "must be after start"}
	}
	if w.Duration() > program.MaxVoyageDuration {
		return ConflictError{Resource: "fishing_voyage", Reason: "voyage exceeds permit duration limit"}
	}
	if w.StartAt.Before(now.Add(-15 * time.Minute)) {
		return ConflictError{Resource: "fishing_voyage", Reason: "voyage window is already closed"}
	}
	for _, stage := range stages {
		if !stage.ExpiresAt.After(w.FinishAt) {
			return ConflictError{Resource: "fishing_vessel", Reason: "vessel certificate expires before landing deadline"}
		}
	}
	return nil
}

func ValidateRoute(source, target PortFacility) error {
	if strings.TrimSpace(source.ID) == "" || strings.TrimSpace(target.ID) == "" {
		return FieldError{Field: "route", Message: "departure and landing ports are required"}
	}
	if source.ID == target.ID {
		return FieldError{Field: "route", Message: "departure and landing ports must differ"}
	}
	if source.Status != PortFacilityActive || target.Status != PortFacilityActive {
		return ConflictError{Resource: "route", Reason: "both port facilitys must be active"}
	}
	return nil
}
