package domain

import (
	"strings"
	"time"
)

type CatchLandingTaskStatus string

const (
	CatchLandingTaskPending  CatchLandingTaskStatus = "pending"
	CatchLandingTaskApproved CatchLandingTaskStatus = "approved"
	CatchLandingTaskDenied   CatchLandingTaskStatus = "denied"
	CatchLandingTaskExpired  CatchLandingTaskStatus = "expired"
)

func (s CatchLandingTaskStatus) IsResolved() bool {
	return s == CatchLandingTaskApproved || s == CatchLandingTaskDenied || s == CatchLandingTaskExpired
}

func (s CatchLandingTaskStatus) IsPending() bool { return s == CatchLandingTaskPending }

type CatchLandingTask struct {
	ID                 string                 `json:"id"`
	FishingVoyageID    string                 `json:"fishing_voyage_id"`
	CoordinatorID      string                 `json:"coordinator_id"`
	FisheriesOfficerID string                 `json:"fisheries_officer_id"`
	LandingStation     string                 `json:"landing_station"`
	Status             CatchLandingTaskStatus `json:"status"`
	ExpiresAt          time.Time              `json:"expires_at"`
	ResolvedAt         *time.Time             `json:"resolved_at,omitempty"`
	ResolutionNote     string                 `json:"resolution_note,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
	Version            int64                  `json:"version"`
}

func (h CatchLandingTask) Validate() error {
	if strings.TrimSpace(h.FishingVoyageID) == "" || strings.TrimSpace(h.CoordinatorID) == "" || strings.TrimSpace(h.FisheriesOfficerID) == "" {
		return FieldError{Field: "catch_landing_task", Message: "voyage, coordinator and fisheries officer are required"}
	}
	if h.CoordinatorID == h.FisheriesOfficerID {
		return FieldError{Field: "fisheries_officer_id", Message: "must differ from coordinator"}
	}
	if strings.TrimSpace(h.LandingStation) == "" || h.ExpiresAt.IsZero() {
		return FieldError{Field: "catch_landing_task", Message: "landing_station and expiry are required"}
	}
	switch h.Status {
	case CatchLandingTaskPending, CatchLandingTaskApproved, CatchLandingTaskDenied, CatchLandingTaskExpired:
		return nil
	default:
		return FieldError{Field: "catch_landing_task_status", Message: "is invalid"}
	}
}

func (h *CatchLandingTask) Resolve(status CatchLandingTaskStatus, note string, now time.Time) error {
	if h.Status != CatchLandingTaskPending {
		return TransitionError{Entity: "catch_landing_task", From: string(h.Status), To: string(status)}
	}
	if now.After(h.ExpiresAt) && status != CatchLandingTaskExpired {
		return ErrExpired
	}
	if status != CatchLandingTaskApproved && status != CatchLandingTaskDenied && status != CatchLandingTaskExpired {
		return FieldError{Field: "catch_landing_task_status", Message: "unsupported resolution"}
	}
	now = now.UTC()
	h.Status = status
	h.ResolutionNote = strings.TrimSpace(note)
	h.ResolvedAt = &now
	h.UpdatedAt = now
	return nil
}
